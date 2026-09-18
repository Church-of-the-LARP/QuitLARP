package runtime

import (
	"context"
	"encoding/json"
	"log"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/jmoiron/sqlx"

	"backend/database"
	"backend/models"
)

// atomicSnapshot lets Runner.Snapshot() be read from any goroutine without
// touching the fields loop() owns — the loop stores a fresh copy after
// every event it processes.
type atomicSnapshot struct {
	v atomic.Pointer[Snapshot]
}

func (a *atomicSnapshot) load() Snapshot {
	if p := a.v.Load(); p != nil {
		return *p
	}
	return Snapshot{}
}

func (a *atomicSnapshot) store(s Snapshot) {
	a.v.Store(&s)
}

// phase is the in-memory lifecycle of a live attempt. "Waiting for the
// schedule" only ever exists here — the persisted attempt status stays
// "pending" until phaseRunning is reached (see models.AttemptStatus).
type phase int

const (
	phaseWaiting phase = iota
	phaseRunning
	phaseEnded
)

// Snapshot is a point-in-time read of a Runner's state, safe to read
// concurrently with the Runner's own goroutine (see Runner.snapshot). It is
// the one thing about a Runner that something other than its own loop
// reads directly — everything else goes through commands.
type Snapshot struct {
	Status               models.AttemptStatus
	ChapterID            *int64
	TimeRemainingSeconds int
	ScheduledStartAt     *time.Time
}

// command is anything sent over Runner.commands. Each carries its own reply
// channel; a sender blocks on it (or on the Runner's context being done, if
// the Runner has already ended and nobody is left to reply).
type command interface{ isCommand() }

type connectCmd struct {
	conn  *websocket.Conn
	reply chan struct{}
}

func (*connectCmd) isCommand() {}

type advanceChapterCmd struct {
	chapterID int64
	reply     chan struct{}
}

func (*advanceChapterCmd) isCommand() {}

type leaveCmd struct {
	reply chan struct{}
}

func (*leaveCmd) isCommand() {}

// Runner is the one goroutine that owns a single attempt's clock and
// chapter cursor. phase, startedAt, deadline and chapterIdx are written
// only by loop(); everything else reaches the Runner through commands.
type Runner struct {
	id      int64
	manager *AttemptManager
	db      *sqlx.DB

	ctx    context.Context
	cancel context.CancelFunc

	commands chan command
	hub      *hub

	assessmentID     int64
	userID           int64
	timeLimitMinutes int
	chapters         []models.Chapter

	// Fields below this point are touched only by loop() and its helpers.
	phase            phase
	scheduledStartAt *time.Time // nil for a solo attempt
	deadline         time.Time  // valid once phase != phaseWaiting
	startedAt        *time.Time
	chapterIdx       int // -1 until the first chapter has begun

	// replayEnded marks a Runner constructed for an attempt that was
	// already over before this Runner existed (previously ended, or
	// found already time-expired/window-expired on construction). Its
	// only job is to answer the connect that triggered its creation and
	// then exit.
	replayEnded      bool
	initialEndReason string // sent as attempt.ended if replayEnded and non-empty

	snapshot atomicSnapshot
}

