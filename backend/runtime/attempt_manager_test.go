package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"backend/internal/testfixtures"
)

// TestStartOrGetConcurrencyCreatesOneRunner drives many goroutines at
// StartOrGet for the same attempt id at once (run with -race -count=20 per
// TASK.md) to exercise the check-lock-check race window directly: exactly
// one Runner must ever be created.
func TestStartOrGetConcurrencyCreatesOneRunner(t *testing.T) {
	db := testfixtures.DB(t)
	ctx := context.Background()

	userID := testfixtures.User(t, db)
	assessmentID := testfixtures.Assessment(t, db, userID, 30)
	startedAt := time.Now()
	attemptID := testfixtures.Attempt(t, db, assessmentID, userID, nil, 30, &startedAt)

	m := NewAttemptManager(ctx, db)
	t.Cleanup(func() {
		if r, ok := m.Get(attemptID); ok {
			r.cancel()
		}
	})

	const n = 20
	results := make([]*Runner, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = m.StartOrGet(ctx, attemptID)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("StartOrGet[%d]: %v", i, err)
		}
	}
	first := results[0]
	for i, r := range results {
		if r != first {
			t.Fatalf("StartOrGet[%d] returned a different *Runner than [0] — more than one Runner was created", i)
		}
	}
}
