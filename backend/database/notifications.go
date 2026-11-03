package database

import (
	"context"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// CreateNotification stores a notification for the user and returns it, ready
// to be pushed over the websocket.
func CreateNotification(ctx context.Context, db *sqlx.DB, userID int64, title, detail string) (models.Notification, error) {
	var n models.Notification
	err := db.GetContext(ctx, &n, `
		INSERT INTO notifications (user_id, title, detail)
		VALUES ($1, $2, $3)
		RETURNING id, title, detail, created_at`, userID, title, detail)
	return n, err
}

// ListNotifications returns the user's notifications, newest first.
func ListNotifications(ctx context.Context, db *sqlx.DB, userID int64, limit int) ([]models.Notification, error) {
	out := []models.Notification{}
	err := db.SelectContext(ctx, &out, `
		SELECT id, title, detail, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2`, userID, limit)
	return out, err
}
