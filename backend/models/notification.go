package models

import "time"

// Notification is one event surfaced to a user, e.g. a solution submission.
// It is private: only the user it was created for can ever read it.
type Notification struct {
	ID        int64      `json:"id" db:"id" doc:"Unique database identifier"`
	Title     string     `json:"title" db:"title" doc:"Short summary"`
	Detail    string     `json:"detail" db:"detail" doc:"Longer description"`
	ReadAt    *time.Time `json:"readAt" db:"read_at" doc:"When the notification was marked read; null while unread"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at" doc:"Creation time"`
}
