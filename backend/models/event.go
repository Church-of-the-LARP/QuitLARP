package models

import (
	"fmt"
	"time"
)

// EventStatus is the lifecycle state of an event. Draft events accept no
// attempts, live events accept edits and pushes, and ended events are
// read-only for members while the host keeps control of the attempts.
type EventStatus string

const (
	EventStatusDraft EventStatus = "draft"
	EventStatusLive  EventStatus = "live"
	EventStatusEnded EventStatus = "ended"
)

// EventMemberRole is the relationship of a user to an event.
type EventMemberRole string

const (
	EventRoleHost   EventMemberRole = "host"
	EventRoleMember EventMemberRole = "member"
)

// EventMember is one participant of an event.
type EventMember struct {
	UserID   int64           `json:"userId" db:"user_id" doc:"User id"`
	Username string          `json:"username" db:"username" doc:"Display name"`
	Role     EventMemberRole `json:"role" db:"role" enum:"host,member" doc:"Host or participant"`
	JoinedAt time.Time       `json:"joinedAt" db:"joined_at" doc:"When the member joined"`
}

// AttemptKind distinguishes a solo submission from a team one.
type AttemptKind string

const (
	AttemptSolo AttemptKind = "solo"
	AttemptTeam AttemptKind = "team"
)

// AttemptMemberRole is the relationship of a user to one attempt.
type AttemptMemberRole string

const (
	AttemptRoleOwner  AttemptMemberRole = "owner"
	AttemptRoleMember AttemptMemberRole = "member"
)

// AttemptMember is one user of a solo or team attempt.
type AttemptMember struct {
	UserID   int64             `json:"userId" db:"user_id" doc:"User id"`
	Username string            `json:"username" db:"username" doc:"Display name"`
	Role     AttemptMemberRole `json:"role" db:"role" enum:"owner,member" doc:"Owner or invited member"`
	JoinedAt time.Time         `json:"joinedAt" db:"joined_at" doc:"When the member joined"`
}

// EventDeployment is the live static deployment of one attempt. PublicID is
// the unguessable path segment used by the /deploy/<id>/ route.
type EventDeployment struct {
	AttemptID  int64      `json:"attemptId" doc:"Attempt that owns the deployment"`
	PublicID   string     `json:"publicId" doc:"Path segment of the deployment URL"`
	URL        string     `json:"url" doc:"URL the deployed app is served from"`
	CommitSHA  string     `json:"commitSha" doc:"Commit that was deployed"`
	Status     string     `json:"status" enum:"empty,deploying,live,failed" doc:"Deployment state"`
	Error      string     `json:"error,omitempty" doc:"Why the last deployment failed"`
	DeployedAt *time.Time `json:"deployedAt,omitempty" doc:"When the current commit went live"`
	Published  bool       `json:"published" doc:"Whether the host published the deployment after the event"`
}

// EventAttempt is one person's or team's submission for one event task. It
// owns a workspace repository and a deployment, independently of every other
// attempt.
type EventAttempt struct {
	ID          int64           `json:"id" doc:"Attempt id"`
	EventID     int64           `json:"eventId" doc:"The event this attempt belongs to"`
	EventTaskID int64           `json:"eventTaskId" doc:"The task this attempt answers"`
	Name        string          `json:"name" doc:"Attempt or team name"`
	Kind        AttemptKind     `json:"kind" enum:"solo,team" doc:"Solo submission or team submission"`
	Owner       *AuthorRef      `json:"owner" doc:"Attempt owner; null when the account was deleted"`
	Members     []AttemptMember `json:"members" doc:"Attempt owner and team members"`
	RepoID      string          `json:"repoId" doc:"Workspace repository path on the local git server"`
	RepoURL     string          `json:"repoUrl" doc:"Clone URL of the attempt workspace"`
	JoinCode    string          `json:"joinCode,omitempty" doc:"Team invite code; only returned to attempt members and the host"`
	IsMine      bool            `json:"isMine" doc:"Whether the caller is a member of this attempt"`
	CanEdit     bool            `json:"canEdit" doc:"Whether the caller may edit this attempt"`
	Dirty       bool            `json:"dirty" doc:"Whether the live editor has changes not yet committed to main"`
	Deployment  EventDeployment `json:"deployment" doc:"Live deployment of this attempt"`
}

