package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
	"backend/models"
)

// InviteOutput is the response of the single-invite endpoints.
type InviteOutput struct {
	Body struct {
		Invite models.Invite `json:"invite"`
	}
}

// AttemptOutput is the response of the single-attempt endpoints (accept
// invite, solo start, leave).
type AttemptOutput struct {
	Body struct {
		Attempt models.Attempt `json:"attempt"`
	}
}

// CreateInviteInput is the request of POST /api/v1/assessments/{id}/invites.
type CreateInviteInput struct {
	AssessmentID int64 `path:"id" example:"1" doc:"Assessment id to schedule a sitting for"`
	Body         struct {
		ScheduledStartAt time.Time `json:"scheduledStartAt" doc:"Must be strictly in the future"`
		TimeLimitMinutes *int      `json:"timeLimitMinutes,omitempty" example:"90" doc:"Overrides the assessment's own time limit for this sitting"`
		MaxUses          *int      `json:"maxUses,omitempty" example:"20" doc:"Cap on distinct acceptors; omit for unlimited"`
	}
}

// AcceptInviteInput is the request of POST /api/v1/invites/{code}/accept.
type AcceptInviteInput struct {
	Code string `path:"code" example:"a1b2c3d4e5f6" doc:"Invite code to accept"`
}

func (h *Handlers) registerAssessmentInvites(api huma.API) {
	// POST /api/v1/assessments/{id}/invites — author or admin only.
	huma.Register(api, huma.Operation{
		OperationID: "createAssessmentInvite",
		Method:      http.MethodPost,
		Path:        "/api/v1/assessments/{id}/invites",
		Summary:     "Schedule an invite for an assessment",
		Description: "The author of the assessment or an admin/superadmin may schedule a group sitting: pick a time, an optional headcount, and get a shareable code.",
	}, func(ctx context.Context, input *CreateInviteInput) (*InviteOutput, error) {
		if !input.Body.ScheduledStartAt.After(time.Now()) {
			return nil, unprocessable("scheduledStartAt must be strictly in the future")
		}
		if input.Body.TimeLimitMinutes != nil && *input.Body.TimeLimitMinutes <= 0 {
			return nil, unprocessable("timeLimitMinutes must be greater than zero")
		}
		if input.Body.MaxUses != nil && *input.Body.MaxUses <= 0 {
			return nil, unprocessable("maxUses must be greater than zero")
		}

		host, err := h.requireManageableAssessment(ctx, input.AssessmentID)
		if err != nil {
			return nil, err
		}
		invite, err := database.CreateInvite(ctx, h.db, input.AssessmentID, host.ID,
			input.Body.ScheduledStartAt, input.Body.TimeLimitMinutes, input.Body.MaxUses)
		if err != nil {
			return nil, toHTTPError(err)
		}
		return &InviteOutput{Body: struct {
			Invite models.Invite `json:"invite"`
		}{Invite: invite}}, nil
	})

	// POST /api/v1/invites/{code}/accept — any authenticated user.
	huma.Register(api, huma.Operation{
		OperationID: "acceptInvite",
		Method:      http.MethodPost,
		Path:        "/api/v1/invites/{code}/accept",
		Summary:     "Accept an invite",
		Description: "Validates the cap and that the window hasn't fully closed, then creates (or returns) this user's pending attempt under that invite. Accepting never itself starts the clock.",
	}, func(ctx context.Context, input *AcceptInviteInput) (*AttemptOutput, error) {
		host, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}

		attempt, err := database.AcceptInvite(ctx, h.db, input.Code, host.ID)
		if err != nil {
			switch {
			case errors.Is(err, database.ErrInviteExpired):
				return nil, conflict("Invite window has closed")
			case errors.Is(err, database.ErrInviteMaxUses):
				return nil, conflict("Invite has reached its maximum number of uses")
			case database.IsUniqueViolation(err):
				return nil, conflict("you already have an active attempt at this assessment")
			default:
				return nil, toHTTPError(err)
			}
		}
		return &AttemptOutput{Body: struct {
			Attempt models.Attempt `json:"attempt"`
		}{Attempt: attempt}}, nil
	})
}
