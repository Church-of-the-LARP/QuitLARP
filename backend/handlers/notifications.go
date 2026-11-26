package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
	"backend/models"
)

// NotificationsOutput is the response of GET /api/v1/notifications.
type NotificationsOutput struct {
	Body struct {
		Notifications []models.Notification `json:"notifications"`
	}
}

// ListNotificationsInput is the request of GET /api/v1/notifications.
type ListNotificationsInput struct {
	Offset int `query:"offset" example:"0" doc:"Number of results to skip"`
	Limit  int `query:"limit" example:"20" doc:"Maximum number of results (1-100, default 20)"`
}

func (h *Handlers) registerNotifications(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listNotifications",
		Method:      http.MethodGet,
		Path:        "/api/v1/notifications",
		Summary:     "List my notifications",
		Description: "Requires authentication. Returns the caller's stored notifications, newest first. Notifications are written by server-side events and are only visible after the next fetch; there is no realtime channel.",
	}, func(ctx context.Context, input *ListNotificationsInput) (*NotificationsOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if input.Offset < 0 {
			return nil, unprocessable("offset must be zero or greater")
		}
		limit := input.Limit
		if limit == 0 {
			limit = 20
		}
		if limit < 1 || limit > 100 {
			return nil, unprocessable("limit must be between 1 and 100")
		}
		list, err := database.ListNotifications(ctx, h.db, u.ID, limit, input.Offset)
		if err != nil {
			log.Printf("list notifications: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list notifications")
		}
		resp := &NotificationsOutput{}
		resp.Body.Notifications = list
		return resp, nil
	})
}

// notifyUser stores one notification. Notification failures are logged but
// never fail the action that triggered them.
func (h *Handlers) notifyUser(ctx context.Context, userID int64, title, detail string) {
	if userID == 0 {
		return
	}
	if err := database.CreateNotification(ctx, h.db, userID, title, detail); err != nil {
		log.Printf("create notification for user %d: %v", userID, err)
	}
}

// notifyEventHosts stores one notification for every host of an event except
// the user who triggered it.
func (h *Handlers) notifyEventHosts(ctx context.Context, eventID, actorID int64, title, detail string) {
	hosts, err := database.ListEventHostIDs(ctx, h.db, eventID)
	if err != nil {
		log.Printf("notification hosts for event %d: %v", eventID, err)
		return
	}
	for _, hostID := range hosts {
		if hostID != actorID {
			h.notifyUser(ctx, hostID, title, detail)
		}
	}
}

// notifyAssessmentResult tells each host of a monitored interview containing
// the assessment that the solver submitted a solution.
func (h *Handlers) notifyAssessmentResult(ctx context.Context, solver models.User, assessmentID int64) {
	targets, err := database.ListInterviewResultTargets(ctx, h.db, solver.ID, assessmentID)
	if err != nil {
		log.Printf("notification targets for assessment %d user %d: %v", assessmentID, solver.ID, err)
		return
	}
	for _, target := range targets {
		if target.HostID == solver.ID {
			continue
		}
		h.notifyUser(ctx, target.HostID, "New interview result",
			fmt.Sprintf("%s submitted a solution for %q in %q.",
				solver.Username, target.TaskTitle, target.EventTitle))
	}
}
