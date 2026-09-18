package handlers

import (
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"

	"github.com/coder/websocket"

	"backend/database"
	"backend/middleware"
)

// AttemptWebSocket is GET /api/v1/attempts/{id}/ws — a raw http.HandlerFunc
// (not a huma operation, since the protocol upgrade stops being HTTP),
// mounted directly on root in main.go. The session cookie rides the
// handshake automatically, so middleware.IdentityFrom works the same as in
// any huma handler.
func (h *Handlers) AttemptWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	identity := middleware.IdentityFrom(ctx)
	if identity == nil {
		writeProblem(w, http.StatusUnauthorized, "authentication required — please sign in")
		return
	}

	attemptID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid attempt id")
		return
	}

	attempt, err := database.GetAttempt(ctx, h.db, attemptID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeProblem(w, http.StatusNotFound, "attempt not found")
			return
		}
		log.Printf("attempt lookup: %v", err)
		writeProblem(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if attempt.UserID != identity.UserID {
		writeProblem(w, http.StatusForbidden, "you do not have permission to do this")
		return
	}

	runner, err := h.attempts.StartOrGet(ctx, attemptID)
	if err != nil {
		log.Printf("start attempt runner: %v", err)
		writeProblem(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	opts := &websocket.AcceptOptions{OriginPatterns: h.cfg.CORSOrigins}
	if slices.Contains(h.cfg.CORSOrigins, "*") {
		opts = &websocket.AcceptOptions{InsecureSkipVerify: true}
	}
	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		return
	}

	runner.Connect(conn)
}
