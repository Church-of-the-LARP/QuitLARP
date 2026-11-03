package sessions

import "errors"

// AdminSnapshot returns the sessions currently tracked by the manager.
func (m *Manager) AdminSnapshot() []*Session {
	return m.snapshot()
}

// AdminAbandon ends a live session by its opaque session ID.
func (m *Manager) AdminAbandon(id string) error {
	for _, session := range m.snapshot() {
		if session.ID == id {
			return m.Abandon(session)
		}
	}
	return errors.New("session not found")
}
