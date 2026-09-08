package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"

	"backend/auth"
	"backend/database"
	"backend/models"
)

// Authorization errors returned by the guard helpers; handlers translate
// them into HTTP responses.
var (
	ErrUnauthenticated = errors.New("authentication required")
	ErrForbidden       = errors.New("insufficient permissions")
)

type identityKey struct{}

// Identity is the parsed session attached to an authenticated request.
type Identity struct {
	UserID int64
}

// Authenticate inspects every request for a session token (auth cookie or
// Authorization: Bearer header) and attaches an Identity to the request
// context when one is found and valid. Requests without a valid token are
// passed through untouched; protected handlers reject them via the guards
// below.
func Authenticate(tokens *auth.Auth, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if c, err := r.Cookie(auth.SessionCookieName); err == nil {
			token = c.Value
		} else if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
		if token != "" {
			if userID, err := tokens.ParseSession(token); err == nil {
				ctx := context.WithValue(r.Context(), identityKey{}, Identity{UserID: userID})
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// IdentityFrom returns the identity attached by Authenticate, or nil.
func IdentityFrom(ctx context.Context) *Identity {
	if id, ok := ctx.Value(identityKey{}).(Identity); ok {
		return &id
	}
	return nil
}

// CurrentUser resolves the authenticated user from the database (fresh role,
// survives account deletion).
func CurrentUser(ctx context.Context, db *sqlx.DB) (models.User, error) {
	id := IdentityFrom(ctx)
	if id == nil {
		return models.User{}, ErrUnauthenticated
	}
	u, err := database.GetUserByID(ctx, db, id.UserID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return models.User{}, ErrUnauthenticated
		}
		return models.User{}, err
	}
	return u, nil
}

// RequireAnyRole resolves the current user and checks their role is one of
// the allowed set.
func RequireAnyRole(ctx context.Context, db *sqlx.DB, allowed ...models.Role) (models.User, error) {
	u, err := CurrentUser(ctx, db)
	if err != nil {
		return models.User{}, err
	}
	for _, role := range allowed {
		if u.Role == role {
			return u, nil
		}
	}
	return models.User{}, ErrForbidden
}
