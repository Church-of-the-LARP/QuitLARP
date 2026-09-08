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
)

// VerifyEmailInput carries the token from the emailed link.
type VerifyEmailInput struct {
	Token string `query:"token" required:"true" doc:"Signed token from the verification email"`
}

// ForgotPasswordInput is the JSON body of POST /api/v1/auth/forgot-password.
type ForgotPasswordInput struct {
	Body struct {
		Email string `json:"email" format:"email" doc:"Account email address"`
	}
}

// ResetPasswordInput is the JSON body of POST /api/v1/auth/reset-password.
type ResetPasswordInput struct {
	Body struct {
		Token       string `json:"token" doc:"Signed token from the reset email"`
		NewPassword string `json:"newPassword" minLength:"8" maxLength:"72" doc:"New password (8-72 characters)"`
	}
}

func (h *Handlers) registerEmailFlows(api huma.API) {
	// GET /api/v1/auth/verify-email?token=... — opened from the emailed link.
	huma.Register(api, huma.Operation{
		OperationID: "verifyEmail",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/verify-email",
		Summary:     "Confirm an email address from a signed link",
	}, func(ctx context.Context, input *VerifyEmailInput) (*MessageOutput, error) {
		userID, err := h.tokens.ParseActionToken(auth.PurposeVerifyEmail, input.Token)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "this verification link is invalid or has expired")
		}
		u, err := database.GetUserByID(ctx, h.db, userID)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "this verification link is invalid or has expired")
		}
		if !u.EmailVerified {
			if err := database.SetEmailVerified(ctx, h.db, u.ID, true); err != nil {
				log.Printf("verify email: %v", err)
				return nil, huma.NewError(http.StatusInternalServerError, "could not verify email")
			}
		}
		resp := &MessageOutput{}
		resp.Body.Message = "Your email is verified — you can close this page and sign in."
		return resp, nil
	})

	// POST /api/v1/auth/resend-verification — signed-in users, no body needed.
	huma.Register(api, huma.Operation{
		OperationID: "resendVerification",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/resend-verification",
		Summary:     "Re-send the verification email to the signed-in user",
	}, func(ctx context.Context, _ *struct{}) (*MessageOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if u.EmailVerified {
			resp := &MessageOutput{}
			resp.Body.Message = "Your email is already verified."
			return resp, nil
		}
		if err := h.sendVerificationEmail(ctx, u); err != nil {
			log.Printf("resend verification mail: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not send verification email")
		}
		resp := &MessageOutput{}
		resp.Body.Message = "Verification email sent. Open the link inside to confirm your email address."
		return resp, nil
	})

	// POST /api/v1/auth/forgot-password — always answers the same way so
	// attackers cannot enumerate registered emails.
	huma.Register(api, huma.Operation{
		OperationID: "forgotPassword",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/forgot-password",
		Summary:     "Request a password reset email",
	}, func(ctx context.Context, input *ForgotPasswordInput) (*MessageOutput, error) {
		email := strings.ToLower(strings.TrimSpace(input.Body.Email))
		row, err := database.GetUserWithPassword(ctx, h.db, email)
		if err == nil && row.PasswordHash != nil {
			if err := h.sendResetEmail(ctx, row.User); err != nil {
				log.Printf("forgot password mail: %v", err)
			}
		} else if err != nil && !errors.Is(err, database.ErrNotFound) {
			log.Printf("forgot password lookup: %v", err)
		}
		// Identical response for found/not-found accounts.
		resp := &MessageOutput{}
		resp.Body.Message = "If an account exists for that email, a reset link is on its way."
		return resp, nil
	})

	// POST /api/v1/auth/reset-password — finishes the flow from the SPA form.
	huma.Register(api, huma.Operation{
		OperationID: "resetPassword",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/reset-password",
		Summary:     "Set a new password using a reset token",
	}, func(ctx context.Context, input *ResetPasswordInput) (*MessageOutput, error) {
		userID, err := h.tokens.ParseActionToken(auth.PurposeResetPassword, input.Body.Token)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "this reset link is invalid or has expired")
		}
		if err := auth.ValidatePassword(input.Body.NewPassword); err != nil {
			return nil, huma.NewError(http.StatusUnprocessableEntity, err.Error())
		}
		u, err := database.GetUserByID(ctx, h.db, userID)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "this reset link is invalid or has expired")
		}
		hash, err := auth.HashPassword(input.Body.NewPassword)
		if err != nil {
			log.Printf("reset hash: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not secure password")
		}
		if err := database.SetPasswordHash(ctx, h.db, u.ID, hash); err != nil {
			log.Printf("reset password: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update password")
		}
		resp := &MessageOutput{}
		resp.Body.Message = "Your password has been updated — you can now sign in with it."
		return resp, nil
	})
}
