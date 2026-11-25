// Package collab keeps the live, uncommitted editing state of event attempts.
// Browser editors exchange small text operations through this package; nothing
// here is visible to the deployed app until a participant explicitly commits
// the workspace to the attempt's git repository.
package collab

import (
	"errors"
	"sort"
	"sync"
	"unicode/utf16"
)

// Errors returned to the HTTP layer.
var (
	ErrNotOpen = errors.New("file is not open in this workspace")
	ErrResync  = errors.New("the client is too far behind; reload the file")
	ErrInvalid = errors.New("invalid edit")
)

// maxInsertBytes caps one operation so a client cannot post an arbitrary
// amount of text in a single edit.
const maxInsertBytes = 256 << 10

// maxHistory bounds the operation history used to transform concurrent
// edits. A client further behind than this is told to reload.
const maxHistory = 512

// User is a connected editor.
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// Doc is a document snapshot with its current server version.
type Doc struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Version int    `json:"version"`
	Dirty   bool   `json:"dirty"`
}

// Op is one text operation. Offsets and DeleteCount are in UTF-16 code units,
// matching JavaScript string indices (and therefore Monaco).
type Op struct {
	Path        string `json:"path"`
	Offset      int    `json:"offset"`
	DeleteCount int    `json:"deleteCount"`
	Insert      string `json:"insert"`
	BaseVersion int    `json:"baseVersion"`
	Version     int    `json:"version"`
	ClientID    string `json:"clientId"`
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
}

// Message is one server-sent event.
type Message struct {
	Type   string `json:"type"` // snapshot | op | presence | committed | reload
	Docs   []Doc  `json:"docs,omitempty"`
	Op     *Op    `json:"op,omitempty"`
	Users  []User `json:"users,omitempty"`
	Dirty  bool   `json:"dirty,omitempty"`
	Commit string `json:"commit,omitempty"`
}

// Manager owns every live room of the process.
type Manager struct {
	mu    sync.Mutex
	rooms map[int64]*room
}

// NewManager builds an empty collaboration manager.
func NewManager() *Manager {
	return &Manager{rooms: map[int64]*room{}}
}

// room is the live state of one attempt workspace.
type room struct {
	id      int64
	mu      sync.Mutex
	loaded  bool
	docs    map[string]*doc
	deleted map[string]bool
	subs    map[int64]*subscriber
	nextSub int64
}

type subscriber struct {
	user User
	ch   chan Message
	dead bool
}

type doc struct {
	content string
	version int
	history []historyOp
	dirty   bool
}

type historyOp struct {
	version     int
	offset      int
	deleteCount int
	insertUnits int
}

// LoadWorkspace initializes a room from git main on first access. Later calls
// keep the live uncommitted edits; use ReplaceWorkspace after an external push.
func (m *Manager) LoadWorkspace(attemptID int64, files map[string]string) []Doc {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.loaded {
		r.loaded = true
		for path, content := range files {
			r.docs[path] = &doc{content: content}
		}
	}
	return snapshotLocked(r)
}

// ReplaceWorkspace discards the live edits and reloads the room from git. It
// is used after an out-of-band push, which is authoritative.
func (m *Manager) ReplaceWorkspace(attemptID int64, files map[string]string) {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loaded = true
	r.docs = map[string]*doc{}
	r.deleted = map[string]bool{}
	for path, content := range files {
		r.docs[path] = &doc{content: content}
	}
	r.broadcastLocked(Message{Type: "reload"})
}

// Snapshot returns every open document of a workspace.
func (m *Manager) Snapshot(attemptID int64) []Doc {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	return snapshotLocked(r)
}

// Doc returns the live state of one open document.
func (m *Manager) Doc(attemptID int64, path string) (Doc, bool) {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.docs[path]
	if !ok {
		return Doc{}, false
	}
	return docView(path, d), true
}

// Loaded reports whether a room has been initialized from git.
func (m *Manager) Loaded(attemptID int64) bool {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loaded
}

// Dirty reports whether the room has edits that are not committed to main.
func (m *Manager) Dirty(attemptID int64) bool {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	return dirtyLocked(r)
}

