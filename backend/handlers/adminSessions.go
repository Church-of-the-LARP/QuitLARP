package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"backend/models"
	"backend/sessions"
)

type AdminSessionsOutput struct {
	Body struct {
		Sessions []sessions.AdminSession `json:"sessions"`
	}
}

type AdminSessionIDInput struct {
	ID string `path:"id" doc:"Assessment session identifier"`
}

func (h *Handlers) registerAdminSessions(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listAdminSessions",
		Method:      http.MethodGet,
		Path:        "/api/v1/admin/sessions",
		Summary:     "List live gVisor assessment instances",
		Description: "Requires the admin or superadmin role. Returns assessment session containers and their runtime state.",
	}, func(ctx context.Context, _ *struct{}) (*AdminSessionsOutput, error) {
		if _, err := h.requireUser(ctx, models.RoleAdmin, models.RoleSuperadmin); err != nil {
			return nil, err
		}
		items, err := h.sessions.AdminSessions(ctx)
		if err != nil {
			log.Printf("list admin sessions: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list live sessions")
		}
		response := &AdminSessionsOutput{}
		response.Body.Sessions = items
		return response, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "stopAdminSession",
		Method:      http.MethodDelete,
		Path:        "/api/v1/admin/sessions/{id}",
		Summary:     "Stop a live gVisor assessment instance",
		Description: "Requires the admin or superadmin role. Ends the session and removes its sandbox container without saving a solution.",
	}, func(ctx context.Context, input *AdminSessionIDInput) (*struct{}, error) {
		if _, err := h.requireUser(ctx, models.RoleAdmin, models.RoleSuperadmin); err != nil {
			return nil, err
		}
		if err := h.sessions.AdminStop(ctx, input.ID); err != nil {
			if errors.Is(err, sessions.ErrSessionNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "live session not found")
			}
			log.Printf("stop admin session %s: %v", input.ID, err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not stop the live session")
		}
		return nil, nil
	})
}
