// Package runtime is the live-taking engine for assessment attempts: one
// goroutine per attempt (a Runner) owns the candidate's clock and chapter
// cursor and talks to the one browser connected to it over a WebSocket
// (a hub). See docs/assessment-runtime/TASK.md for the full design.
package runtime

import (
	"encoding/json"
	"time"
)

// Client -> server message types.
const (
	typeChapterAdvance = "chapter.advance"
	typeLeave          = "leave"
)

// Server -> client message types.
const (
	typeAttemptSnapshot   = "attempt.snapshot"
	typeAttemptWaiting    = "attempt.waiting"
	typeAttemptStarted    = "attempt.started"
	typeChapterTransition = "chapter.transition"
	typeChapterRejected   = "chapter.rejected"
	typeTick              = "tick"
	typeAttemptEnded      = "attempt.ended"
	typeError             = "error"
)

// attempt.ended reasons.
const (
	reasonTimeUp    = "time_up"
	reasonCompleted = "completed"
	reasonAborted   = "aborted"
	reasonExpired   = "expired"
)

// chapter.rejected reasons.
const reasonOutOfOrder = "out_of_order"

// finishChapterID is the chapterId sentinel a candidate on the last chapter
// sends to signal "no further chapter, I'm done" — chapter.advance never
// otherwise has anything to advance to past the last real chapter, and this
// reuses the one client->server navigation message rather than inventing a
// second one. Documented design decision (see TASK.md's own pattern of
// calling out choices left ambiguous by the spec).
const finishChapterID int64 = 0

// envelope is the WS wire format: {"type": "...", "payload": {...}}.
type envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ChapterAdvancePayload is the client->server chapter.advance payload.
type ChapterAdvancePayload struct {
	ChapterID int64 `json:"chapterId"`
}

// SnapshotPayload is the attempt.snapshot payload: enough to render the
// page on load or reconnect. Mirrors AttemptSnapshotOutput in
// handlers/assessmentAttempts.go so REST and WS agree on shape.
type SnapshotPayload struct {
	Status               string     `json:"status"`
	ChapterID            *int64     `json:"chapterId"`
	TimeRemainingSeconds int        `json:"timeRemainingSeconds"`
	ScheduledStartAt     *time.Time `json:"scheduledStartAt,omitempty"`
}

// WaitingPayload is the attempt.waiting payload.
type WaitingPayload struct {
	ScheduledStartAt time.Time `json:"scheduledStartAt"`
	StartsInSeconds  int       `json:"startsInSeconds"`
}

// StartedPayload is the attempt.started payload.
type StartedPayload struct {
	StartedAt time.Time `json:"startedAt"`
}

// ChapterTransitionPayload is the chapter.transition payload.
type ChapterTransitionPayload struct {
	FromChapterID    *int64 `json:"fromChapterId"`
	ToChapterID      int64  `json:"toChapterId"`
	ChapterTitle     string `json:"chapterTitle"`
	TimeLimitMinutes int    `json:"timeLimitMinutes"`
}

// ChapterRejectedPayload is the chapter.rejected payload.
type ChapterRejectedPayload struct {
	Reason string `json:"reason"`
}

// TickPayload is the tick payload.
type TickPayload struct {
	TimeRemainingSeconds int `json:"timeRemainingSeconds"`
}

// EndedPayload is the attempt.ended payload.
type EndedPayload struct {
	Reason string `json:"reason"`
}

// ErrorPayload is the error payload.
type ErrorPayload struct {
	Message string `json:"message"`
}

// encodeEnvelope marshals a typed payload into the {"type","payload"} wire
// format.
func encodeEnvelope(msgType string, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope{Type: msgType, Payload: raw})
}
