package database_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"backend/database"
	"backend/internal/testfixtures"
)

// TestAcceptInviteConcurrencyOneWinner exercises the concurrent-accept path
// TASK.md calls out: two acceptors racing the last slot on an invite with
// maxUses 1 must not both succeed. AcceptInvite serializes this with a real
// transaction + SELECT ... FOR UPDATE rather than the literal
// single-statement UPDATE ... RETURNING sketched in TASK.md, but the
// guarantee under test is the same: Postgres, not a Go mutex, decides the
// winner.
func TestAcceptInviteConcurrencyOneWinner(t *testing.T) {
	db := testfixtures.DB(t)
	ctx := context.Background()

	host := testfixtures.User(t, db)
	assessmentID := testfixtures.Assessment(t, db, host, 30)
	maxUses := 1
	invite, err := database.CreateInvite(ctx, db, assessmentID, host, time.Now().Add(time.Hour), nil, &maxUses)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	userA := testfixtures.User(t, db)
	userB := testfixtures.User(t, db)

	errs := make([]error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, errs[0] = database.AcceptInvite(ctx, db, invite.InviteCode, userA)
	}()
	go func() {
		defer wg.Done()
		_, errs[1] = database.AcceptInvite(ctx, db, invite.InviteCode, userB)
	}()
	wg.Wait()

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one acceptor to succeed, got %d (errs=%v)", successes, errs)
	}

	got, err := database.GetInvite(ctx, db, invite.ID)
	if err != nil {
		t.Fatalf("get invite: %v", err)
	}
	if got.UsesCount != 1 {
		t.Fatalf("expected uses_count to land on exactly 1, got %d", got.UsesCount)
	}
}

// TestAcceptInviteReacceptReturnsExistingAttempt documents (per TASK.md's
// "Design decision") that re-accepting an invite you're already registered
// under returns your existing attempt instead of erroring or double-using
// a slot.
func TestAcceptInviteReacceptReturnsExistingAttempt(t *testing.T) {
	db := testfixtures.DB(t)
	ctx := context.Background()

	host := testfixtures.User(t, db)
	assessmentID := testfixtures.Assessment(t, db, host, 30)
	maxUses := 1
	invite, err := database.CreateInvite(ctx, db, assessmentID, host, time.Now().Add(time.Hour), nil, &maxUses)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	user := testfixtures.User(t, db)
	first, err := database.AcceptInvite(ctx, db, invite.InviteCode, user)
	if err != nil {
		t.Fatalf("first accept: %v", err)
	}
	second, err := database.AcceptInvite(ctx, db, invite.InviteCode, user)
	if err != nil {
		t.Fatalf("second accept: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected the same attempt back, got %d and %d", first.ID, second.ID)
	}

	got, err := database.GetInvite(ctx, db, invite.ID)
	if err != nil {
		t.Fatalf("get invite: %v", err)
	}
	if got.UsesCount != 1 {
		t.Fatalf("re-accepting should not consume a second use, got uses_count=%d", got.UsesCount)
	}
}

// TestAcceptInviteReacceptAfterLeaveDoesNotDoubleCount covers the case
// TestAcceptInviteReacceptReturnsExistingAttempt doesn't: uses_count is
// documented as counting distinct acceptors, so a user who left and comes
// back for another sitting must not be charged against max_uses twice.
func TestAcceptInviteReacceptAfterLeaveDoesNotDoubleCount(t *testing.T) {
	db := testfixtures.DB(t)
	ctx := context.Background()

	host := testfixtures.User(t, db)
	assessmentID := testfixtures.Assessment(t, db, host, 30)
	maxUses := 1
	invite, err := database.CreateInvite(ctx, db, assessmentID, host, time.Now().Add(time.Hour), nil, &maxUses)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	user := testfixtures.User(t, db)
	first, err := database.AcceptInvite(ctx, db, invite.InviteCode, user)
	if err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if _, err := database.LeaveAttempt(ctx, db, first.ID); err != nil {
		t.Fatalf("leave attempt: %v", err)
	}

	second, err := database.AcceptInvite(ctx, db, invite.InviteCode, user)
	if err != nil {
		t.Fatalf("re-accept after leaving: %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("expected a new attempt after leaving, got the same one back")
	}

	got, err := database.GetInvite(ctx, db, invite.ID)
	if err != nil {
		t.Fatalf("get invite: %v", err)
	}
	if got.UsesCount != 1 {
		t.Fatalf("re-accepting after leaving should not consume a second use, got uses_count=%d", got.UsesCount)
	}
}