// DirtyDocs returns the documents that changed since the last commit.
func (m *Manager) DirtyDocs(attemptID int64) []Doc {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Doc{}
	for path, d := range r.docs {
		if d.dirty {
			out = append(out, docView(path, d))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Deleted returns the paths removed in the live editor since the last commit.
func (m *Manager) Deleted(attemptID int64) []string {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.deleted))
	for path := range r.deleted {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// MarkCommitted clears the dirty state after a successful commit and tells
// every browser the new main commit.
func (m *Manager) MarkCommitted(attemptID int64, sha string) {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range r.docs {
		d.dirty = false
	}
	r.deleted = map[string]bool{}
	r.broadcastLocked(Message{Type: "committed", Commit: sha})
}

// Create adds a new empty document to the live workspace without touching git.
func (m *Manager) Create(attemptID int64, path, content string) int {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if d, ok := r.docs[path]; ok {
		return d.version
	}
	delete(r.deleted, path)
	r.docs[path] = &doc{content: content, dirty: true}
	r.broadcastLocked(Message{Type: "snapshot", Docs: snapshotLocked(r), Dirty: true})
	return 0
}

// Remove deletes a document from the live workspace without touching git.
func (m *Manager) Remove(attemptID int64, path string) {
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.docs, path)
	r.deleted[path] = true
	r.broadcastLocked(Message{Type: "snapshot", Docs: snapshotLocked(r), Dirty: dirtyLocked(r)})
}

// Subscribe attaches a browser to an attempt workspace. The returned channel
// carries an initial snapshot, then every operation and presence change.
func (m *Manager) Subscribe(attemptID int64, user User) (<-chan Message, func()) {
	r := m.room(attemptID)
	r.mu.Lock()
	r.nextSub++
	id := r.nextSub
	sub := &subscriber{user: user, ch: make(chan Message, 256)}
	r.subs[id] = sub
	sub.ch <- Message{Type: "snapshot", Docs: snapshotLocked(r), Dirty: dirtyLocked(r)}
	r.broadcastPresenceLocked()
	r.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			r.mu.Lock()
			if _, ok := r.subs[id]; ok {
				delete(r.subs, id)
				close(sub.ch)
				r.broadcastPresenceLocked()
			}
			r.mu.Unlock()
		})
	}
	return sub.ch, unsubscribe
}

// Apply transforms and applies one edit in the live workspace.
func (m *Manager) Apply(attemptID int64, in Op) (Op, error) {
	if in.Path == "" || in.Offset < 0 || in.DeleteCount < 0 || len(in.Insert) > maxInsertBytes {
		return Op{}, ErrInvalid
	}
	r := m.room(attemptID)
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.docs[in.Path]
	if !ok {
		return Op{}, ErrNotOpen
	}
	if in.BaseVersion > d.version {
		return Op{}, ErrResync
	}
	if len(d.history) > 0 && in.BaseVersion < d.history[0].version-1 {
		return Op{}, ErrResync
	}

	op := in
	op.Version = 0
	for _, h := range d.history {
		if h.version <= in.BaseVersion {
			continue
		}
		transform(&op, h)
	}
	if err := apply(&d.content, op.Offset, op.DeleteCount, op.Insert); err != nil {
		return Op{}, err
	}
	d.version++
	d.dirty = true
	op.Version = d.version
	d.history = append(d.history, historyOp{
		version:     d.version,
		offset:      op.Offset,
		deleteCount: op.DeleteCount,
		insertUnits: utf16Len(op.Insert),
	})
	if len(d.history) > maxHistory {
		d.history = d.history[len(d.history)-maxHistory:]
	}
	r.broadcastLocked(Message{Type: "op", Op: &op, Dirty: true})
	return op, nil
}

// room returns the room, creating it when missing.
func (m *Manager) room(id int64) *room {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[id]
	if !ok {
		r = &room{
			id:      id,
			docs:    map[string]*doc{},
			deleted: map[string]bool{},
			subs:    map[int64]*subscriber{},
		}
		m.rooms[id] = r
	}
	return r
}

func snapshotLocked(r *room) []Doc {
	out := make([]Doc, 0, len(r.docs))
	for path, d := range r.docs {
		out = append(out, docView(path, d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func docView(path string, d *doc) Doc {
	return Doc{Path: path, Content: d.content, Version: d.version, Dirty: d.dirty}
}

func dirtyLocked(r *room) bool {
	if len(r.deleted) > 0 {
		return true
	}
	for _, d := range r.docs {
		if d.dirty {
			return true
		}
	}
	return false
}

func (r *room) broadcastLocked(msg Message) {
	for id, sub := range r.subs {
		if sub.dead {
			continue
		}
		select {
		case sub.ch <- msg:
		default:
			sub.dead = true
			close(sub.ch)
			delete(r.subs, id)
		}
	}
}

func (r *room) broadcastPresenceLocked() {
	users := make([]User, 0, len(r.subs))
	seen := map[int64]bool{}
	for _, sub := range r.subs {
		if sub.dead || seen[sub.user.ID] {
			continue
		}
		seen[sub.user.ID] = true
		users = append(users, sub.user)
	}
	r.broadcastLocked(Message{Type: "presence", Users: users})
}

// transform adjusts op's offset for one operation that happened since the
// client's base version.
func transform(op *Op, h historyOp) {
	end := op.Offset + op.DeleteCount
	switch {
	case h.offset+h.deleteCount <= op.Offset:
		op.Offset += h.insertUnits - h.deleteCount
	case h.offset >= end:
		// The intervening edit is after ours; nothing to do.
	default:
		// Overlapping edits cannot be merged safely without a CRDT; move
		// ours after the intervening insertion instead of corrupting text.
		op.Offset = h.offset + h.insertUnits
		op.DeleteCount = 0
	}
}

// apply splices an operation into content using UTF-16 indices.
func apply(content *string, offset, deleteCount int, insert string) error {
	units := utf16.Encode([]rune(*content))
	if offset > len(units) || offset+deleteCount > len(units) {
		return ErrInvalid
	}
	out := make([]uint16, 0, len(units)-deleteCount+utf16Len(insert))
	out = append(out, units[:offset]...)
	out = append(out, utf16.Encode([]rune(insert))...)
	out = append(out, units[offset+deleteCount:]...)
	*content = string(utf16.Decode(out))
	return nil
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}
