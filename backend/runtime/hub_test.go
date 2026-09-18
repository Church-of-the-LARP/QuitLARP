package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestHubReplacesConnection: a second WS connection replaces the first —
// the first is forcibly closed rather than both staying live.
func TestHubReplacesConnection(t *testing.T) {
	h := newHub()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		h.set(conn, func([]byte) {})
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[len("http"):]
	ctx := context.Background()

	connA, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial A: %v", err)
	}
	defer connA.CloseNow()

	waitUntil(t, time.Second, func() bool { return h.disconnected() != nil })

	connB, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial B: %v", err)
	}
	defer connB.CloseNow()

	// A should be forcibly closed now that B has replaced it.
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, _, err := connA.Read(readCtx); err == nil {
		t.Fatalf("expected connection A to be closed after B connected")
	}

	// B should still be the live viewer: a hub.send reaches it.
	waitUntil(t, time.Second, func() bool { return h.disconnected() != nil })
	h.send([]byte("hello"))

	readCtx2, cancel2 := context.WithTimeout(ctx, 2*time.Second)
	defer cancel2()
	_, data, err := connB.Read(readCtx2)
	if err != nil {
		t.Fatalf("expected B to receive a message: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected message on B: %s", data)
	}
}

func waitUntil(t *testing.T, within time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", within)
}
