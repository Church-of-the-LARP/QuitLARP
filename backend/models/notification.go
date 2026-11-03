package models

import "time"

// Notification is one entry of the caller's dashboard notification feed.
type Notification struct {
	ID        int64     `json:"id" db:"id" doc:"Unique database identifier"`
	Title     string    `json:"title" db:"title" example:"You were invited" doc:"Short headline"`
	Detail    string    `json:"detail" db:"detail" doc:"Longer description"`
	CreatedAt time.Time `json:"createdAt" db:"created_at" doc:"When the notification was created"`
}
