package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
	"backend/models"
)

// UsersOutput is the response of GET /api/v1/users.
type UsersOutput struct {
	Body struct {
		Users []models.User `json:"users"`
	}
}

// UpdateUserRoleInput is the request of PATCH /api/v1/users/{id}/role.
type UpdateUserRoleInput struct {
	ID   int64 `path:"id" example:"1" doc:"User id to change"`
	Body struct {
		Role models.Role `json:"role" enum:"user,moderator,admin" doc:"New role"`
	}
}

func (h *Handlers) registerUsers(api huma.API) {
	// GET /api/v1/users — admins only.
	huma.Register(api, huma.Operation{
		OperationID: "listUsers",
		Method:      http.MethodGet,
		Path:        "/api/v1/users",
		Summary:     "List users (admin only)",
		Description: "Requires the caller to hold the admin role.",
	}, func(ctx context.Context, _ *struct{}) (*UsersOutput, error) {
		if _, err := h.requireUser(ctx, models.RoleAdmin); err != nil {
			return nil, err
		}
		users, err := database.ListUsers(ctx, h.db)
		if err != nil {
			log.Printf("list users: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list users")
		}
		resp := &UsersOutput{}
		resp.Body.Users = users
		return resp, nil
	})

	// PATCH /api/v1/users/{id}/role — admins may assign any supported role.
	huma.Register(api, huma.Operation{
		OperationID: "updateUserRole",
		Method:      http.MethodPatch,
		Path:        "/api/v1/users/{id}/role",
		Summary:     "Change a user's role (admin only)",
		Description: "An admin may assign the user, moderator, or admin role to any account.",
	}, func(ctx context.Context, input *UpdateUserRoleInput) (*UserOutput, error) {
		caller, err := h.requireUser(ctx, models.RoleAdmin)
		if err != nil {
			return nil, err
		}

		newRole, ok := models.ParseRole(string(input.Body.Role))
		if !ok {
			return nil, huma.NewError(http.StatusUnprocessableEntity, "role must be one of: user, moderator, admin")
		}

		target, err := database.GetUserByID(ctx, h.db, input.ID)
		if err != nil {
			if err == database.ErrNotFound {
				return nil, huma.NewError(http.StatusNotFound, "user not found")
			}
			log.Printf("update role lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update role")
		}
		// The configured bootstrap admin can never be demoted, and no admin
		// may demote themselves (so the last admin cannot lock everyone out).
		if newRole != models.RoleAdmin {
			if strings.EqualFold(target.Username, h.cfg.Admin.Username) {
				return nil, huma.NewError(http.StatusForbidden, "the configured bootstrap admin cannot be demoted")
			}
			if target.ID == caller.ID {
				return nil, huma.NewError(http.StatusForbidden, "admins cannot demote themselves")
			}
		}

		if err := database.SetUserRole(ctx, h.db, target.ID, newRole); err != nil {
			log.Printf("update role: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update role")
		}
		updated, err := database.GetUserByID(ctx, h.db, target.ID)
		if err != nil {
			log.Printf("update role re-read: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update role")
		}
		out := &UserOutput{}
		out.Body.User = updated
		return out, nil
	})
}
