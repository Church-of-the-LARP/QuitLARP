package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
	"backend/models"
)

// StartAttemptInput is the request of POST /api/v1/assessments/{id}/attempts.
type StartAttemptInput struct {
	AssessmentID int64 `path:"id" example:"1" doc:"Assessment id to self-start"`
}

// GetAttemptInput is the request of GET /api/v1/attempts/{id}.
type GetAttemptInput struct {
	ID int64 `path:"id" example:"42" doc:"Attempt id"`
}

// AttemptSnapshotOutput is the response of GET /api/v1/attempts/{id}: a
// snapshot for page load or reconnect, flat (no wrapper key) per TASK.md.
type AttemptSnapshotOutput struct {
	Body struct {
		Status               models.AttemptStatus `json:"status" enum:"pending,running,ended" doc:"Lifecycle status"`
		ChapterID            *int64               `json:"chapterId" doc:"The chapter the candidate is currently on; null before starting"`
		TimeRemainingSeconds int                  `json:"timeRemainingSeconds" doc:"Seconds left on the clock"`
		ScheduledStartAt     *time.Time           `json:"scheduledStartAt" doc:"Set only for a scheduled/invited attempt whose clock hasn't started yet"`
	}
}

// LeaveAttemptInput is the request of POST /api/v1/attempts/{id}/leave.
type LeaveAttemptInput struct {
	ID int64 `path:"id" example:"42" doc:"Attempt id to leave"`
}

func (h *Handlers) registerAssessmentAttempts(api huma.API) {
	// POST /api/v1/assessments/{id}/attempts — any authenticated user,
	// solo self-start.
	huma.Register(api, huma.Operation{
		OperationID: "startAttempt",
		Method:      http.MethodPost,
		Path:        "/api/v1/assessments/{id}/attempts",
		Summary:     "Self-start a solo attempt",
		Description: "Any authenticated user may start a solo attempt: creates a pending attempt with inviteId null, using the assessment's own timeLimitMinutes. Deadline is connect-time + timeLimitMinutes, computed once the WebSocket connects.",
	}, func(ctx context.Context, input *StartAttemptInput) (*AttemptOutput, error) {
		user, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		assessment, err := database.GetAssessmentDetail(ctx, h.db, input.AssessmentID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "assessment not found")
			}
			log.Printf("assessment lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "something went wrong")
		}

		attempt, err := database.CreateAttempt(ctx, h.db, input.AssessmentID, user.ID, assessment.TimeLimitMinutes)
		if err != nil {
			if database.IsUniqueViolation(err) {
				return nil, conflict("you already have an active attempt at this assessment")
			}
			log.Printf("create attempt: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not start attempt")
		}
		resp := &AttemptOutput{}
		resp.Body.Attempt = attempt
		return resp, nil
	})

	// GET /api/v1/attempts/{id} — the attempt's owner only.
	huma.Register(api, huma.Operation{
		OperationID: "getAttempt",
		Method:      http.MethodGet,
		Path:        "/api/v1/attempts/{id}",
		Summary:     "Get an attempt snapshot",
		Description: "The attempt's owner only. A snapshot for page load or reconnect: status, current chapter, time remaining. If a Runner is live for this attempt, its in-memory state is the source of truth; otherwise this falls back to the database row.",
	}, func(ctx context.Context, input *GetAttemptInput) (*AttemptSnapshotOutput, error) {
		user, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		attempt, err := database.GetAttempt(ctx, h.db, input.ID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "attempt not found")
			}
			log.Printf("attempt lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "something went wrong")
		}
		if attempt.UserID != user.ID {
			return nil, huma.NewError(http.StatusForbidden, "you do not have permission to do this")
		}

		resp := &AttemptSnapshotOutput{}
		if runner, ok := h.attempts.Get(input.ID); ok {
			snap := runner.Snapshot()
			resp.Body.Status = snap.Status
			resp.Body.ChapterID = snap.ChapterID
			resp.Body.TimeRemainingSeconds = snap.TimeRemainingSeconds
			resp.Body.ScheduledStartAt = snap.ScheduledStartAt
			return resp, nil
		}

		// No live Runner: this attempt either never started or already
		// ended, so the database row is the whole truth.
		resp.Body.Status = attempt.Status
		resp.Body.ChapterID = attempt.CurrentChapterID
		switch attempt.Status {
		case models.AttemptRunning:
			if attempt.StartedAt != nil {
				deadline := attempt.StartedAt.Add(time.Duration(attempt.TimeLimitMinutes) * time.Minute)
				resp.Body.TimeRemainingSeconds = max(0, int(time.Until(deadline).Seconds()))
			}
		case models.AttemptPending:
			resp.Body.TimeRemainingSeconds = attempt.TimeLimitMinutes * 60
			if attempt.InviteID != nil {
				invite, err := database.GetInvite(ctx, h.db, *attempt.InviteID)
				if err != nil {
					log.Printf("invite lookup: %v", err)
					return nil, huma.NewError(http.StatusInternalServerError, "something went wrong")
				}
				resp.Body.ScheduledStartAt = &invite.ScheduledStartAt
			}
		}
		return resp, nil
	})

	// POST /api/v1/attempts/{id}/leave — the attempt's owner only.
	huma.Register(api, huma.Operation{
		OperationID: "leaveAttempt",
		Method:      http.MethodPost,
		Path:        "/api/v1/attempts/{id}/leave",
		Summary:     "Leave an attempt",
		Description: "The attempt's owner only. Marks the attempt abandoned. If a Runner is live, it's sent a leaveCmd so it ends cleanly and notifies the (possibly still connected) websocket with attempt.ended; otherwise the row is updated directly.",
	}, func(ctx context.Context, input *LeaveAttemptInput) (*AttemptOutput, error) {
		user, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		attempt, err := database.GetAttempt(ctx, h.db, input.ID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "attempt not found")
			}
			log.Printf("attempt lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "something went wrong")
		}
		if attempt.UserID != user.ID {
			return nil, huma.NewError(http.StatusForbidden, "you do not have permission to do this")
		}

		resp := &AttemptOutput{}
		if runner, ok := h.attempts.Get(input.ID); ok {
			runner.Leave()
			attempt, err = database.GetAttempt(ctx, h.db, input.ID)
			if err != nil {
				log.Printf("attempt lookup after leave: %v", err)
				return nil, huma.NewError(http.StatusInternalServerError, "something went wrong")
			}
			resp.Body.Attempt = attempt
			return resp, nil
		}

		// No live Runner: a never-started (pending) attempt is marked
		// ended in place rather than deleted, so it still shows up as a
		// short attempt history entry — same as any other leave.
		attempt, err = database.LeaveAttempt(ctx, h.db, input.ID)
		if err != nil {
			log.Printf("leave attempt: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not leave attempt")
		}
		resp.Body.Attempt = attempt
		return resp, nil
	})
}
