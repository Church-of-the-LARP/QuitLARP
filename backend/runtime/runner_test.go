package runtime

import (
	"context"
	"testing"
	"time"

	"backend/database"
	"backend/internal/testfixtures"
	"backend/models"
)

func waitForStatus(t *testing.T, r *Runner, want models.AttemptStatus, within time.Duration) Snapshot {
	t.Helper()
	deadline := time.Now().Add(within)
	var snap Snapshot
	for time.Now().Before(deadline) {
		snap = r.Snapshot()
		if snap.Status == want {
			return snap
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for status %q, last snapshot: %+v", want, snap)
	return snap
}

// TestSoloRunnerTimesOutWithoutConnecting: a solo Runner self-terminates on
// timeout even though no client ever connects (loop() drives the deadline
// off the ticker, not off any connection).
func TestSoloRunnerTimesOutWithoutConnecting(t *testing.T) {
	db := testfixtures.DB(t)
	ctx := context.Background()

	userID := testfixtures.User(t, db)
	assessmentID := testfixtures.Assessment(t, db, userID, 1)
	startedAt := time.Now().Add(-58 * time.Second) // 1-minute limit, ~2s left
	attemptID := testfixtures.Attempt(t, db, assessmentID, userID, nil, 1, &startedAt)

	m := NewAttemptManager(ctx, db)
	r, err := newRunner(ctx, m, attemptID)
	if err != nil {
		t.Fatalf("newRunner: %v", err)
	}
	if r.phase != phaseRunning {
		t.Fatalf("expected phaseRunning immediately for an already-started solo attempt, got %v", r.phase)
	}
	go r.loop()

	waitForStatus(t, r, models.AttemptEnded, 5*time.Second)

	attempt, err := database.GetAttempt(ctx, db, attemptID)
	if err != nil {
		t.Fatalf("get attempt: %v", err)
	}
	if attempt.Status != models.AttemptEnded {
		t.Fatalf("expected the attempt row to be marked ended, got %v", attempt.Status)
	}
}

// TestScheduledRunnerFlipsToRunningAtScheduledInstant: a scheduled Runner
// reports phaseWaiting immediately on construction, then flips to
// phaseRunning on its own — nothing external prompts it.
func TestScheduledRunnerFlipsToRunningAtScheduledInstant(t *testing.T) {
	db := testfixtures.DB(t)
	ctx := context.Background()

	hostID := testfixtures.User(t, db)
	userID := testfixtures.User(t, db)
	assessmentID := testfixtures.Assessment(t, db, hostID, 5)
	scheduledAt := time.Now().Add(2 * time.Second)
	invite := testfixtures.Invite(t, db, assessmentID, hostID, scheduledAt, nil, nil)
	inviteID := invite.ID
	attemptID := testfixtures.Attempt(t, db, assessmentID, userID, &inviteID, 5, nil)

	m := NewAttemptManager(ctx, db)
	r, err := newRunner(ctx, m, attemptID)
	if err != nil {
		t.Fatalf("newRunner: %v", err)
	}
	if r.phase != phaseWaiting {
		t.Fatalf("expected phaseWaiting immediately, got %v", r.phase)
	}
	if snap := r.Snapshot(); snap.Status != models.AttemptPending {
		t.Fatalf("expected pending status while waiting, got %v", snap.Status)
	}
	go r.loop()

	waitForStatus(t, r, models.AttemptRunning, 5*time.Second)
}

// TestScheduledRunnerAlreadyExpiredEndsImmediately: an invite whose window
// has already fully closed ends the attempt right on construction — no
// tick, no waiting for the ticker to fire.
func TestScheduledRunnerAlreadyExpiredEndsImmediately(t *testing.T) {
	db := testfixtures.DB(t)
	ctx := context.Background()

	hostID := testfixtures.User(t, db)
	userID := testfixtures.User(t, db)
	assessmentID := testfixtures.Assessment(t, db, hostID, 1)
	scheduledAt := time.Now().Add(-10 * time.Minute) // window closed long ago
	invite := testfixtures.Invite(t, db, assessmentID, hostID, scheduledAt, nil, nil)
	inviteID := invite.ID
	attemptID := testfixtures.Attempt(t, db, assessmentID, userID, &inviteID, 1, nil)

	m := NewAttemptManager(ctx, db)
	start := time.Now()
	r, err := newRunner(ctx, m, attemptID)
	if err != nil {
		t.Fatalf("newRunner: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("newRunner took %v to notice the window had closed; expected an immediate decision", elapsed)
	}
	if r.phase != phaseEnded || !r.replayEnded || r.initialEndReason != reasonExpired {
		t.Fatalf("expected an immediately-expired runner, got phase=%v replayEnded=%v reason=%q",
			r.phase, r.replayEnded, r.initialEndReason)
	}

	attempt, err := database.GetAttempt(ctx, db, attemptID)
	if err != nil {
		t.Fatalf("get attempt: %v", err)
	}
	if attempt.Status != models.AttemptEnded {
		t.Fatalf("expected the attempt row to already be marked ended, got %v", attempt.Status)
	}
}

// TestAdvanceChapterRejectsOutOfOrder: skipping ahead is rejected, not
// silently accepted; the real next chapter still works afterwards.
func TestAdvanceChapterRejectsOutOfOrder(t *testing.T) {
	db := testfixtures.DB(t)
	ctx := context.Background()

	userID := testfixtures.User(t, db)
	assessmentID := testfixtures.Assessment(t, db, userID, 30)
	chapterIDs := testfixtures.Chapters(t, db, assessmentID, 3)
	startedAt := time.Now()
	attemptID := testfixtures.Attempt(t, db, assessmentID, userID, nil, 30, &startedAt)

	m := NewAttemptManager(ctx, db)
	r, err := newRunner(ctx, m, attemptID)
	if err != nil {
		t.Fatalf("newRunner: %v", err)
	}
	go r.loop()
	t.Cleanup(r.cancel)

	if snap := r.Snapshot(); snap.ChapterID == nil || *snap.ChapterID != chapterIDs[0] {
		t.Fatalf("expected to start on chapter 1, got %v", snap.ChapterID)
	}

	skipReply := make(chan struct{})
	r.submit(&advanceChapterCmd{chapterID: chapterIDs[2], reply: skipReply}, skipReply)

	if snap := r.Snapshot(); snap.ChapterID == nil || *snap.ChapterID != chapterIDs[0] {
		t.Fatalf("skip-ahead to chapter 3 should have been rejected, chapter cursor is now %v", snap.ChapterID)
	}

	nextReply := make(chan struct{})
	r.submit(&advanceChapterCmd{chapterID: chapterIDs[1], reply: nextReply}, nextReply)

	if snap := r.Snapshot(); snap.ChapterID == nil || *snap.ChapterID != chapterIDs[1] {
		t.Fatalf("expected the immediate next chapter to be accepted, chapter cursor is %v", snap.ChapterID)
	}
}
