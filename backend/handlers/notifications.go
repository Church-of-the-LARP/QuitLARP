package handlers

import (
	"context"
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

func (h *Handlers) registerNotifications(api huma.API) {
	// GET /api/v1/notifications — the caller's notification feed.
	huma.Register(api, huma.Operation{
		OperationID: "listNotifications",
		Method:      http.MethodGet,
		Path:        "/api/v1/notifications",
		Summary:     "List notifications",
		Description: "Requires authentication. Returns the caller's notifications, newest first.",
	}, func(ctx context.Context, input *struct {
		Limit int `query:"limit" example:"20" doc:"Maximum number of entries (1-100, default 20)"`
	}) (*NotificationsOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		limit := input.Limit
		if limit == 0 {
			limit = 20
		}
		if limit < 1 || limit > 100 {
			return nil, unprocessable("limit must be between 1 and 100")
		}
		items, err := database.ListNotifications(ctx, h.db, u.ID, limit)
		if err != nil {
			log.Printf("list notifications: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list notifications")
		}
		resp := &NotificationsOutput{}
		resp.Body.Notifications = items
		return resp, nil
	})
}
