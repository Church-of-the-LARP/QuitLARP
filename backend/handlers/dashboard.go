package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
)

// UserStatsOutput is the response of GET /api/v1/users/me/stats.
type UserStatsOutput struct {
	Body struct {
		Completed      int64 `json:"completed" doc:"Number of assessments the caller has submitted a solution for"`
		InProgress     int   `json:"inProgress" doc:"Number of assessments the caller is currently solving"`
		CompletionRate int   `json:"completionRate" doc:"Completed as a percentage of completed plus in-progress, 0-100"`
	}
}

// continuingSession is one of the caller's live solving sessions.
type continuingSession struct {
	AssessmentID     int64  `json:"assessmentId"`
	AssessmentTitle  string `json:"assessmentTitle"`
	ChapterTitle     string `json:"chapterTitle"`
	Deadline         string `json:"deadline" doc:"When the session ends, RFC 3339"`
	TimeLimitMinutes *int   `json:"timeLimitMinutes" doc:"Assessment time limit, null when unlimited"`
}

// UserSessionsOutput is the response of GET /api/v1/users/me/sessions.
type UserSessionsOutput struct {
	Body struct {
		Sessions []continuingSession `json:"sessions"`
	}
}

// UserNotificationsOutput is the response of GET
// /api/v1/users/me/notifications.
type UserNotificationsOutput struct {
	Body struct {
		Notifications []notificationBody `json:"notifications"`
	}
}

type notificationBody struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Detail    string  `json:"detail"`
	Read      bool    `json:"read"`
	CreatedAt string  `json:"createdAt" doc:"RFC 3339"`
	ReadAt    *string `json:"readAt" doc:"RFC 3339, null while unread"`
}

func (h *Handlers) registerDashboard(api huma.API) {
	// GET /api/v1/users/me/stats
	huma.Register(api, huma.Operation{
		OperationID: "getMyStats",
		Method:      http.MethodGet,
		Path:        "/api/v1/users/me/stats",
		Summary:     "Get my progress stats",
		Description: "Requires authentication. Completed counts submitted solutions; in-progress counts currently running solving sessions.",
	}, func(ctx context.Context, _ *struct{}) (*UserStatsOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		completed, err := database.CountSolutionsByUser(ctx, h.db, u.ID)
		if err != nil {
			log.Printf("count solutions: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not load stats")
		}
		inProgress := len(h.sessions.ListByUser(u.ID))

		resp := &UserStatsOutput{}
		resp.Body.Completed = completed
		resp.Body.InProgress = inProgress
		total := completed + int64(inProgress)
		if total > 0 {
			resp.Body.CompletionRate = int(completed * 100 / total)
		}
		return resp, nil
	})

	// GET /api/v1/users/me/sessions
	huma.Register(api, huma.Operation{
		OperationID: "listMySessions",
		Method:      http.MethodGet,
		Path:        "/api/v1/users/me/sessions",
		Summary:     "List my running solving sessions",
		Description: "Requires authentication. Returns every assessment the caller is currently solving, across the whole process.",
	}, func(ctx context.Context, _ *struct{}) (*UserSessionsOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		live := h.sessions.ListByUser(u.ID)
		resp := &UserSessionsOutput{}
		resp.Body.Sessions = make([]continuingSession, 0, len(live))
		for _, session := range live {
			cs := continuingSession{
				AssessmentID:    session.AssessmentID,
				AssessmentTitle: session.AssessmentTitle,
				ChapterTitle:    session.ChapterTitle,
				Deadline:        session.Deadline.UTC().Format(time.RFC3339),
			}
			if session.TimeLimit > 0 {
				limit := session.TimeLimit
				cs.TimeLimitMinutes = &limit
			}
			resp.Body.Sessions = append(resp.Body.Sessions, cs)
		}
		return resp, nil
	})

	// GET /api/v1/users/me/notifications
	huma.Register(api, huma.Operation{
		OperationID: "listMyNotifications",
		Method:      http.MethodGet,
		Path:        "/api/v1/users/me/notifications",
		Summary:     "List my notifications",
		Description: "Requires authentication. Returns the caller's own notifications, newest first.",
	}, func(ctx context.Context, _ *struct{}) (*UserNotificationsOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		list, err := database.ListNotificationsByUser(ctx, h.db, u.ID, 20)
		if err != nil {
			log.Printf("list notifications: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not load notifications")
		}
		resp := &UserNotificationsOutput{}
		resp.Body.Notifications = make([]notificationBody, 0, len(list))
		for _, n := range list {
			body := notificationBody{
				ID:        n.ID,
				Title:     n.Title,
				Detail:    n.Detail,
				Read:      n.ReadAt != nil,
				CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
			}
			if n.ReadAt != nil {
				readAt := n.ReadAt.UTC().Format(time.RFC3339)
				body.ReadAt = &readAt
			}
			resp.Body.Notifications = append(resp.Body.Notifications, body)
		}
		return resp, nil
	})

	// POST /api/v1/users/me/notifications/read
	huma.Register(api, huma.Operation{
		OperationID: "markMyNotificationsRead",
		Method:      http.MethodPost,
		Path:        "/api/v1/users/me/notifications/read",
		Summary:     "Mark every notification read",
		Description: "Requires authentication.",
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := database.MarkAllNotificationsRead(ctx, h.db, u.ID); err != nil {
			log.Printf("mark notifications read: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update notifications")
		}
		return nil, nil
	})
}