// EventTask is an unbound task attached to an event. Attempts contains only
// the attempts the caller may see: the host sees all of them, participants
// see their own.
type EventTask struct {
	ID           int64          `json:"id" doc:"Event task id"`
	EventID      int64          `json:"eventId" doc:"The event this task belongs to"`
	AssessmentID int64          `json:"assessmentId" doc:"The assessment attached to the event"`
	Kind         string         `json:"kind" doc:"Assessment kind: leetcode or unbound"`
	Position     int            `json:"position" doc:"1-based ordering hint"`
	Title        string         `json:"title" doc:"Task title"`
	Description  string         `json:"description" doc:"Task description"`
	Attempts     []EventAttempt `json:"attempts" doc:"Attempts visible to the caller"`
}

// EventSummary is one row of GET /api/v1/events.
type EventSummary struct {
	ID           int64       `json:"id" doc:"Event id"`
	Title        string      `json:"title" doc:"Event title"`
	Description  string      `json:"description" doc:"Event description"`
	Status       EventStatus `json:"status" enum:"draft,live,ended" doc:"Lifecycle state"`
	Host         *AuthorRef  `json:"host" doc:"User running the event"`
	Solo         bool        `json:"solo" doc:"Whether this is a private solo practice workspace"`
	Monitored    bool        `json:"monitored" doc:"Whether HR monitors this event and its results"`
	AllowInvites bool        `json:"allowInvites" doc:"Whether participants may invite collaborators into their attempt"`
	StartsAt     *time.Time  `json:"startsAt,omitempty" doc:"Planned start"`
	EndsAt       *time.Time  `json:"endsAt,omitempty" doc:"Planned end"`
	TaskCount    int64       `json:"taskCount" doc:"Number of unbound tasks attached"`
	CreatedAt    time.Time   `json:"createdAt" doc:"Creation time"`
}

// Event is the full representation returned by GET /api/v1/events/{id}.
type Event struct {
	ID           int64         `json:"id" doc:"Event id"`
	Title        string        `json:"title" doc:"Event title"`
	Description  string        `json:"description" doc:"Event description"`
	Status       EventStatus   `json:"status" enum:"draft,live,ended" doc:"Lifecycle state"`
	Host         *AuthorRef    `json:"host" doc:"User running the event"`
	IsHost       bool          `json:"isHost" doc:"Whether the caller hosts this event"`
	Solo         bool          `json:"solo" doc:"Whether this is a private solo practice workspace"`
	Monitored    bool          `json:"monitored" doc:"Whether HR monitors this event and its results"`
	AllowInvites bool          `json:"allowInvites" doc:"Whether participants may invite collaborators into their attempt"`
	JoinCode     string        `json:"joinCode,omitempty" doc:"Code members use to join; only returned to the host"`
	StartsAt     *time.Time    `json:"startsAt,omitempty" doc:"Planned start"`
	EndsAt       *time.Time    `json:"endsAt,omitempty" doc:"Planned end"`
	Tasks        []EventTask   `json:"tasks" doc:"Unbound tasks of this event"`
	Members      []EventMember `json:"members" doc:"Participants, host first"`
	CreatedAt    time.Time     `json:"createdAt" doc:"Creation time"`
	UpdatedAt    time.Time     `json:"updatedAt" doc:"Last modification time"`
}

// EventDraft is the payload of POST /api/v1/events.
type EventDraft struct {
	Title           string     `json:"title" doc:"Event title"`
	Description     string     `json:"description" doc:"Event description"`
	TaskIDs         []int64    `json:"taskIds,omitempty" doc:"Ids of assessments to attach"`
	MemberUsernames []string   `json:"memberUsernames,omitempty" doc:"Users assigned to the event up front"`
	Monitored       bool       `json:"monitored,omitempty" doc:"HR interview session; requires the moderator or admin role"`
	AllowInvites    bool       `json:"allowInvites,omitempty" doc:"Let candidates invite collaborators into their attempt"`
	StartsAt        *time.Time `json:"startsAt,omitempty" doc:"Planned start"`
	EndsAt          *time.Time `json:"endsAt,omitempty" doc:"Planned end"`
}

// AttemptDraft is the payload of POST /events/{id}/tasks/{taskId}/attempts.
type AttemptDraft struct {
	Kind          AttemptKind `json:"kind" enum:"solo,team" doc:"Solo or team attempt"`
	Name          string      `json:"name,omitempty" doc:"Attempt or team name; defaults to the owner's username"`
	OwnerUsername string      `json:"ownerUsername,omitempty" doc:"Host only: create the attempt for another user"`
}

// AttemptRepoID is the path of an attempt's workspace repository on the local
// git server. The stored repo_path of an attempt is authoritative; this helper
// is used when a new attempt is created.
func AttemptRepoID(eventID, attemptID int64) string {
	return fmt.Sprintf("events/event-%d/attempt-%d", eventID, attemptID)
}

// EventDeployPrefix is the URL prefix deployments are served under.
const EventDeployPrefix = "/deploy/"
