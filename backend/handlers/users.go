package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"backend/auth"
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
		Role models.Role `json:"role" enum:"user,admin,superadmin" doc:"New role"`
	}
}

// UpdateMeInput is the request of PATCH /api/v1/users/me.
type UpdateMeInput struct {
	Body struct {
		Username        string `json:"username,omitempty"`
		Email           string `json:"email,omitempty"`
		CurrentPassword string `json:"currentPassword,omitempty"`
		NewPassword     string `json:"newPassword,omitempty"`
	}
}

func (h *Handlers) registerUsers(api huma.API) {
	// GET /api/v1/users — any admin or the superadmin.
	huma.Register(api, huma.Operation{
		OperationID: "listUsers",
		Method:      http.MethodGet,
		Path:        "/api/v1/users",
		Summary:     "List users (admin or superadmin)",
		Description: "Requires the caller to hold the admin or superadmin role.",
	}, func(ctx context.Context, _ *struct{}) (*UsersOutput, error) {
		if _, err := h.requireUser(ctx, models.RoleAdmin, models.RoleSuperadmin); err != nil {
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

	// PATCH /api/v1/users/{id}/role — only the bootstrap superadmin.
	huma.Register(api, huma.Operation{
		OperationID: "updateUserRole",
		Method:      http.MethodPatch,
		Path:        "/api/v1/users/{id}/role",
		Summary:     "Change a user's role (superadmin only)",
		Description: "Only the superadmin defined via SUPERADMIN_* env vars may change roles, and only that account may hold the superadmin role.",
	}, func(ctx context.Context, input *UpdateUserRoleInput) (*UserOutput, error) {
		if _, err := h.requireUser(ctx, models.RoleSuperadmin); err != nil {
			return nil, err
		}

		newRole, ok := models.ParseRole(string(input.Body.Role))
		if !ok {
			return nil, huma.NewError(http.StatusUnprocessableEntity, "role must be one of: user, admin, superadmin")
		}

		target, err := database.GetUserByID(ctx, h.db, input.ID)
		if err != nil {
			if err == database.ErrNotFound {
				return nil, huma.NewError(http.StatusNotFound, "user not found")
			}
			log.Printf("update role lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update role")
		}
		// Keep the bootstrap superadmin singular: it cannot be demoted and
		// nobody else may be promoted into its role.
		isBootstrap := strings.EqualFold(target.Username, h.cfg.Superadmin.Username)
		if newRole == models.RoleSuperadmin && !isBootstrap {
			return nil, huma.NewError(http.StatusForbidden, "only the configured superadmin may hold the superadmin role")
		}
		if isBootstrap && newRole != models.RoleSuperadmin {
			return nil, huma.NewError(http.StatusForbidden, "the configured superadmin cannot be demoted")
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

	// PATCH /api/v1/users/me — signed-in user may update username, email or password.
	huma.Register(api, huma.Operation{
		OperationID: "updateMe",
		Method:      http.MethodPatch,
		Path:        "/api/v1/users/me",
		Summary:     "Update the signed-in user's profile",
	}, func(ctx context.Context, input *UpdateMeInput) (*UserOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}

		if username := strings.TrimSpace(input.Body.Username); username != "" {
			if err := auth.ValidateUsername(username); err != nil {
				return nil, huma.NewError(http.StatusUnprocessableEntity, err.Error())
			}
			taken, err := database.UsernameTaken(ctx, h.db, username)
			if err != nil {
				log.Printf("update me username check: %v", err)
				return nil, huma.NewError(http.StatusInternalServerError, "could not update profile")
			}
			if taken && !strings.EqualFold(username, u.Username) {
				return nil, huma.NewError(http.StatusConflict, "that username is already taken")
			}
			if err := database.UpdateUserUsername(ctx, h.db, u.ID, username); err != nil {
				log.Printf("update me username: %v", err)
				return nil, huma.NewError(http.StatusInternalServerError, "could not update username")
			}
			u.Username = username
		}

		if email := strings.TrimSpace(input.Body.Email); email != "" {
			email = strings.ToLower(email)
			if !validEmail(email) {
				return nil, huma.NewError(http.StatusUnprocessableEntity, auth.ErrInvalidEmail.Error())
			}
			if !strings.EqualFold(email, u.Email) {
				if _, err := database.GetUserByEmail(ctx, h.db, email); err == nil {
					return nil, huma.NewError(http.StatusConflict, "that email is already registered")
				} else if err != database.ErrNotFound {
					log.Printf("update me email lookup: %v", err)
					return nil, huma.NewError(http.StatusInternalServerError, "could not update email")
				}
				if err := database.UpdateUserEmail(ctx, h.db, u.ID, email); err != nil {
					log.Printf("update me email: %v", err)
					return nil, huma.NewError(http.StatusInternalServerError, "could not update email")
				}
				if err := database.SetEmailVerified(ctx, h.db, u.ID, false); err != nil {
					log.Printf("update me email verification flag: %v", err)
					return nil, huma.NewError(http.StatusInternalServerError, "could not update email")
				}
				u.Email = email
				u.EmailVerified = false
				if err := h.sendVerificationEmail(ctx, u); err != nil {
					log.Printf("update me verification mail: %v", err)
					return nil, huma.NewError(http.StatusInternalServerError, "could not send verification email")
				}
			}
		}

		if input.Body.NewPassword != "" {
			if input.Body.CurrentPassword == "" {
				return nil, huma.NewError(http.StatusUnprocessableEntity, "current password is required")
			}
			row, err := database.GetUserWithPassword(ctx, h.db, u.Email)
			if err != nil {
				log.Printf("update me password lookup: %v", err)
				return nil, huma.NewError(http.StatusInternalServerError, "could not update password")
			}
			if row.PasswordHash == nil || !auth.CheckPassword(*row.PasswordHash, input.Body.CurrentPassword) {
				return nil, huma.NewError(http.StatusUnauthorized, "current password is incorrect")
			}
			if err := auth.ValidatePassword(input.Body.NewPassword); err != nil {
				return nil, huma.NewError(http.StatusUnprocessableEntity, err.Error())
			}
			hash, err := auth.HashPassword(input.Body.NewPassword)
			if err != nil {
				log.Printf("update me hash: %v", err)
				return nil, huma.NewError(http.StatusInternalServerError, "could not secure password")
			}
			if err := database.SetPasswordHash(ctx, h.db, u.ID, hash); err != nil {
				log.Printf("update me password: %v", err)
				return nil, huma.NewError(http.StatusInternalServerError, "could not update password")
			}
		}

		updated, err := database.GetUserByID(ctx, h.db, u.ID)
		if err != nil {
			log.Printf("update me re-read: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update profile")
		}
		out := &UserOutput{}
		out.Body.User = updated
		return out, nil
	})
}
