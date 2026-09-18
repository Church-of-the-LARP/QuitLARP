package runtime

import (
	"context"
	"sync"

	"github.com/coder/websocket"
)

// hub owns the (at most one) live WebSocket viewer of an attempt. Exactly
// two goroutines run per connection: a write-pump (the only one calling
// conn.Write) and a read-pump (the only one calling conn.Read). hub.set
// replaces the current viewer rather than adding a second one — the
// previous connection is forcibly closed.
type hub struct {
	mu   sync.Mutex
	conn *connection
}

// connection is one registered viewer. Its ctx is its own — independent of
// the Runner's ctx — precisely so that the Runner ending (and cancelling
// its own ctx as it exits) can never race the write-pump's final flush of
// the attempt.ended frame: only terminate() ever cancels this ctx, and
// terminate() is what runs *after* a graceful close has drained send.
type connection struct {
	conn   *websocket.Conn
	send   chan []byte
	closed chan struct{}
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
}

func newHub() *hub {
	return &hub{}
}

// terminate force-closes the connection exactly once, however it was
// reached (a read error, being replaced, or a graceful close draining out).
func (c *connection) terminate() {
	c.once.Do(func() {
		close(c.closed)
		c.cancel()
		c.conn.CloseNow()
	})
}

// writePump drains send until it is closed (graceful end) or a write fails
// (dead connection), and is the only goroutine that ever calls conn.Write.
func (c *connection) writePump() {
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.Close(websocket.StatusNormalClosure, "")
				c.terminate()
				return
			}
			if err := c.conn.Write(c.ctx, websocket.MessageText, msg); err != nil {
				c.terminate()
				return
			}
		case <-c.closed:
			return
		}
	}
}

// readPump is the only goroutine that ever calls conn.Read. On any error it
// just returns — a dropped connection is never turned into a synthesized
// leaveCmd; the Runner notices via hub.disconnected() instead.
func (c *connection) readPump(onMessage func([]byte)) {
	for {
		_, data, err := c.conn.Read(c.ctx)
		if err != nil {
			c.terminate()
			return
		}
		onMessage(data)
	}
}

// set registers ws as the hub's one viewer, replacing (and force-closing)
// any previous connection. onMessage is invoked from the read-pump
// goroutine for every inbound frame.
func (h *hub) set(ws *websocket.Conn, onMessage func([]byte)) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &connection{
		conn:   ws,
		send:   make(chan []byte, 16),
		closed: make(chan struct{}),
		ctx:    ctx,
		cancel: cancel,
	}

	h.mu.Lock()
	prev := h.conn
	h.conn = c
	h.mu.Unlock()

	if prev != nil {
		prev.terminate()
	}

	go c.writePump()
	go c.readPump(onMessage)
}

// send is a non-blocking push to the current viewer, if any — a slow or
// absent client just drops the frame rather than stalling the Runner.
func (h *hub) send(msg []byte) {
	h.mu.Lock()
	c := h.conn
	h.mu.Unlock()
	if c == nil {
		return
	}
	select {
	case c.send <- msg:
	default:
	}
}

// closeCurrent gracefully closes the current viewer's connection — used
// right after the final attempt.ended frame has been queued.
func (h *hub) closeCurrent() {
	h.mu.Lock()
	c := h.conn
	h.conn = nil
	h.mu.Unlock()
	if c == nil {
		return
	}
	close(c.send)
}

// disconnected returns a channel that closes when the current viewer's
// connection drops; nil (blocks forever in a select) when there is none.
func (h *hub) disconnected() <-chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conn == nil {
		return nil
	}
	return h.conn.closed
}