// newRunner loads an attempt (plus its invite and assessment's chapters)
// and derives the phase/deadline it should start in. This is the only
// place attempt state is read to decide "what should be happening right
// now" — a scheduled attempt's phase is a pure function of
// now/scheduledStartAt/timeLimitMinutes, so this same logic transparently
// resumes an attempt that was running when the server last restarted (see
// AttemptManager's package doc for the restart story).
func newRunner(ctx context.Context, m *AttemptManager, id int64) (*Runner, error) {
	attempt, err := database.GetAttempt(ctx, m.db, id)
	if err != nil {
		return nil, err
	}
	chapters, err := database.ListChapters(ctx, m.db, attempt.AssessmentID)
	if err != nil {
		return nil, err
	}

	var invite *models.Invite
	if attempt.InviteID != nil {
		inv, err := database.GetInvite(ctx, m.db, *attempt.InviteID)
		if err != nil {
			return nil, err
		}
		invite = &inv
	}

	runCtx, cancel := context.WithCancel(m.ctx)
	r := &Runner{
		id:               id,
		manager:          m,
		db:               m.db,
		ctx:              runCtx,
		cancel:           cancel,
		commands:         make(chan command),
		hub:              newHub(),
		assessmentID:     attempt.AssessmentID,
		userID:           attempt.UserID,
		timeLimitMinutes: attempt.TimeLimitMinutes,
		chapters:         chapters,
		chapterIdx:       -1,
	}
	if attempt.CurrentChapterID != nil {
		for i, ch := range chapters {
			if ch.ID == *attempt.CurrentChapterID {
				r.chapterIdx = i
				break
			}
		}
	}

	now := time.Now()
	limit := time.Duration(attempt.TimeLimitMinutes) * time.Minute

	switch {
	case attempt.Status == models.AttemptEnded:
		r.phase = phaseEnded
		r.replayEnded = true

	case attempt.StartedAt != nil:
		// Already running — either this Runner is resuming an in-flight
		// attempt after a server restart, or (impossible in practice,
		// since a Runner is the only thing that starts a clock) started
		// some other way. Either way, startedAt on the row is the truth.
		startedAt := *attempt.StartedAt
		r.startedAt = &startedAt
		r.deadline = startedAt.Add(limit)
		if !now.Before(r.deadline) {
			r.phase = phaseEnded
			r.replayEnded = true
			r.initialEndReason = reasonTimeUp
			_ = database.MarkAttemptEnded(ctx, m.db, id, now)
		} else {
			r.phase = phaseRunning
			r.beginFirstChapter(now)
		}

	case invite != nil:
		windowClose := invite.ScheduledStartAt.Add(limit)
		scheduledStartAt := invite.ScheduledStartAt
		r.scheduledStartAt = &scheduledStartAt
		r.deadline = windowClose
		switch {
		case !now.Before(windowClose):
			r.phase = phaseEnded
			r.replayEnded = true
			r.initialEndReason = reasonExpired
			_ = database.MarkAttemptEnded(ctx, m.db, id, now)
		case !now.Before(scheduledStartAt):
			r.phase = phaseRunning
			r.startedAt = &scheduledStartAt
			_ = database.MarkAttemptRunning(ctx, m.db, id, scheduledStartAt)
			r.beginFirstChapter(now)
		default:
			r.phase = phaseWaiting
		}

	default:
		// Solo, first ever connect: the clock starts right now.
		r.phase = phaseRunning
		r.startedAt = &now
		r.deadline = now.Add(limit)
		_ = database.MarkAttemptRunning(ctx, m.db, id, now)
		r.beginFirstChapter(now)
	}

	r.publishSnapshot()
	return r, nil
}

// loop is the single goroutine that owns this attempt's clock. Nothing
// outside this function (and the helpers it calls) ever writes phase,
// startedAt, deadline or chapterIdx.
func (r *Runner) loop() {
	defer r.cancel()
	defer r.manager.remove(r.id)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			// Abandoned, or the server is shutting down. The attempt's
			// row is left exactly as it was — see the restart note on
			// AttemptManager. The viewer's connection has its own
			// independent lifecycle (see hub.go), so it's force-closed
			// explicitly rather than left dangling.
			r.hub.closeCurrent()
			return
		case now := <-ticker.C:
			if r.handleTick(now) {
				return
			}
		case cmd := <-r.commands:
			if r.handleCommand(cmd) {
				return
			}
		case <-r.hub.disconnected():
			// The one viewer dropped. There is no reconnect grace period
			// to track — the clock keeps running regardless, and a
			// future Connect just re-registers a new viewer.
		}
	}
}

func (r *Runner) handleCommand(cmd command) (loopShouldEnd bool) {
	switch c := cmd.(type) {
	case *connectCmd:
		r.handleConnect(c)
		return r.replayEnded
	case *advanceChapterCmd:
		return r.handleAdvance(c)
	case *leaveCmd:
		r.handleLeave(c)
		return true
	default:
		return false
	}
}

func (r *Runner) handleTick(now time.Time) (ended bool) {
	switch r.phase {
	case phaseWaiting:
		if !now.Before(*r.scheduledStartAt) {
			r.startRunning(*r.scheduledStartAt)
			r.sendStarted()
		} else {
			r.sendWaiting(now)
		}
		r.publishSnapshot()
		return false
	case phaseRunning:
		if !now.Before(r.deadline) {
			r.endAttempt(reasonTimeUp)
			return true
		}
		r.sendTick(r.deadline.Sub(now))
		r.publishSnapshot()
		return false
	default:
		return false
	}
}

