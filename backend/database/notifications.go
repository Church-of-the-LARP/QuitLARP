package database

import (
	"context"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

const notificationColumns = `id, title, detail, read_at, created_at`

// CreateNotification records one event for a user.
func CreateNotification(ctx context.Context, db *sqlx.DB, userID int64, title, detail string) error {
	_, err := db.ExecContext(ctx,
		"INSERT INTO notifications (user_id, title, detail) VALUES ($1, $2, $3)",
		userID, title, detail)
	return err
}

// ListNotificationsByUser returns a user's notifications, newest first.
func ListNotificationsByUser(ctx context.Context, db *sqlx.DB, userID int64, limit int) ([]models.Notification, error) {
	out := []models.Notification{}
	err := db.SelectContext(ctx, &out,
		"SELECT "+notificationColumns+` FROM notifications
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, userID, limit)
	return out, err
}

// MarkAllNotificationsRead marks every unread notification of a user read.
func MarkAllNotificationsRead(ctx context.Context, db *sqlx.DB, userID int64) error {
	_, err := db.ExecContext(ctx,
		"UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL",
		userID)
	return err
}
