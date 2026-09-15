package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"backend/auth"
	"backend/database"
	"backend/middleware"
	"backend/models"
)

type UserOutput struct {
	Body struct {
		User models.User `json:"user"`
	}
}

type MessageOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

// RegisterInput is the JSON body of POST /api/v1/auth/register.
type RegisterInput struct {
	Body struct {
		Username string `json:"username" minLength:"3" maxLength:"32" doc:"Desired username (3-32 chars, letters/digits/-/_)"`
		Email    string `json:"email" format:"email" doc:"Login email address"`
		Password string `json:"password" minLength:"8" maxLength:"72" doc:"Password (8-72 characters)"`
	}
}

// LoginInput is the JSON body of POST /api/v1/auth/login.
type LoginInput struct {
	Body struct {
		Email    string `json:"email" format:"email" doc:"Login email address"`
		Password string `json:"password" doc:"Account password"`
	}
}

func (h *Handlers) registerAuth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "register",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/register",
		Summary:     "Create an account with email + password",
		Description: "Signs the user in immediately and sends a verification link by email.",
	}, func(ctx context.Context, input *RegisterInput) (*UserOutput, error) {
		username := strings.TrimSpace(input.Body.Username)
		email := strings.ToLower(strings.TrimSpace(input.Body.Email))

		if err := auth.ValidateUsername(username); err != nil {
			return nil, huma.NewError(http.StatusUnprocessableEntity, err.Error())
		}
		if !validEmail(email) {
			return nil, huma.NewError(http.StatusUnprocessableEntity, auth.ErrInvalidEmail.Error())
		}
		if err := auth.ValidatePassword(input.Body.Password); err != nil {
			return nil, huma.NewError(http.StatusUnprocessableEntity, err.Error())
		}

		hash, err := auth.HashPassword(input.Body.Password)
		if err != nil {
			log.Printf("register hash: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not secure password")
		}
		u, err := database.CreateUser(ctx, h.db, username, email, &hash, models.RoleUser, false, nil)
		if err != nil {
			if database.IsUniqueViolation(err) {
				return nil, huma.NewError(http.StatusConflict, "that email or username is already registered")
			}
			log.Printf("register create user: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not create account")
		}

		if err := h.signInHuma(ctx, u); err != nil {
			log.Printf("register sign in: %v", err)
		}
		if err := h.sendVerificationEmail(ctx, u); err != nil {
			log.Printf("register verification mail: %v", err)
		}

		out := &UserOutput{}
		out.Body.User = u
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "Sign in with email and password",
	}, func(ctx context.Context, input *LoginInput) (*UserOutput, error) {
		email := strings.ToLower(strings.TrimSpace(input.Body.Email))
		row, err := database.GetUserWithPassword(ctx, h.db, email)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusUnauthorized, "invalid email or password")
			}
			log.Printf("login lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not sign in")
		}
		// Google-only accounts have no password: same generic failure.
		if row.PasswordHash == nil || !auth.CheckPassword(*row.PasswordHash, input.Body.Password) {
			return nil, huma.NewError(http.StatusUnauthorized, "invalid email or password")
		}

		if err := h.signInHuma(ctx, row.User); err != nil {
			log.Printf("login sign in: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not sign in")
		}

		out := &UserOutput{}
		out.Body.User = row.User
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Sign out and clear the session cookie",
	}, func(ctx context.Context, _ *struct{}) (*MessageOutput, error) {
		middleware.AddCookie(ctx, h.sessionCookie("", -1))
		resp := &MessageOutput{}
		resp.Body.Message = "signed out"
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMe",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/me",
		Summary:     "Get the signed-in user",
	}, func(ctx context.Context, _ *struct{}) (*UserOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		out := &UserOutput{}
		out.Body.User = u
		return out, nil
	})
}

// signInHuma queues the session cookie via middleware.AddCookie; the
// middleware.CookieJar wrapper attaches it to the actual response.
func (h *Handlers) signInHuma(ctx context.Context, u models.User) error {
	token, err := h.tokens.SignSession(u.ID)
	if err != nil {
		return err
	}
	middleware.AddCookie(ctx, h.sessionCookie(token, int(h.cfg.SessionTTL.Seconds())))
	return nil
}
