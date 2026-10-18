// Package websocket holds the shared connection plumbing (users, registry,
// read/write pumps). Feature modules such as notifications build on Hub.Send.
package websocket

import (
	"net/http"
	"sync"
	"time"

	gws "github.com/gorilla/websocket"
)

var (
	SendBuffer = 32

	WriteWait  = 10 * time.Second
	PongWait   = 60 * time.Second
	PingPeriod = 50 * time.Second
)

type client struct {
	user int64
	conn *gws.Conn

	send chan []byte
	done chan struct{}
	once sync.Once
}

func (c *client) close() {
	c.once.Do(func() { close(c.done) })
}

// Hub tracks every open connection by user. A user may have several (tabs,
// devices).
type Hub struct {
	// Authenticate resolves the user behind an upgrade request; ok=false
	// rejects the connection with 401.
	Authenticate func(r *http.Request) (userID int64, ok bool)
	Upgrader     gws.Upgrader

	mu    sync.RWMutex
	users map[int64]map[*client]struct{}
}

func NewHub(authenticate func(r *http.Request) (int64, bool)) *Hub {
	return &Hub{
		Authenticate: authenticate,
		users:        make(map[int64]map[*client]struct{}),
	}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.users[c.user] == nil {
		h.users[c.user] = make(map[*client]struct{})
	}
	h.users[c.user][c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	if set, ok := h.users[c.user]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.users, c.user)
		}
	}
	h.mu.Unlock()

	c.close()
}

// Send queues payload on all of a user's connections and returns how many
// received it. Connections whose buffer is full are dropped.
func (h *Hub) Send(userID int64, payload []byte) int {
	h.mu.RLock()
	targets := make([]*client, 0, len(h.users[userID]))
	for c := range h.users[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	sent := 0
	for _, c := range targets {
		select {
		case c.send <- payload:
			sent++
		case <-c.done:
		default:
			h.remove(c)
		}
	}
	return sent
}

// IsOnline reports whether the user has at least one open connection.
func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users[userID]) > 0
}

// ServeHTTP upgrades the request and blocks until the connection closes.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, ok := h.Authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &client{
		user: user,
		conn: conn,
		send: make(chan []byte, SendBuffer),
		done: make(chan struct{}),
	}
	h.add(c)

	go c.writePump()
	c.readLoop(h)
}

func (c *client) readLoop(h *Hub) {
	defer func() {
		h.remove(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(PongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(PongWait))
	})

	for {
		if _, _, err := c.conn.NextReader(); err != nil {
			return
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.done:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			c.conn.WriteMessage(gws.CloseMessage,
				gws.FormatCloseMessage(gws.CloseNormalClosure, ""))
			return

		case payload := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(gws.TextMessage, payload); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(gws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
