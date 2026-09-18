package runtime

import (
	"context"
	"sync"

	"github.com/jmoiron/sqlx"
)

// AttemptManager is the process-wide registry of live attempts: at most one
// Runner goroutine per attempt id, for as long as this process is up.
//
// Restart story (Definition of done, TASK.md): this map is purely
// in-memory. If the server restarts, every live Runner goroutine is gone
// and every connected browser's socket drops — that in-flight state is
// lost, and is an accepted tradeoff for this project. Nothing else is
// lost: every persisted field a Runner depends on (attempt.started_at,
// attempt.current_chapter_id, the invite's scheduled_start_at) is written
// to the database as it changes, not just at the end, so newRunner's phase
// derivation reconstructs the correct phase/deadline/chapter cursor from
// those rows alone the next time something connects — a scheduled attempt
// resumes waiting or running (or is found already expired) exactly as if
// the server had never restarted. The only real loss is the tiny window of
// events between whatever a client last saw and the moment they reconnect.
type AttemptManager struct {
	ctx context.Context
	db  *sqlx.DB

	mu      sync.RWMutex
	runners map[int64]*Runner
}

// NewAttemptManager builds an AttemptManager. ctx bounds the lifetime of
// every Runner it ever starts (cancel it to shut them all down).
func NewAttemptManager(ctx context.Context, db *sqlx.DB) *AttemptManager {
	return &AttemptManager{ctx: ctx, db: db, runners: make(map[int64]*Runner)}
}

// Get returns the live Runner for an attempt, if any, without creating one.
func (m *AttemptManager) Get(id int64) (*Runner, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runners[id]
	return r, ok
}

// StartOrGet returns the live Runner for id, creating (and starting) one if
// none exists yet. check-lock-check: an RLock-guarded lookup first, and
// only on a miss does it take the write Lock and check again before
// creating — closing the race where two callers both miss the read-lock
// check and would otherwise each spin up their own Runner for the same
// attempt. This is the only thing that ever starts an attempt's clock.
func (m *AttemptManager) StartOrGet(ctx context.Context, id int64) (*Runner, error) {
	m.mu.RLock()
	if r, ok := m.runners[id]; ok {
		m.mu.RUnlock()
		return r, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.runners[id]; ok {
		return r, nil
	}

	r, err := newRunner(ctx, m, id)
	if err != nil {
		return nil, err
	}
	m.runners[id] = r
	go r.loop()
	return r, nil
}

// remove drops a finished Runner from the map; called once by the Runner's
// own loop as it exits, so the next connect starts a fresh one.
func (m *AttemptManager) remove(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.runners, id)
}
