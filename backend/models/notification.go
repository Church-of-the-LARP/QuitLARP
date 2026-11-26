package models

import "time"

// Notification is one message stored for a user. Notifications are created by
// server-side events (invitations, interview creation, submitted results) and
// read back over the REST API; there is no push or realtime channel.
type Notification struct {
	ID        int64      `json:"id" db:"id" doc:"Unique notification id"`
	Title     string     `json:"title" db:"title" doc:"Short notification title"`
	Detail    string     `json:"detail" db:"detail" doc:"Human-readable notification body"`
	ReadAt    *time.Time `json:"readAt,omitempty" db:"read_at" doc:"When the user read the notification; null while unread"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at" doc:"Creation time"`
}
