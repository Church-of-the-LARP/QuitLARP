package collab

import (
	"testing"
	"time"
)

func TestApplyEdits(t *testing.T) {
	m := NewManager()
	m.LoadWorkspace(1, map[string]string{"app.js": "const x = 1;\n"})

	op, err := m.Apply(1, Op{Path: "app.js", Offset: 10, DeleteCount: 1, Insert: "2", BaseVersion: 0})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if op.Version != 1 {
		t.Fatalf("version = %d, want 1", op.Version)
	}
	doc, ok := m.Doc(1, "app.js")
	if !ok {
		t.Fatal("document missing")
	}
	if doc.Content != "const x = 2;\n" || doc.Version != 1 {
		t.Fatalf("doc = %#v", doc)
	}
}

func TestApplyUsesUTF16Offsets(t *testing.T) {
	m := NewManager()
	m.LoadWorkspace(1, map[string]string{"a.txt": "a😀b"})

	// Emoji is two UTF-16 code units, so offset 1 with deleteCount 2 removes it.
	if _, err := m.Apply(1, Op{Path: "a.txt", Offset: 1, DeleteCount: 2, Insert: "z", BaseVersion: 0}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	doc, _ := m.Doc(1, "a.txt")
	if doc.Content != "azb" {
		t.Fatalf("content = %q, want %q", doc.Content, "azb")
	}
}

func TestConcurrentEditsAreTransformed(t *testing.T) {
	m := NewManager()
	m.LoadWorkspace(1, map[string]string{"a.txt": "abc"})

	if _, err := m.Apply(1, Op{Path: "a.txt", Offset: 0, Insert: "X", BaseVersion: 0}); err != nil {
		t.Fatalf("first: %v", err)
	}
	// The second client edited "abc" at offset 1 before seeing X.
	op, err := m.Apply(1, Op{Path: "a.txt", Offset: 1, Insert: "Y", BaseVersion: 0})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	doc, _ := m.Doc(1, "a.txt")
	if doc.Content != "XaYbc" {
		t.Fatalf("content = %q, want %q (offset %d)", doc.Content, "XaYbc", op.Offset)
	}
}

func TestApplyRejectsUnknownFileAndFutureVersion(t *testing.T) {
	m := NewManager()
	m.LoadWorkspace(1, map[string]string{"a.txt": ""})
	if _, err := m.Apply(1, Op{Path: "other.txt", Insert: "x", BaseVersion: 0}); err != ErrNotOpen {
		t.Fatalf("unknown file error = %v", err)
	}
	if _, err := m.Apply(1, Op{Path: "a.txt", Insert: "x", BaseVersion: 9}); err != ErrResync {
		t.Fatalf("future version error = %v", err)
	}
}

func TestSubscribeSendsSnapshotAndOps(t *testing.T) {
	m := NewManager()
	m.LoadWorkspace(7, map[string]string{"index.html": "<h1>hi</h1>"})
	ch, unsubscribe := m.Subscribe(7, User{ID: 4, Username: "mara"})
	defer unsubscribe()

	first := waitMessage(t, ch)
	if first.Type != "snapshot" || len(first.Docs) != 1 || first.Docs[0].Path != "index.html" {
		t.Fatalf("first message = %#v", first)
	}
	if _, err := m.Apply(7, Op{Path: "index.html", Offset: 4, DeleteCount: 2, Insert: "bye", BaseVersion: 0}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	for {
		msg := waitMessage(t, ch)
		if msg.Type == "op" {
			if msg.Op == nil || msg.Op.Version != 1 {
				t.Fatalf("op message = %#v", msg)
			}
			return
		}
	}
}

func waitMessage(t *testing.T, ch <-chan Message) Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for a message")
		return Message{}
	}
}

func TestDirtyStaysUntilCommitted(t *testing.T) {
	m := NewManager()
	m.LoadWorkspace(3, map[string]string{"a.txt": "one"})
	if m.Dirty(3) {
		t.Fatal("fresh workspace is dirty")
	}
	if _, err := m.Apply(3, Op{Path: "a.txt", Offset: 3, Insert: " two", BaseVersion: 0}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !m.Dirty(3) {
		t.Fatal("edit did not mark the workspace dirty")
	}
	// Reloading from git must not overwrite the live draft.
	m.LoadWorkspace(3, map[string]string{"a.txt": "from git"})
	if doc, _ := m.Doc(3, "a.txt"); doc.Content != "one two" {
		t.Fatalf("draft was overwritten: %q", doc.Content)
	}
	m.MarkCommitted(3, "abc")
	if m.Dirty(3) {
		t.Fatal("commit did not clear dirty state")
	}
	if doc, _ := m.Doc(3, "a.txt"); doc.Dirty {
		t.Fatal("document stayed dirty after commit")
	}
}

func TestCreateAndRemoveAreDrafts(t *testing.T) {
	m := NewManager()
	m.LoadWorkspace(4, map[string]string{"a.txt": "one"})
	m.Create(4, "b.txt", "two")
	if _, ok := m.Doc(4, "b.txt"); !ok {
		t.Fatal("created document missing")
	}
	if !m.Dirty(4) {
		t.Fatal("create did not mark dirty")
	}
	m.Remove(4, "a.txt")
	if deleted := m.Deleted(4); len(deleted) != 1 || deleted[0] != "a.txt" {
		t.Fatalf("deleted = %v", deleted)
	}
	m.MarkCommitted(4, "abc")
	if deleted := m.Deleted(4); len(deleted) != 0 {
		t.Fatalf("deleted after commit = %v", deleted)
	}
}