func (r *Runner) handleConnect(c *connectCmd) {
	defer close(c.reply)

	r.hub.set(c.conn, r.HandleMessage)
	r.sendSnapshot()

	switch {
	case r.replayEnded:
		if r.initialEndReason != "" {
			r.sendEnded(r.initialEndReason)
		}
		r.hub.closeCurrent()
	case r.phase == phaseWaiting:
		r.sendWaiting(time.Now())
	case r.phase == phaseRunning && r.scheduledStartAt != nil:
		// Connected after the schedule started (possibly already behind)
		// — this client never saw the waiting->running flip itself.
		r.sendStarted()
	}
}

func (r *Runner) handleAdvance(c *advanceChapterCmd) (ended bool) {
	defer close(c.reply)

	if r.phase != phaseRunning {
		r.sendRejected()
		return false
	}

	if c.chapterID == finishChapterID {
		if r.chapterIdx >= 0 && r.chapterIdx == len(r.chapters)-1 {
			r.completeCurrentChapter(time.Now())
			r.endAttempt(reasonCompleted)
			return true
		}
		r.sendRejected()
		return false
	}

	nextIdx := r.chapterIdx + 1
	if nextIdx >= len(r.chapters) || r.chapters[nextIdx].ID != c.chapterID {
		r.sendRejected()
		return false
	}

	r.advanceToChapter(nextIdx, time.Now())
	return false
}

func (r *Runner) handleLeave(c *leaveCmd) {
	defer close(c.reply)
	r.endAttempt(reasonAborted)
}

// startRunning flips a waiting attempt to running at exactly its scheduled
// instant.
func (r *Runner) startRunning(startedAt time.Time) {
	r.phase = phaseRunning
	r.startedAt = &startedAt
	_ = database.MarkAttemptRunning(r.ctx, r.db, r.id, startedAt)
	r.beginFirstChapter(startedAt)
}

// beginFirstChapter puts the candidate on chapter one, unless a chapter
// cursor was already resumed from the database.
func (r *Runner) beginFirstChapter(now time.Time) {
	if r.chapterIdx >= 0 || len(r.chapters) == 0 {
		return
	}
	r.chapterIdx = 0
	ch := r.chapters[0]
	_ = database.UpsertChapterProgressStart(r.ctx, r.db, r.id, ch.ID, now)
	_ = database.SetAttemptChapter(r.ctx, r.db, r.id, ch.ID)
}

func (r *Runner) advanceToChapter(nextIdx int, now time.Time) {
	from := r.chapters[r.chapterIdx]
	to := r.chapters[nextIdx]

	_ = database.CompleteChapterProgress(r.ctx, r.db, r.id, from.ID, now)
	_ = database.UpsertChapterProgressStart(r.ctx, r.db, r.id, to.ID, now)
	_ = database.SetAttemptChapter(r.ctx, r.db, r.id, to.ID)

	r.chapterIdx = nextIdx
	fromID := from.ID
	r.send(typeChapterTransition, ChapterTransitionPayload{
		FromChapterID:    &fromID,
		ToChapterID:      to.ID,
		ChapterTitle:     to.Title,
		TimeLimitMinutes: to.TimeLimitMinutes,
	})
	r.publishSnapshot()
}

func (r *Runner) completeCurrentChapter(now time.Time) {
	if r.chapterIdx < 0 || r.chapterIdx >= len(r.chapters) {
		return
	}
	_ = database.CompleteChapterProgress(r.ctx, r.db, r.id, r.chapters[r.chapterIdx].ID, now)
}

// endAttempt flips the Runner to phaseEnded, persists it and sends the
// final attempt.ended frame — always the last frame before the socket
// closes.
func (r *Runner) endAttempt(reason string) {
	now := time.Now()
	r.phase = phaseEnded
	_ = database.MarkAttemptEnded(r.ctx, r.db, r.id, now)
	r.sendEnded(reason)
	r.hub.closeCurrent()
	r.publishSnapshot()
}

// ---- outbound messages -----------------------------------------------

func (r *Runner) send(msgType string, payload any) {
	data, err := encodeEnvelope(msgType, payload)
	if err != nil {
		log.Printf("runtime: encode %s: %v", msgType, err)
		return
	}
	r.hub.send(data)
}

func (r *Runner) statusFor() models.AttemptStatus {
	switch r.phase {
	case phaseRunning:
		return models.AttemptRunning
	case phaseEnded:
		return models.AttemptEnded
	default:
		return models.AttemptPending
	}
}

