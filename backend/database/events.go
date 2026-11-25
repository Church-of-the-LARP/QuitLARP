package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// joinCodeAlphabet omits characters that are easy to confuse when someone
// reads a code aloud or from a screen.
const joinCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// eventRow is one row of events joined with its host's username.
type eventRow struct {
	ID           int64              `db:"id"`
	Title        string             `db:"title"`
	Description  string             `db:"description"`
	HostID       *int64             `db:"host_id"`
	HostUsername *string            `db:"host_username"`
	Status       models.EventStatus `db:"status"`
	JoinCode     string             `db:"join_code"`
	StartsAt     *time.Time         `db:"starts_at"`
	EndsAt       *time.Time         `db:"ends_at"`
	CreatedAt    time.Time          `db:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at"`
}

const eventColumns = `
	e.id, e.title, e.description, e.host_id, u.username AS host_username,
	e.status::text, e.join_code, e.starts_at, e.ends_at,
	e.created_at, e.updated_at`

func (r eventRow) host() *models.AuthorRef {
	if r.HostID == nil {
		return nil
	}
	return &models.AuthorRef{ID: *r.HostID, Username: *r.HostUsername}
}

// EventStatusError reports an invalid lifecycle transition.
var EventStatusError = errors.New("invalid event status transition")

// GetEventRow loads the scalar event row. ErrNotFound when it does not exist.
func GetEventRow(ctx context.Context, db *sqlx.DB, id int64) (eventRow, error) {
	var row eventRow
	err := db.GetContext(ctx, &row, `
		SELECT `+eventColumns+`
		FROM events e LEFT JOIN users u ON u.id = e.host_id
		WHERE e.id = $1`, id)
	if err != nil {
		return eventRow{}, mapNotFound(err)
	}
	return row, nil
}

// GetEventMemberRole returns the caller's relationship to an event: host,
// member, or "" for a non-member. An unknown event is ErrNotFound.
func GetEventMemberRole(ctx context.Context, db *sqlx.DB, eventID, userID int64) (models.EventMemberRole, error) {
	row, err := GetEventRow(ctx, db, eventID)
	if err != nil {
		return "", err
	}
	if row.HostID != nil && *row.HostID == userID {
		return models.EventRoleHost, nil
	}
	var role string
	err = db.GetContext(ctx, &role, `
		SELECT role::text FROM event_members
		WHERE event_id = $1 AND user_id = $2`, eventID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return models.EventMemberRole(role), nil
}

// ListUserEvents returns the events the user hosts or joined, newest first.
func ListUserEvents(ctx context.Context, db *sqlx.DB, userID int64) ([]models.EventSummary, error) {
	rows := []struct {
		ID           int64              `db:"id"`
		Title        string             `db:"title"`
		Description  string             `db:"description"`
		HostID       *int64             `db:"host_id"`
		HostUsername *string            `db:"host_username"`
		Status       models.EventStatus `db:"status"`
		JoinCode     string             `db:"join_code"`
		StartsAt     *time.Time         `db:"starts_at"`
		EndsAt       *time.Time         `db:"ends_at"`
		CreatedAt    time.Time          `db:"created_at"`
		UpdatedAt    time.Time          `db:"updated_at"`
		TaskCount    int64              `db:"task_count"`
	}{}
	err := db.SelectContext(ctx, &rows, `
		SELECT `+eventColumns+`,
		       (SELECT COUNT(*) FROM event_tasks t WHERE t.event_id = e.id) AS task_count
		FROM events e
		LEFT JOIN users u ON u.id = e.host_id
		JOIN event_members m ON m.event_id = e.id
		WHERE m.user_id = $1
		ORDER BY e.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.EventSummary, 0, len(rows))
	for _, row := range rows {
		var host *models.AuthorRef
		if row.HostID != nil && row.HostUsername != nil {
			host = &models.AuthorRef{ID: *row.HostID, Username: *row.HostUsername}
		}
		out = append(out, models.EventSummary{
			ID:          row.ID,
			Title:       row.Title,
			Description: row.Description,
			Status:      row.Status,
			Host:        host,
			StartsAt:    row.StartsAt,
			EndsAt:      row.EndsAt,
			TaskCount:   row.TaskCount,
			CreatedAt:   row.CreatedAt,
		})
	}
	return out, nil
}

// CreateEvent inserts an event, its host membership, its unbound tasks and
// their positions in a single transaction. Task ids must refer to unbound
// assessments. Returns the new event id.
func CreateEvent(ctx context.Context, db *sqlx.DB, hostID int64, draft models.EventDraft) (int64, error) {
	taskIDs := dedupeIDs(draft.TaskIDs)
	for _, taskID := range taskIDs {
		var kind string
		if err := db.GetContext(ctx, &kind, "SELECT kind FROM assessments WHERE id = $1", taskID); err != nil {
			return 0, fmt.Errorf("task %d: %w", taskID, mapNotFound(err))
		}
		if kind != models.KindUnbound {
			return 0, fmt.Errorf("task %d is not an unbound task", taskID)
		}
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var id int64
	for range 5 {
		code, err := randomCode(10)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, "SAVEPOINT join_code"); err != nil {
			return 0, err
		}
		err = tx.GetContext(ctx, &id, `
			INSERT INTO events (title, description, host_id, status, join_code, starts_at, ends_at)
			VALUES ($1, $2, $3, 'draft', $4, $5, $6)
			RETURNING id`,
			draft.Title, draft.Description, hostID, code, draft.StartsAt, draft.EndsAt)
		if err == nil {
			if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT join_code"); err != nil {
				return 0, err
			}
			break
		}
		if !IsUniqueViolation(err) {
			return 0, err
		}
		if _, rollbackErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT join_code"); rollbackErr != nil {
			return 0, rollbackErr
		}
	}
	if id == 0 {
		return 0, errors.New("could not allocate a unique join code")
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_members (event_id, user_id, role)
		VALUES ($1, $2, 'host')`, id, hostID); err != nil {
		return 0, err
	}

	for i, taskID := range taskIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_tasks (event_id, assessment_id, position)
			VALUES ($1, $2, $3)`, id, taskID, i+1); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// AddEventMember joins a user to an event. Joining twice is a no-op.
func AddEventMember(ctx context.Context, db *sqlx.DB, eventID, userID int64) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO event_members (event_id, user_id, role)
		VALUES ($1, $2, 'member')
		ON CONFLICT (event_id, user_id) DO NOTHING`, eventID, userID)
	return err
}

