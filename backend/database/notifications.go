package database

import (
	"context"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// CreateNotification stores one notification for a user.
func CreateNotification(ctx context.Context, db *sqlx.DB, userID int64, title, detail string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO notifications (user_id, title, detail)
		VALUES ($1, $2, $3)`, userID, title, detail)
	return err
}

// ListNotifications returns one page of a user's notifications, newest first.
func ListNotifications(ctx context.Context, db *sqlx.DB, userID int64, limit, offset int) ([]models.Notification, error) {
	list := []models.Notification{}
	err := db.SelectContext(ctx, &list, `
		SELECT id, title, detail, read_at, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	return list, err
}

// InterviewResultTarget is one host that should learn about a participant's
// result in a monitored interview.
type InterviewResultTarget struct {
	EventTitle string `db:"event_title"`
	TaskTitle  string `db:"task_title"`
	HostID     int64  `db:"host_id"`
}

// ListInterviewResultTargets returns the hosts of every live or ended
// monitored interview that contains the assessment and has the solver as a
// member. The same host may appear once per matching interview.
func ListInterviewResultTargets(ctx context.Context, db *sqlx.DB, solverID, assessmentID int64) ([]InterviewResultTarget, error) {
	targets := []InterviewResultTarget{}
	err := db.SelectContext(ctx, &targets, `
		SELECT DISTINCT e.title AS event_title, a.title AS task_title, m.user_id AS host_id
		FROM events e
		JOIN event_members m ON m.event_id = e.id AND m.role = 'host'
		JOIN event_tasks t ON t.event_id = e.id
		JOIN assessments a ON a.id = t.assessment_id
		JOIN event_members participant ON participant.event_id = e.id AND participant.user_id = $1
		WHERE t.assessment_id = $2
		  AND e.monitored
		  AND e.status <> 'draft'
		ORDER BY e.title, m.user_id`, solverID, assessmentID)
	return targets, err
}

// ListEventHostIDs returns the users hosting an event.
func ListEventHostIDs(ctx context.Context, db *sqlx.DB, eventID int64) ([]int64, error) {
	hosts := []int64{}
	err := db.SelectContext(ctx, &hosts, `
		SELECT user_id FROM event_members
		WHERE event_id = $1 AND role = 'host'
		ORDER BY user_id`, eventID)
	return hosts, err
}

// GetEventTaskTitle returns the assessment title attached to one event task.
func GetEventTaskTitle(ctx context.Context, db *sqlx.DB, eventID, taskID int64) (string, error) {
	var title string
	err := db.GetContext(ctx, &title, `
		SELECT a.title
		FROM event_tasks t JOIN assessments a ON a.id = t.assessment_id
		WHERE t.event_id = $1 AND t.id = $2`, eventID, taskID)
	if err != nil {
		return "", mapNotFound(err)
	}
	return title, nil
}