func (r *Runner) currentChapterID() *int64 {
	if r.chapterIdx < 0 || r.chapterIdx >= len(r.chapters) {
		return nil
	}
	id := r.chapters[r.chapterIdx].ID
	return &id
}

func (r *Runner) sendSnapshot() {
	remaining := 0
	var scheduledStartAt *time.Time
	switch r.phase {
	case phaseWaiting:
		remaining = r.timeLimitMinutes * 60
		scheduledStartAt = r.scheduledStartAt
	case phaseRunning:
		remaining = clampSeconds(time.Until(r.deadline))
	}
	r.send(typeAttemptSnapshot, SnapshotPayload{
		Status:               string(r.statusFor()),
		ChapterID:            r.currentChapterID(),
		TimeRemainingSeconds: remaining,
		ScheduledStartAt:     scheduledStartAt,
	})
}

func (r *Runner) sendWaiting(now time.Time) {
	r.send(typeAttemptWaiting, WaitingPayload{
		ScheduledStartAt: *r.scheduledStartAt,
		StartsInSeconds:  clampSeconds(r.scheduledStartAt.Sub(now)),
	})
}

func (r *Runner) sendStarted() {
	r.send(typeAttemptStarted, StartedPayload{StartedAt: *r.startedAt})
}

func (r *Runner) sendTick(remaining time.Duration) {
	r.send(typeTick, TickPayload{TimeRemainingSeconds: clampSeconds(remaining)})
}

func (r *Runner) sendEnded(reason string) {
	r.send(typeAttemptEnded, EndedPayload{Reason: reason})
}

func (r *Runner) sendRejected() {
	r.send(typeChapterRejected, ChapterRejectedPayload{Reason: reasonOutOfOrder})
}

func clampSeconds(d time.Duration) int {
	if d < 0 {
		return 0
	}
	return int(d.Seconds())
}

// ---- inbound messages / public API -------------------------------------

// Connect registers ws as this attempt's one live viewer and sends the
// initial handshake frames. It blocks until that handshake has been
// processed by the Runner's own goroutine.
func (r *Runner) Connect(ws *websocket.Conn) {
	reply := make(chan struct{})
	r.submit(&connectCmd{conn: ws, reply: reply}, reply)
}

// Leave marks the attempt aborted, the same as a leaveCmd arriving over the
// socket — used by the REST leave endpoint.
func (r *Runner) Leave() {
	reply := make(chan struct{})
	r.submit(&leaveCmd{reply: reply}, reply)
}

// HandleMessage parses one inbound WS frame and forwards it as a command.
// Called from the hub's read-pump goroutine.
func (r *Runner) HandleMessage(data []byte) {
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		r.send(typeError, ErrorPayload{Message: "malformed message"})
		return
	}
	switch env.Type {
	case typeChapterAdvance:
		var p ChapterAdvancePayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			r.send(typeError, ErrorPayload{Message: "malformed chapter.advance payload"})
			return
		}
		reply := make(chan struct{})
		r.submit(&advanceChapterCmd{chapterID: p.ChapterID, reply: reply}, reply)
	case typeLeave:
		r.Leave()
	default:
		r.send(typeError, ErrorPayload{Message: "unknown message type: " + env.Type})
	}
}

// submit sends cmd to the Runner's loop and waits for it to be processed,
// giving up if the Runner has already ended (and so will never read
// commands or close reply again).
func (r *Runner) submit(cmd command, reply chan struct{}) {
	select {
	case r.commands <- cmd:
	case <-r.ctx.Done():
		return
	}
	select {
	case <-reply:
	case <-r.ctx.Done():
	}
}

// Snapshot returns the Runner's last-published state without going through
// the commands channel — the one read that bypasses it, safe because it
// only ever reads an atomically-published copy (see atomicSnapshot).
func (r *Runner) Snapshot() Snapshot {
	return r.snapshot.load()
}

func (r *Runner) publishSnapshot() {
	now := time.Now()
	snap := Snapshot{Status: r.statusFor(), ChapterID: r.currentChapterID()}
	switch r.phase {
	case phaseWaiting:
		snap.TimeRemainingSeconds = r.timeLimitMinutes * 60
		t := *r.scheduledStartAt
		snap.ScheduledStartAt = &t
	case phaseRunning:
		snap.TimeRemainingSeconds = clampSeconds(r.deadline.Sub(now))
	}
	r.snapshot.store(snap)
}