// JoinEventByCode adds the user to the event with the given join code and
// returns its id. ErrNotFound when the code matches no event.
func JoinEventByCode(ctx context.Context, db *sqlx.DB, userID int64, code string) (int64, error) {
	var eventID int64
	err := db.GetContext(ctx, &eventID,
		"SELECT id FROM events WHERE UPPER(join_code) = UPPER($1)", strings.TrimSpace(code))
	if err != nil {
		return 0, mapNotFound(err)
	}
	if err := AddEventMember(ctx, db, eventID, userID); err != nil {
		return 0, err
	}
	return eventID, nil
}

// RemoveEventMember removes a participant. The host cannot be removed.
func RemoveEventMember(ctx context.Context, db *sqlx.DB, eventID, userID int64) error {
	res, err := db.ExecContext(ctx, `
		DELETE FROM event_members
		WHERE event_id = $1 AND user_id = $2 AND role <> 'host'`, eventID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListEventMembers returns the members of an event, host first.
func ListEventMembers(ctx context.Context, db *sqlx.DB, eventID int64) ([]models.EventMember, error) {
	members := []models.EventMember{}
	err := db.SelectContext(ctx, &members, `
		SELECT m.user_id, u.username, m.role::text, m.joined_at
		FROM event_members m JOIN users u ON u.id = m.user_id
		WHERE m.event_id = $1
		ORDER BY (m.role = 'host') DESC, LOWER(u.username)`, eventID)
	if err != nil {
		return nil, err
	}
	return members, nil
}

// AttemptRow is the database view of one attempt, used by authorization,
// deployment and git code.
type AttemptRow struct {
	ID               int64      `db:"id"`
	EventID          int64      `db:"event_id"`
	EventTaskID      int64      `db:"event_task_id"`
	AssessmentID     int64      `db:"assessment_id"`
	Name             string     `db:"name"`
	Kind             string     `db:"kind"`
	OwnerID          *int64     `db:"owner_id"`
	OwnerUsername    *string    `db:"owner_username"`
	JoinCode         string     `db:"join_code"`
	RepoPath         string     `db:"repo_path"`
	EventStatus      string     `db:"event_status"`
	PublicID         string     `db:"public_id"`
	CommitSHA        string     `db:"commit_sha"`
	DeploymentStatus string     `db:"deployment_status"`
	DeploymentError  string     `db:"deployment_error"`
	Published        bool       `db:"published"`
	DeployedAt       *time.Time `db:"deployed_at"`
}

const attemptColumns = `
	att.id, att.event_id, att.event_task_id, t.assessment_id, att.name, att.kind::text,
	att.owner_id, o.username AS owner_username, att.join_code, att.repo_path,
	e.status::text AS event_status,
	d.public_id, d.commit_sha, d.status::text AS deployment_status,
	d.error AS deployment_error, d.published, d.deployed_at`

const attemptJoins = `
	FROM event_attempts att
	JOIN event_tasks t ON t.id = att.event_task_id
	JOIN events e ON e.id = att.event_id
	JOIN attempt_deployments d ON d.attempt_id = att.id
	LEFT JOIN users o ON o.id = att.owner_id`

func (r AttemptRow) owner() *models.AuthorRef {
	if r.OwnerID == nil || r.OwnerUsername == nil {
		return nil
	}
	return &models.AuthorRef{ID: *r.OwnerID, Username: *r.OwnerUsername}
}

// GetAttempt loads one attempt by id.
func GetAttempt(ctx context.Context, db *sqlx.DB, id int64) (AttemptRow, error) {
	var row AttemptRow
	err := db.GetContext(ctx, &row, `SELECT `+attemptColumns+attemptJoins+` WHERE att.id = $1`, id)
	if err != nil {
		return AttemptRow{}, mapNotFound(err)
	}
	return row, nil
}

// GetAttemptInEvent loads one attempt scoped to an event.
func GetAttemptInEvent(ctx context.Context, db *sqlx.DB, eventID, attemptID int64) (AttemptRow, error) {
	var row AttemptRow
	err := db.GetContext(ctx, &row, `SELECT `+attemptColumns+attemptJoins+`
		WHERE att.event_id = $1 AND att.id = $2`, eventID, attemptID)
	if err != nil {
		return AttemptRow{}, mapNotFound(err)
	}
	return row, nil
}

// GetAttemptByRepoPath loads one attempt by its workspace repository path.
// It is the authorization and deploy hook for the git server.
func GetAttemptByRepoPath(ctx context.Context, db *sqlx.DB, repoPath string) (AttemptRow, error) {
	var row AttemptRow
	err := db.GetContext(ctx, &row, `SELECT `+attemptColumns+attemptJoins+`
		WHERE att.repo_path = $1`, strings.TrimSuffix(repoPath, ".git"))
	if err != nil {
		return AttemptRow{}, mapNotFound(err)
	}
	return row, nil
}

// GetAttemptMemberRole returns the caller's role in an attempt, or "" when
// they are not a member. An unknown attempt is ErrNotFound.
func GetAttemptMemberRole(ctx context.Context, db *sqlx.DB, attemptID, userID int64) (models.AttemptMemberRole, error) {
	var role string
	err := db.GetContext(ctx, &role, `
		SELECT role::text FROM event_attempt_members
		WHERE attempt_id = $1 AND user_id = $2`, attemptID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := GetAttempt(ctx, db, attemptID); err != nil {
			return "", err
		}
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return models.AttemptMemberRole(role), nil
}

// ListAttemptMembers returns the members of one attempt, owner first.
func ListAttemptMembers(ctx context.Context, db *sqlx.DB, attemptID int64) ([]models.AttemptMember, error) {
	members := []models.AttemptMember{}
	err := db.SelectContext(ctx, &members, `
		SELECT m.user_id, u.username, m.role::text, m.joined_at
		FROM event_attempt_members m JOIN users u ON u.id = m.user_id
		WHERE m.attempt_id = $1
		ORDER BY (m.role = 'owner') DESC, LOWER(u.username)`, attemptID)
	if err != nil {
		return nil, err
	}
	return members, nil
}

// ListAttempts returns the attempts of one event task that the viewer may
// see: hosts see every attempt, participants only the ones they belong to.
func ListAttempts(ctx context.Context, db *sqlx.DB, taskID, viewerID int64, isHost bool, baseURL string) ([]models.EventAttempt, error) {
	rows := []AttemptRow{}
	err := db.SelectContext(ctx, &rows, `SELECT `+attemptColumns+attemptJoins+`
		WHERE att.event_task_id = $1
		  AND ($2::boolean OR EXISTS (
		      SELECT 1 FROM event_attempt_members mm
		      WHERE mm.attempt_id = att.id AND mm.user_id = $3))
		ORDER BY att.created_at, att.id`, taskID, isHost, viewerID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []models.EventAttempt{}, nil
	}

	memberRows := []struct {
		AttemptID int64                    `db:"attempt_id"`
		UserID    int64                    `db:"user_id"`
		Username  string                   `db:"username"`
		Role      models.AttemptMemberRole `db:"role"`
		JoinedAt  time.Time                `db:"joined_at"`
	}{}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	if err := db.SelectContext(ctx, &memberRows, `
		SELECT m.attempt_id, m.user_id, u.username, m.role::text, m.joined_at
		FROM event_attempt_members m JOIN users u ON u.id = m.user_id
		WHERE m.attempt_id = ANY($1)
		ORDER BY (m.role = 'owner') DESC, LOWER(u.username)`, ids); err != nil {
		return nil, err
	}
	membersByAttempt := map[int64][]models.AttemptMember{}
	for _, m := range memberRows {
		membersByAttempt[m.AttemptID] = append(membersByAttempt[m.AttemptID], models.AttemptMember{
			UserID: m.UserID, Username: m.Username, Role: m.Role, JoinedAt: m.JoinedAt,
		})
	}

	out := make([]models.EventAttempt, 0, len(rows))
	for _, row := range rows {
		members := membersByAttempt[row.ID]
		isMine := false
		for _, member := range members {
			if member.UserID == viewerID {
				isMine = true
				break
			}
		}
		out = append(out, attemptView(row, members, isMine, isHost, baseURL))
	}
	return out, nil
}

// attemptView assembles the API representation of an attempt.
func attemptView(row AttemptRow, members []models.AttemptMember, isMine, isHost bool, baseURL string) models.EventAttempt {
	attempt := models.EventAttempt{
		ID:          row.ID,
		EventID:     row.EventID,
		EventTaskID: row.EventTaskID,
		Name:        row.Name,
		Kind:        models.AttemptKind(row.Kind),
		Owner:       row.owner(),
		Members:     members,
		RepoID:      row.RepoPath,
		RepoURL:     baseURL + "/git/" + row.RepoPath + ".git",
		IsMine:      isMine,
		CanEdit:     isMine && row.EventStatus == string(models.EventStatusLive),
		Deployment: models.EventDeployment{
			AttemptID:  row.ID,
			PublicID:   row.PublicID,
			URL:        baseURL + models.EventDeployPrefix + row.PublicID + "/",
			CommitSHA:  row.CommitSHA,
			Status:     row.DeploymentStatus,
			Error:      row.DeploymentError,
			DeployedAt: row.DeployedAt,
			Published:  row.Published,
		},
	}
	if (isHost || isMine) && row.Kind == string(models.AttemptTeam) {
		attempt.JoinCode = row.JoinCode
	}
	return attempt
}

// GetAttemptView assembles the API representation of one attempt for a viewer.
func GetAttemptView(ctx context.Context, db *sqlx.DB, attemptID, viewerID int64, isHost bool, baseURL string) (models.EventAttempt, error) {
	row, err := GetAttempt(ctx, db, attemptID)
	if err != nil {
		return models.EventAttempt{}, err
	}
	members, err := ListAttemptMembers(ctx, db, attemptID)
	if err != nil {
		return models.EventAttempt{}, err
	}
	isMine := false
	for _, member := range members {
		if member.UserID == viewerID {
			isMine = true
			break
		}
	}
	return attemptView(row, members, isMine, isHost, baseURL), nil
}

// ListEventTasks returns the tasks of an event with the attempts visible to
// the viewer.
func ListEventTasks(ctx context.Context, db *sqlx.DB, eventID, viewerID int64, isHost bool, baseURL string) ([]models.EventTask, error) {
	rows := []struct {
		ID           int64  `db:"id"`
		EventID      int64  `db:"event_id"`
		AssessmentID int64  `db:"assessment_id"`
		Position     int    `db:"position"`
		Title        string `db:"title"`
		Description  string `db:"description"`
	}{}
	err := db.SelectContext(ctx, &rows, `
		SELECT t.id, t.event_id, t.assessment_id, t.position, a.title, a.description
		FROM event_tasks t JOIN assessments a ON a.id = t.assessment_id
		WHERE t.event_id = $1
		ORDER BY t.position, t.id`, eventID)
	if err != nil {
		return nil, err
	}
	out := make([]models.EventTask, 0, len(rows))
	for _, row := range rows {
		attempts, err := ListAttempts(ctx, db, row.ID, viewerID, isHost, baseURL)
		if err != nil {
			return nil, err
		}
		out = append(out, models.EventTask{
			ID:           row.ID,
			EventID:      row.EventID,
			AssessmentID: row.AssessmentID,
			Position:     row.Position,
			Title:        row.Title,
			Description:  row.Description,
			Attempts:     attempts,
		})
	}
	return out, nil
}

// CreateAttempt creates one attempt for an event task, its owner membership
// and its deployment row. The caller is responsible for seeding the repository.
func CreateAttempt(ctx context.Context, db *sqlx.DB, eventID, taskID, ownerID int64, kind models.AttemptKind, name string) (AttemptRow, error) {
	if kind != models.AttemptSolo && kind != models.AttemptTeam {
		return AttemptRow{}, fmt.Errorf("attempt kind must be solo or team")
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return AttemptRow{}, err
	}
	defer tx.Rollback()

	var assessmentID int64
	if err := tx.GetContext(ctx, &assessmentID, `
		SELECT assessment_id FROM event_tasks
		WHERE id = $1 AND event_id = $2`, taskID, eventID); err != nil {
		return AttemptRow{}, mapNotFound(err)
	}
	var assessmentKind string
	if err := tx.GetContext(ctx, &assessmentKind,
		"SELECT kind FROM assessments WHERE id = $1", assessmentID); err != nil {
		return AttemptRow{}, mapNotFound(err)
	}
	if assessmentKind != models.KindUnbound {
		return AttemptRow{}, fmt.Errorf("task %d is not an unbound task", taskID)
	}

	var id int64
	for range 5 {
		code, err := randomCode(10)
		if err != nil {
			return AttemptRow{}, err
		}
		placeholder := "pending-" + code
		if _, err := tx.ExecContext(ctx, "SAVEPOINT attempt_code"); err != nil {
			return AttemptRow{}, err
		}
		err = tx.GetContext(ctx, &id, `
			INSERT INTO event_attempts
			    (event_id, event_task_id, name, kind, owner_id, join_code, repo_path)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			eventID, taskID, name, string(kind), ownerID, code, placeholder)
		if err == nil {
			if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT attempt_code"); err != nil {
				return AttemptRow{}, err
			}
			break
		}
		if !IsUniqueViolation(err) {
			return AttemptRow{}, err
		}
		if _, rollbackErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT attempt_code"); rollbackErr != nil {
			return AttemptRow{}, rollbackErr
		}
	}
	if id == 0 {
		return AttemptRow{}, errors.New("could not allocate a unique attempt code")
	}

	repoPath := models.AttemptRepoID(eventID, id)
	if _, err := tx.ExecContext(ctx, "UPDATE event_attempts SET repo_path = $2 WHERE id = $1", id, repoPath); err != nil {
		return AttemptRow{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_attempt_members (attempt_id, user_id, role)
		VALUES ($1, $2, 'owner')`, id, ownerID); err != nil {
		return AttemptRow{}, err
	}
	publicID, err := randomCode(20)
	if err != nil {
		return AttemptRow{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO attempt_deployments (attempt_id, public_id)
		VALUES ($1, $2)`, id, strings.ToLower(publicID)); err != nil {
		return AttemptRow{}, err
	}
	if err := tx.Commit(); err != nil {
		return AttemptRow{}, err
	}
	return GetAttempt(ctx, db, id)
}

// AddAttemptMember adds a user to an attempt.
func AddAttemptMember(ctx context.Context, db *sqlx.DB, attemptID, userID int64) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO event_attempt_members (attempt_id, user_id, role)
		VALUES ($1, $2, 'member')
		ON CONFLICT (attempt_id, user_id) DO NOTHING`, attemptID, userID)
	return err
}

// JoinAttemptByCode adds the user to the team attempt with the given invite
// code and returns its id. ErrNotFound when the code matches no attempt.
func JoinAttemptByCode(ctx context.Context, db *sqlx.DB, userID int64, code string) (int64, error) {
	var attemptID int64
	var kind string
	err := db.QueryRowxContext(ctx, `
		SELECT id, kind::text FROM event_attempts
		WHERE UPPER(join_code) = UPPER($1)`, strings.TrimSpace(code)).
		Scan(&attemptID, &kind)
	if err != nil {
		return 0, mapNotFound(err)
	}
	if kind != string(models.AttemptTeam) {
		return 0, ErrNotFound
	}
	if err := AddAttemptMember(ctx, db, attemptID, userID); err != nil {
		return 0, err
	}
	return attemptID, nil
}

// RemoveAttemptMember removes an invited member. The owner cannot be removed.
func RemoveAttemptMember(ctx context.Context, db *sqlx.DB, attemptID, userID int64) error {
	res, err := db.ExecContext(ctx, `
		DELETE FROM event_attempt_members
		WHERE attempt_id = $1 AND user_id = $2 AND role <> 'owner'`, attemptID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetEventDetail assembles the full event payload. Tasks carry only the
// attempts the viewer may see. includeJoinCode should be true only for the host.
func GetEventDetail(ctx context.Context, db *sqlx.DB, id int64, baseURL string, viewerID int64, isHost, includeJoinCode bool) (models.Event, error) {
	row, err := GetEventRow(ctx, db, id)
	if err != nil {
		return models.Event{}, err
	}
	members, err := ListEventMembers(ctx, db, id)
	if err != nil {
		return models.Event{}, err
	}
	tasks, err := ListEventTasks(ctx, db, id, viewerID, isHost, baseURL)
	if err != nil {
		return models.Event{}, err
	}
	out := models.Event{
		ID:          row.ID,
		Title:       row.Title,
		Description: row.Description,
		Status:      row.Status,
		Host:        row.host(),
		IsHost:      isHost,
		StartsAt:    row.StartsAt,
		EndsAt:      row.EndsAt,
		Tasks:       tasks,
		Members:     members,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if includeJoinCode {
		out.JoinCode = row.JoinCode
	}
	return out, nil
}

// EventPatch carries the optional scalar updates of PATCH /events/{id}.
type EventPatch struct {
	Title       *string
	Description *string
	StartsAt    **time.Time
	EndsAt      **time.Time
}

// UpdateEvent applies the provided fields.
func UpdateEvent(ctx context.Context, db *sqlx.DB, id int64, patch EventPatch) error {
	sets := []string{}
	args := []any{}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if patch.Title != nil {
		add("title", *patch.Title)
	}
	if patch.Description != nil {
		add("description", *patch.Description)
	}
	if patch.StartsAt != nil {
		add("starts_at", *patch.StartsAt)
	}
	if patch.EndsAt != nil {
		add("ends_at", *patch.EndsAt)
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	res, err := db.ExecContext(ctx,
		"UPDATE events SET "+strings.Join(sets, ", ")+fmt.Sprintf(" WHERE id = $%d", len(args)), args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetEventStatus moves an event from one status to another. The transition is
// checked against the current row inside the UPDATE so two concurrent calls
// cannot both pass.
func SetEventStatus(ctx context.Context, db *sqlx.DB, id int64, from, to models.EventStatus) error {
	res, err := db.ExecContext(ctx, `
		UPDATE events SET status = $3
		WHERE id = $1 AND status = $2`, id, string(from), string(to))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := GetEventRow(ctx, db, id); err != nil {
			return err
		}
		return EventStatusError
	}
	return nil
}

// SetAttemptDeploymentDeploying marks a deployment as in progress.
func SetAttemptDeploymentDeploying(ctx context.Context, db *sqlx.DB, attemptID int64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE attempt_deployments SET status = 'deploying', error = ''
		WHERE attempt_id = $1`, attemptID)
	return err
}

// SetAttemptDeploymentResult records the outcome of a deploy.
func SetAttemptDeploymentResult(ctx context.Context, db *sqlx.DB, attemptID int64, sha, status, message string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE attempt_deployments
		SET commit_sha = $2, status = $3, error = $4,
		    deployed_at = CASE WHEN $3 = 'live' THEN NOW() ELSE deployed_at END
		WHERE attempt_id = $1`, attemptID, sha, status, message)
	return err
}

// SetAttemptDeploymentPublished toggles whether a finished deployment is
// published as a project.
func SetAttemptDeploymentPublished(ctx context.Context, db *sqlx.DB, attemptID int64, published bool) error {
	res, err := db.ExecContext(ctx, `
		UPDATE attempt_deployments SET published = $2
		WHERE attempt_id = $1`, attemptID, published)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func dedupeIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func randomCode(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = joinCodeAlphabet[int(b)%len(joinCodeAlphabet)]
	}
	return string(out), nil
}
