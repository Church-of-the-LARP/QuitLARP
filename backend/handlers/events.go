package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	gitobject "github.com/go-git/go-git/v5/plumbing/object"

	"backend/assessment"
	"backend/collab"
	"backend/database"
	"backend/middleware"
	"backend/models"
	"backend/repos"
)

// maxEventFileBytes caps files the browser editor may load, create or commit.
const maxEventFileBytes = 512 << 10

// EventListOutput is the response of GET /api/v1/events.
type EventListOutput struct {
	Body struct {
		Events []models.EventSummary `json:"events"`
	}
}

// EventOutput is the response of the single-event endpoints.
type EventOutput struct {
	Body struct {
		Event models.Event `json:"event"`
	}
}

// AttemptOutput is the response of the single-attempt endpoints.
type AttemptOutput struct {
	Body struct {
		Attempt models.EventAttempt `json:"attempt"`
	}
}

// EventResultsOutput is the response of GET /api/v1/events/{id}/results.
type EventResultsOutput struct {
	Body struct {
		Results models.EventResults `json:"results"`
	}
}

// WorkspaceFile is a non-editable file of an attempt workspace (binary or
// larger than the editor limit).
type WorkspaceFile struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Binary    bool   `json:"binary"`
	Truncated bool   `json:"truncated"`
}

// WorkspaceOutput is the response of GET .../workspace. Docs are the editable
// live documents; Files lists binary/oversized files on disk; Deleted lists
// paths removed in the editor but not committed yet.
type WorkspaceOutput struct {
	Body struct {
		Docs    []collab.Doc    `json:"docs"`
		Files   []WorkspaceFile `json:"files"`
		Deleted []string        `json:"deleted"`
		Dirty   bool            `json:"dirty"`
	}
}

// EventEditOutput is the response of POST .../edits.
type EventEditOutput struct {
	Body struct {
		Version int       `json:"version"`
		Op      collab.Op `json:"op"`
	}
}

// CreateEventInput is the request of POST /api/v1/events.
type CreateEventInput struct {
	Body models.EventDraft
}

// EventIDInput is the request of endpoints scoped to one event.
type EventIDInput struct {
	ID int64 `path:"id" example:"1" doc:"Event id"`
}

// CreateAttemptInput is the request of
// POST /api/v1/events/{id}/tasks/{taskId}/attempts.
type CreateAttemptInput struct {
	ID     int64 `path:"id" example:"1" doc:"Event id"`
	TaskID int64 `path:"taskId" example:"1" doc:"Event task id"`
	Body   models.AttemptDraft
}

// AttemptIDInput is the request of endpoints scoped to one attempt.
type AttemptIDInput struct {
	ID        int64 `path:"id" example:"1" doc:"Event id"`
	AttemptID int64 `path:"attemptId" example:"1" doc:"Attempt id"`
}

// EventEditInput is the request of POST .../edits.
type EventEditInput struct {
	ID        int64 `path:"id" example:"1" doc:"Event id"`
	AttemptID int64 `path:"attemptId" example:"1" doc:"Attempt id"`
	Body      struct {
		Path        string `json:"path" doc:"File being edited"`
		Offset      int    `json:"offset" doc:"UTF-16 offset of the change"`
		DeleteCount int    `json:"deleteCount" doc:"UTF-16 units to delete"`
		Insert      string `json:"insert" doc:"Text to insert"`
		BaseVersion int    `json:"baseVersion" doc:"Version the edit was made against"`
		ClientID    string `json:"clientId" doc:"Client identifier, echoed in the broadcast"`
	}
}

// CreateEventFileInput is the request of POST .../files.
type CreateEventFileInput struct {
	ID        int64 `path:"id" example:"1" doc:"Event id"`
	AttemptID int64 `path:"attemptId" example:"1" doc:"Attempt id"`
	Body      struct {
		Path    string `json:"path" doc:"File path within the workspace"`
		Content string `json:"content" doc:"Initial contents"`
	}
}

// DeleteEventFileInput is the request of DELETE .../files.
type DeleteEventFileInput struct {
	ID        int64  `path:"id" example:"1" doc:"Event id"`
	AttemptID int64  `path:"attemptId" example:"1" doc:"Attempt id"`
	Path      string `query:"path" doc:"File path to delete"`
}

// CommitAttemptInput is the request of POST .../commit.
type CommitAttemptInput struct {
	ID        int64 `path:"id" example:"1" doc:"Event id"`
	AttemptID int64 `path:"attemptId" example:"1" doc:"Attempt id"`
	Body      struct {
		Message string `json:"message,omitempty" doc:"Commit message"`
	}
}

// PublishAttemptDeploymentInput is the request of PATCH .../deployment.
type PublishAttemptDeploymentInput struct {
	ID        int64 `path:"id" example:"1" doc:"Event id"`
	AttemptID int64 `path:"attemptId" example:"1" doc:"Attempt id"`
	Body      struct {
		Published bool `json:"published" doc:"Whether the finished deployment is published"`
	}
}

// AddAttemptMemberInput is the request of POST .../members.
type AddAttemptMemberInput struct {
	ID        int64 `path:"id" example:"1" doc:"Event id"`
	AttemptID int64 `path:"attemptId" example:"1" doc:"Attempt id"`
	Body      struct {
		Username string `json:"username" doc:"Username to add"`
	}
}

// RemoveAttemptMemberInput is the request of DELETE .../members/{userId}.
type RemoveAttemptMemberInput struct {
	ID        int64 `path:"id" example:"1" doc:"Event id"`
	AttemptID int64 `path:"attemptId" example:"1" doc:"Attempt id"`
	UserID    int64 `path:"userId" example:"2" doc:"User to remove"`
}

// PatchEventInput is the request of PATCH /api/v1/events/{id}.
type PatchEventInput struct {
	ID   int64 `path:"id" example:"1" doc:"Event id"`
	Body struct {
		Title        *string    `json:"title,omitempty" doc:"Event title"`
		Description  *string    `json:"description,omitempty" doc:"Event description"`
		StartsAt     *time.Time `json:"startsAt,omitempty" doc:"Planned start"`
		EndsAt       *time.Time `json:"endsAt,omitempty" doc:"Planned end"`
		AllowInvites *bool      `json:"allowInvites,omitempty" doc:"Let candidates invite collaborators into their attempt"`
	}
}

// JoinEventInput is the request of POST /api/v1/events/join.
type JoinEventInput struct {
	Body struct {
		Code string `json:"code" doc:"Join code shared by the host"`
	}
}

// JoinAttemptInput is the request of POST /api/v1/attempts/join.
type JoinAttemptInput struct {
	Body struct {
		Code string `json:"code" doc:"Team invite code"`
	}
}

// AddEventMemberInput is the request of POST /api/v1/events/{id}/members.
type AddEventMemberInput struct {
	ID   int64 `path:"id" example:"1" doc:"Event id"`
	Body struct {
		Username string `json:"username" doc:"Username to add"`
	}
}

// RemoveEventMemberInput is the request of
// DELETE /api/v1/events/{id}/members/{userId}.
type RemoveEventMemberInput struct {
	ID     int64 `path:"id" example:"1" doc:"Event id"`
	UserID int64 `path:"userId" example:"2" doc:"User to remove"`
}

func (h *Handlers) registerEvents(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listEvents",
		Method:      http.MethodGet,
		Path:        "/api/v1/events",
		Summary:     "List my collaborations",
		Description: "Requires authentication. Returns the events the caller hosts or has joined, newest first. Creating events remains restricted to moderators and admins.",
	}, func(ctx context.Context, _ *struct{}) (*EventListOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		list, err := database.ListUserEvents(ctx, h.db, u.ID)
		if err != nil {
			log.Printf("list events: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list events")
		}
		resp := &EventListOutput{}
		resp.Body.Events = list
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listUpcomingEvents",
		Method:      http.MethodGet,
		Path:        "/api/v1/upcoming-events",
		Summary:     "List upcoming events",
		Description: "Requires authentication. Returns the live events the caller is assigned to whose deadline has not passed, for the dashboard's upcoming-tests list.",
	}, func(ctx context.Context, _ *struct{}) (*EventListOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		list, err := database.ListUpcomingEvents(ctx, h.db, u.ID)
		if err != nil {
			log.Printf("list upcoming events: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list upcoming events")
		}
		resp := &EventListOutput{}
		resp.Body.Events = list
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createEvent",
		Method:      http.MethodPost,
		Path:        "/api/v1/events",
		Summary:     "Create an event",
		Description: "Requires authentication. Moderators and admins host monitored interview sessions; regular users host unmonitored collaborations. Tasks may be any assessment and memberUsernames assigns participants up front.",
	}, func(ctx context.Context, input *CreateEventInput) (*EventOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		draft := input.Body
		draft.Title = strings.TrimSpace(draft.Title)
		draft.Description = strings.TrimSpace(draft.Description)
		if draft.Title == "" {
			return nil, unprocessable("title is required")
		}
		if draft.StartsAt != nil && draft.EndsAt != nil && draft.EndsAt.Before(*draft.StartsAt) {
			return nil, unprocessable("endsAt must not be before startsAt")
		}
		monitored := draft.Monitored
		if monitored && u.Role != models.RoleModerator && u.Role != models.RoleAdmin {
			return nil, huma.NewError(http.StatusForbidden, "only moderators and admins can host monitored interviews")
		}
		if !monitored || (u.Role != models.RoleModerator && u.Role != models.RoleAdmin) {
			draft.AllowInvites = false
		}
		id, err := database.CreateEvent(ctx, h.db, u.ID, monitored, draft)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, unprocessable("one of the tasks or users does not exist")
			}
			log.Printf("create event: %v", err)
			return nil, unprocessable(err.Error())
		}
		out, err := h.eventOutput(ctx, id, u)
		if err != nil {
			return nil, err
		}
		event := out.Body.Event
		if monitored {
			h.notifyUser(ctx, u.ID, "Interview created",
				fmt.Sprintf("You created %q with %d task(s) and %d candidate(s).",
					event.Title, len(event.Tasks), len(event.Members)-1))
		}
		inviteTitle := "Collaboration invitation"
		if monitored {
			inviteTitle = "Interview invitation"
		}
		for _, member := range event.Members {
			if member.Role == models.EventRoleMember && member.UserID != u.ID {
				h.notifyUser(ctx, member.UserID, inviteTitle,
					fmt.Sprintf("%s invited you to %q.", u.Username, event.Title))
			}
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getEvent",
		Method:      http.MethodGet,
		Path:        "/api/v1/events/{id}",
		Summary:     "Get an event",
		Description: "The host sees every attempt and deployment; participants see their own attempts. The join code is only returned to the host.",
	}, func(ctx context.Context, input *EventIDInput) (*EventOutput, error) {
		u, _, _, err := h.requireEventUser(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		return h.eventOutput(ctx, input.ID, u)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getEventResults",
		Method:      http.MethodGet,
		Path:        "/api/v1/events/{id}/results",
		Summary:     "Get event results",
		Description: "The host sees every unbound attempt with its workspace/deployment, and each participant's latest solution and test run for coding tasks.",
	}, func(ctx context.Context, input *EventIDInput) (*EventResultsOutput, error) {
		u, err := h.requireEventHost(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		row, err := database.GetEventRow(ctx, h.db, input.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		if !row.Monitored {
			return nil, huma.NewError(http.StatusForbidden, "results are only available for HR-hosted interviews")
		}
		tasks, err := database.ListEventTasks(ctx, h.db, input.ID, u.ID, true, h.cfg.PublicBaseURL)
		if err != nil {
			return nil, h.eventError(err)
		}
		members, err := database.ListEventMembers(ctx, h.db, input.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		results := models.EventResults{Tasks: make([]models.EventTaskResults, 0, len(tasks))}
		for _, task := range tasks {
			taskResults := models.EventTaskResults{
				ID:           task.ID,
				AssessmentID: task.AssessmentID,
				Kind:         task.Kind,
				Title:        task.Title,
				Attempts:     task.Attempts,
				Participants: []models.EventResultParticipant{},
			}
			if task.Kind != models.KindUnbound {
				for _, member := range members {
					participant := models.EventResultParticipant{
						UserID:   member.UserID,
						Username: member.Username,
					}
					solution, err := database.GetSolution(ctx, h.db, task.AssessmentID, member.UserID)
					switch {
					case err == nil:
						ref := &models.SolutionRef{
							CommitSHA: solution.CommitSHA,
							CreatedAt: solution.CreatedAt,
							UpdatedAt: solution.UpdatedAt,
							Files:     []models.SolutionFile{},
						}
						dir := filepath.Join(h.cfg.GitReposDir,
							filepath.FromSlash(models.SolutionRepoID(task.AssessmentID, member.UserID)+".git"))
						files, fileErr := repos.SnapshotFiles(dir)
						if fileErr != nil {
							log.Printf("event results: read solution of user %d: %v", member.UserID, fileErr)
						}
						for path, data := range files {
							if len(data) > maxEventFileBytes || repos.IsBinary(data) {
								continue
							}
							ref.Files = append(ref.Files, models.SolutionFile{
								Path:    path,
								Content: string(data),
							})
						}
						sort.Slice(ref.Files, func(i, j int) bool {
							return ref.Files[i].Path < ref.Files[j].Path
						})
						participant.Solution = ref
					case !errors.Is(err, database.ErrNotFound):
						log.Printf("event results: solution of user %d: %v", member.UserID, err)
					}
					run, err := database.LatestTestRun(ctx, h.db, task.AssessmentID, member.UserID)
					switch {
					case err == nil:
						participant.TestRun = &run
					case !errors.Is(err, database.ErrNotFound):
						log.Printf("event results: test run of user %d: %v", member.UserID, err)
					}
					if member.Role == models.EventRoleHost &&
						participant.Solution == nil && participant.TestRun == nil {
						continue
					}
					taskResults.Participants = append(taskResults.Participants, participant)
				}
			}
			results.Tasks = append(results.Tasks, taskResults)
		}
		resp := &EventResultsOutput{}
		resp.Body.Results = results
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "patchEvent",
		Method:      http.MethodPatch,
		Path:        "/api/v1/events/{id}",
		Summary:     "Update an event",
		Description: "The host may update the title, description and planned start/end of a draft or live event.",
	}, func(ctx context.Context, input *PatchEventInput) (*EventOutput, error) {
		u, err := h.requireEventHost(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		patch := database.EventPatch{}
		if input.Body.Title != nil {
			v := strings.TrimSpace(*input.Body.Title)
			if v == "" {
				return nil, unprocessable("title cannot be blank")
			}
			patch.Title = &v
		}
		if input.Body.Description != nil {
			v := strings.TrimSpace(*input.Body.Description)
			patch.Description = &v
		}
		if input.Body.StartsAt != nil {
			v := input.Body.StartsAt
			patch.StartsAt = &v
		}
		if input.Body.EndsAt != nil {
			v := input.Body.EndsAt
			patch.EndsAt = &v
		}
		if input.Body.AllowInvites != nil {
			patch.AllowInvites = input.Body.AllowInvites
		}
		if err := database.UpdateEvent(ctx, h.db, input.ID, patch); err != nil {
			return nil, h.eventError(err)
		}
		return h.eventOutput(ctx, input.ID, u)
	})

	huma.Register(api, huma.Operation{
		OperationID: "startEvent",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/start",
		Summary:     "Start an event",
		Description: "The host starts a draft event. Participants can then create solo or team attempts, each with its own workspace and deployment.",
	}, func(ctx context.Context, input *EventIDInput) (*EventOutput, error) {
		u, err := h.requireEventHost(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		row, err := database.GetEventRow(ctx, h.db, input.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		switch row.Status {
		case models.EventStatusLive:
			return h.eventOutput(ctx, input.ID, u)
		case models.EventStatusDraft:
			if err := database.SetEventStatus(ctx, h.db, input.ID, models.EventStatusDraft, models.EventStatusLive); err != nil {
				return nil, h.eventError(err)
			}
			return h.eventOutput(ctx, input.ID, u)
		default:
			return nil, huma.NewError(http.StatusConflict, "only a draft event can be started")
		}
	})

	huma.Register(api, huma.Operation{
		OperationID: "endEvent",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/end",
		Summary:     "End an event",
		Description: "The host ends a live event. Attempts become read-only; only committed main branches are deployed. The host can publish finished attempts.",
	}, func(ctx context.Context, input *EventIDInput) (*EventOutput, error) {
		u, err := h.requireEventHost(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		row, err := database.GetEventRow(ctx, h.db, input.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		switch row.Status {
		case models.EventStatusEnded:
			return h.eventOutput(ctx, input.ID, u)
		case models.EventStatusLive:
			if err := database.SetEventStatus(ctx, h.db, input.ID, models.EventStatusLive, models.EventStatusEnded); err != nil {
				return nil, h.eventError(err)
			}
			return h.eventOutput(ctx, input.ID, u)
		default:
			return nil, huma.NewError(http.StatusConflict, "only a live event can be ended")
		}
	})

	huma.Register(api, huma.Operation{
		OperationID: "joinEvent",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/join",
		Summary:     "Join an event",
		Description: "Requires authentication. Adds the caller to the event with the given join code.",
	}, func(ctx context.Context, input *JoinEventInput) (*EventOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		id, err := database.JoinEventByCode(ctx, h.db, u.ID, input.Body.Code)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "no event has that join code")
			}
			log.Printf("join event: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not join the event")
		}
		return h.eventOutput(ctx, id, u)
	})

	huma.Register(api, huma.Operation{
		OperationID: "joinAttempt",
		Method:      http.MethodPost,
		Path:        "/api/v1/attempts/join",
		Summary:     "Join a team attempt",
		Description: "Requires authentication. Adds the caller to the team attempt with the given invite code and to its event.",
	}, func(ctx context.Context, input *JoinAttemptInput) (*EventOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		attemptID, err := database.JoinAttemptByCode(ctx, h.db, u.ID, input.Body.Code)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "no team attempt has that invite code")
			}
			log.Printf("join attempt: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not join the team")
		}
		attempt, err := database.GetAttempt(ctx, h.db, attemptID)
		if err != nil {
			return nil, h.eventError(err)
		}
		if err := database.AddEventMember(ctx, h.db, attempt.EventID, u.ID); err != nil {
			return nil, h.eventError(err)
		}
		return h.eventOutput(ctx, attempt.EventID, u)
	})

	huma.Register(api, huma.Operation{
		OperationID: "addEventMember",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/members",
		Summary:     "Add an event member",
		Description: "The host may add a user by username without sharing the join code.",
	}, func(ctx context.Context, input *AddEventMemberInput) (*EventOutput, error) {
		u, err := h.requireEventHost(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		member, err := database.GetUserByUsername(ctx, h.db, strings.TrimSpace(input.Body.Username))
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "no user has that username")
			}
			return nil, h.eventError(err)
		}
		row, err := database.GetEventRow(ctx, h.db, input.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		existing, err := database.GetEventMemberRole(ctx, h.db, input.ID, member.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		if existing == "" {
			if err := database.AddEventMember(ctx, h.db, input.ID, member.ID); err != nil {
				return nil, h.eventError(err)
			}
			if member.ID != u.ID {
				title := "Collaboration invitation"
				if row.Monitored {
					title = "Interview invitation"
				}
				h.notifyUser(ctx, member.ID, title,
					fmt.Sprintf("%s invited you to %q.", u.Username, row.Title))
			}
		}
		return h.eventOutput(ctx, input.ID, u)
	})

	huma.Register(api, huma.Operation{
		OperationID: "removeEventMember",
		Method:      http.MethodDelete,
		Path:        "/api/v1/events/{id}/members/{userId}",
		Summary:     "Remove an event member",
		Description: "The host may remove a participant. The host cannot be removed.",
	}, func(ctx context.Context, input *RemoveEventMemberInput) (*EventOutput, error) {
		u, err := h.requireEventHost(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		if err := database.RemoveEventMember(ctx, h.db, input.ID, input.UserID); err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "that user is not a removable member")
			}
			return nil, h.eventError(err)
		}
		return h.eventOutput(ctx, input.ID, u)
	})

	huma.Register(api, huma.Operation{
		OperationID: "createEventAttempt",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/tasks/{taskId}/attempts",
		Summary:     "Create or join a workspace",
		Description: "A live event member joins the shared collaboration workspace, or creates their own attempt in an HR interview. Collaborations reuse one workspace per task; interviews give every candidate a private attempt.",
	}, func(ctx context.Context, input *CreateAttemptInput) (*AttemptOutput, error) {
		u, _, isHost, err := h.requireEventUser(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		row, err := database.GetEventRow(ctx, h.db, input.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		if row.Status != models.EventStatusLive {
			return nil, huma.NewError(http.StatusConflict, "the event must be live to create an attempt")
		}
		if !row.Monitored {
			shared, err := database.FirstAttemptForTask(ctx, h.db, input.ID, input.TaskID)
			switch {
			case err == nil:
				if err := database.AddAttemptMember(ctx, h.db, shared.ID, u.ID); err != nil {
					return nil, h.eventError(err)
				}
				return h.attemptOutput(ctx, input.ID, shared.ID, u, isHost)
			case !errors.Is(err, database.ErrNotFound):
				return nil, h.eventError(err)
			}
		}
		kind := input.Body.Kind
		if kind == "" {
			kind = models.AttemptSolo
		}
		if kind != models.AttemptSolo && kind != models.AttemptTeam {
			return nil, unprocessable("kind must be solo or team")
		}
		owner := u
		if input.Body.OwnerUsername != "" {
			if !isHost {
				return nil, huma.NewError(http.StatusForbidden, "only the host can create an attempt for another user")
			}
			owner, err = database.GetUserByUsername(ctx, h.db, strings.TrimSpace(input.Body.OwnerUsername))
			if err != nil {
				if errors.Is(err, database.ErrNotFound) {
					return nil, huma.NewError(http.StatusNotFound, "no user has that username")
				}
				return nil, h.eventError(err)
			}
			if err := database.AddEventMember(ctx, h.db, input.ID, owner.ID); err != nil {
				return nil, h.eventError(err)
			}
		}
		name := strings.TrimSpace(input.Body.Name)
		if name == "" {
			name = owner.Username
			if kind == models.AttemptTeam {
				name = owner.Username + "'s team"
			}
		}
		attempt, err := database.CreateAttempt(ctx, h.db, input.ID, input.TaskID, owner.ID, kind, name)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "event task not found")
			}
			log.Printf("create attempt: %v", err)
			return nil, unprocessable(err.Error())
		}
		if err := h.ensureAttemptWorkspace(ctx, owner, attempt); err != nil {
			log.Printf("seed attempt %d: %v", attempt.ID, err)
			return nil, huma.NewError(http.StatusBadGateway, "could not prepare the attempt workspace: "+err.Error())
		}
		return h.attemptOutput(ctx, input.ID, attempt.ID, u, isHost)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getAttemptWorkspace",
		Method:      http.MethodGet,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/workspace",
		Summary:     "Load an attempt workspace",
		Description: "Returns the editable live documents, binary files and uncommitted deletions of an attempt. Git main remains the deployment source until an explicit commit.",
	}, func(ctx context.Context, input *AttemptIDInput) (*WorkspaceOutput, error) {
		access, err := h.loadAttemptAccess(ctx, input.ID, input.AttemptID)
		if err != nil {
			return nil, err
		}
		if err := h.ensureAttemptWorkspace(ctx, access.user, access.attempt); err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "could not prepare the workspace: "+err.Error())
		}
		return h.workspaceOutput(ctx, access.attempt)
	})

	huma.Register(api, huma.Operation{
		OperationID: "applyEventTaskEdit",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/edits",
		Summary:     "Apply a collaborative text edit",
		Description: "Attempt members edit a live workspace. Offsets are UTF-16 code units; stale edits are transformed against newer ones. Edits stay in the live editor until an explicit commit.",
	}, func(ctx context.Context, input *EventEditInput) (*EventEditOutput, error) {
		access, err := h.loadAttemptEditor(ctx, input.ID, input.AttemptID)
		if err != nil {
			return nil, err
		}
		if err := h.ensureAttemptRoom(ctx, access.attempt); err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "could not load the workspace: "+err.Error())
		}
		op, err := h.collab.Apply(access.attempt.ID, collab.Op{
			Path:        input.Body.Path,
			Offset:      input.Body.Offset,
			DeleteCount: input.Body.DeleteCount,
			Insert:      input.Body.Insert,
			BaseVersion: input.Body.BaseVersion,
			ClientID:    input.Body.ClientID,
			UserID:      access.user.ID,
			Username:    access.user.Username,
		})
		if err != nil {
			switch {
			case errors.Is(err, collab.ErrResync):
				return nil, huma.NewError(http.StatusConflict, "this file changed too much; reload it")
			case errors.Is(err, collab.ErrNotOpen):
				return nil, huma.NewError(http.StatusConflict, "this file is not open; open it first")
			default:
				return nil, unprocessable("the edit is not valid for this file")
			}
		}
		resp := &EventEditOutput{}
		resp.Body.Version = op.Version
		resp.Body.Op = op
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createEventTaskFile",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/files",
		Summary:     "Create a workspace file",
		Description: "Adds a file to the live editor only. Nothing reaches git or the deployment until the attempt is committed.",
	}, func(ctx context.Context, input *CreateEventFileInput) (*WorkspaceOutput, error) {
		access, err := h.loadAttemptEditor(ctx, input.ID, input.AttemptID)
		if err != nil {
			return nil, err
		}
		filePath := strings.TrimSpace(input.Body.Path)
		if filePath == "" || strings.HasPrefix(filePath, "/") || strings.Contains(filePath, "..") {
			return nil, unprocessable("path must be a relative file path")
		}
		if len(input.Body.Content) > maxEventFileBytes {
			return nil, unprocessable("file is larger than the editor allows")
		}
		if err := h.ensureAttemptRoom(ctx, access.attempt); err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "could not load the workspace: "+err.Error())
		}
		if _, ok := h.collab.Doc(access.attempt.ID, filePath); ok {
			return nil, huma.NewError(http.StatusConflict, "that file already exists")
		}
		if files, err := h.attemptSnapshotFiles(access.attempt); err == nil {
			if _, exists := files[filePath]; exists {
				return nil, huma.NewError(http.StatusConflict, "that file already exists and is not editable")
			}
		}
		h.collab.Create(access.attempt.ID, filePath, input.Body.Content)
		return h.workspaceOutput(ctx, access.attempt)
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteEventTaskFile",
		Method:      http.MethodDelete,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/files",
		Summary:     "Delete a workspace file",
		Description: "Removes a file from the live editor only. The deletion reaches git and the deployment on the next commit.",
	}, func(ctx context.Context, input *DeleteEventFileInput) (*WorkspaceOutput, error) {
		access, err := h.loadAttemptEditor(ctx, input.ID, input.AttemptID)
		if err != nil {
			return nil, err
		}
		if err := h.ensureAttemptRoom(ctx, access.attempt); err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "could not load the workspace: "+err.Error())
		}
		if _, ok := h.collab.Doc(access.attempt.ID, input.Path); !ok {
			if files, err := h.attemptSnapshotFiles(access.attempt); err != nil || files[input.Path] == nil {
				return nil, huma.NewError(http.StatusNotFound, "no such file")
			}
		}
		h.collab.Remove(access.attempt.ID, input.Path)
		return h.workspaceOutput(ctx, access.attempt)
	})

	huma.Register(api, huma.Operation{
		OperationID: "commitEventAttempt",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/commit",
		Summary:     "Commit the live workspace and deploy",
		Description: "Writes the attempt's live documents and deletions to git main in one commit, then deploys that commit. This is the only browser path that affects the deployment.",
	}, func(ctx context.Context, input *CommitAttemptInput) (*AttemptOutput, error) {
		access, err := h.loadAttemptEditor(ctx, input.ID, input.AttemptID)
		if err != nil {
			return nil, err
		}
		if err := h.ensureAttemptRoom(ctx, access.attempt); err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "could not load the workspace: "+err.Error())
		}
		if err := h.ensureAttemptWorkspace(ctx, access.user, access.attempt); err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "could not prepare the workspace: "+err.Error())
		}
		docs := h.collab.DirtyDocs(access.attempt.ID)
		deletes := h.collab.Deleted(access.attempt.ID)
		if len(docs) == 0 && len(deletes) == 0 {
			return nil, huma.NewError(http.StatusConflict, "there are no uncommitted changes")
		}
		files := make(map[string][]byte, len(docs))
		for _, doc := range docs {
			files[doc.Path] = []byte(doc.Content)
		}
		message := strings.TrimSpace(input.Body.Message)
		if message == "" {
			message = "Commit by " + access.user.Username
		}
		dir, err := h.gitRepoDir(access.attempt.RepoPath)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "could not open the workspace repository")
		}
		sha, err := repos.CommitChanges(dir, files, deletes, message, gitIdentity(access.user))
		if err != nil {
			if errors.Is(err, repos.ErrPathNotFound) {
				return nil, huma.NewError(http.StatusConflict, "one of the changes no longer applies")
			}
			return nil, huma.NewError(http.StatusBadGateway, "could not commit: "+err.Error())
		}
		h.collab.MarkCommitted(access.attempt.ID, sha)
		if err := h.deployAttempt(ctx, access.attempt); err != nil {
			log.Printf("deploy attempt %d after commit: %v", access.attempt.ID, err)
		}
		if row, err := database.GetEventRow(ctx, h.db, input.ID); err == nil &&
			row.Monitored && row.Status != models.EventStatusDraft {
			taskTitle, titleErr := database.GetEventTaskTitle(ctx, h.db, input.ID, access.attempt.EventTaskID)
			if titleErr != nil {
				log.Printf("commit notification task title: %v", titleErr)
				taskTitle = "the task"
			}
			h.notifyEventHosts(ctx, input.ID, access.user.ID, "New interview result",
				fmt.Sprintf("%s submitted %q for %q in %q.",
					access.user.Username, access.attempt.Name, taskTitle, row.Title))
		}
		return h.attemptOutput(ctx, input.ID, access.attempt.ID, access.user, access.isHost)
	})

	huma.Register(api, huma.Operation{
		OperationID: "deployEventAttempt",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/deploy",
		Summary:     "Deploy the attempt's git main",
		Description: "The host may redeploy the current main commit of an attempt. Uncommitted browser edits are never deployed.",
	}, func(ctx context.Context, input *AttemptIDInput) (*AttemptOutput, error) {
		u, err := h.requireEventHost(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		attempt, err := database.GetAttemptInEvent(ctx, h.db, input.ID, input.AttemptID)
		if err != nil {
			return nil, h.eventError(err)
		}
		if err := h.deployAttempt(ctx, attempt); err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "deployment failed: "+err.Error())
		}
		return h.attemptOutput(ctx, input.ID, input.AttemptID, u, true)
	})

	huma.Register(api, huma.Operation{
		OperationID: "publishEventDeployment",
		Method:      http.MethodPatch,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/deployment",
		Summary:     "Publish a finished attempt deployment",
		Description: "After an event has ended the host may publish an attempt's deployment as a finished project.",
	}, func(ctx context.Context, input *PublishAttemptDeploymentInput) (*AttemptOutput, error) {
		u, err := h.requireEventHost(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		row, err := database.GetEventRow(ctx, h.db, input.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		if input.Body.Published && row.Status != models.EventStatusEnded {
			return nil, huma.NewError(http.StatusConflict, "end the event before publishing its deployment")
		}
		if _, err := database.GetAttemptInEvent(ctx, h.db, input.ID, input.AttemptID); err != nil {
			return nil, h.eventError(err)
		}
		if err := database.SetAttemptDeploymentPublished(ctx, h.db, input.AttemptID, input.Body.Published); err != nil {
			return nil, h.eventError(err)
		}
		return h.attemptOutput(ctx, input.ID, input.AttemptID, u, true)
	})

	huma.Register(api, huma.Operation{
		OperationID: "addAttemptMember",
		Method:      http.MethodPost,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/members",
		Summary:     "Add a team member",
		Description: "The attempt owner or the host may add a user to a team attempt by username.",
	}, func(ctx context.Context, input *AddAttemptMemberInput) (*AttemptOutput, error) {
		access, err := h.loadAttemptAccess(ctx, input.ID, input.AttemptID)
		if err != nil {
			return nil, err
		}
		row, err := database.GetEventRow(ctx, h.db, input.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		allowed := access.isHost
		if !allowed && access.myRole == models.AttemptRoleOwner {
			allowed = !row.Monitored || row.AllowInvites
		}
		if !allowed {
			return nil, huma.NewError(http.StatusForbidden, "inviting collaborators is disabled for this session")
		}
		if row.Status != models.EventStatusLive {
			return nil, huma.NewError(http.StatusConflict, "the session must be live to invite collaborators")
		}
		member, err := database.GetUserByUsername(ctx, h.db, strings.TrimSpace(input.Body.Username))
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "no user has that username")
			}
			return nil, h.eventError(err)
		}
		existing, err := database.GetAttemptMemberRole(ctx, h.db, input.AttemptID, member.ID)
		if err != nil {
			return nil, h.eventError(err)
		}
		if existing == "" {
			if err := database.AddEventMember(ctx, h.db, input.ID, member.ID); err != nil {
				return nil, h.eventError(err)
			}
			if err := database.AddAttemptMember(ctx, h.db, input.AttemptID, member.ID); err != nil {
				return nil, h.eventError(err)
			}
			if member.ID != access.user.ID {
				taskTitle, titleErr := database.GetEventTaskTitle(ctx, h.db, input.ID, access.attempt.EventTaskID)
				if titleErr != nil {
					log.Printf("attempt invite task title: %v", titleErr)
					taskTitle = "the task"
				}
				h.notifyUser(ctx, member.ID, "Collaboration invitation",
					fmt.Sprintf("%s invited you to collaborate on %q for %q in %q.",
						access.user.Username, access.attempt.Name, taskTitle, row.Title))
			}
		}
		return h.attemptOutput(ctx, input.ID, input.AttemptID, access.user, access.isHost)
	})

	huma.Register(api, huma.Operation{
		OperationID: "removeAttemptMember",
		Method:      http.MethodDelete,
		Path:        "/api/v1/events/{id}/attempts/{attemptId}/members/{userId}",
		Summary:     "Remove a team member",
		Description: "The attempt owner or the host may remove an invited team member. The owner cannot be removed.",
	}, func(ctx context.Context, input *RemoveAttemptMemberInput) (*AttemptOutput, error) {
		access, err := h.loadAttemptAccess(ctx, input.ID, input.AttemptID)
		if err != nil {
			return nil, err
		}
		if !access.canManageMembers() {
			return nil, huma.NewError(http.StatusForbidden, "only the attempt owner or host can remove members")
		}
		if err := database.RemoveAttemptMember(ctx, h.db, input.AttemptID, input.UserID); err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "that user is not a removable member")
			}
			return nil, h.eventError(err)
		}
		return h.attemptOutput(ctx, input.ID, input.AttemptID, access.user, access.isHost)
	})
}

// attemptAccess is the resolved authorization state for one attempt.
type attemptAccess struct {
	user    models.User
	attempt database.AttemptRow
	isHost  bool
	myRole  models.AttemptMemberRole
}

func (a attemptAccess) isMine() bool {
	return a.myRole != ""
}

func (a attemptAccess) canManageMembers() bool {
	return a.isHost || a.myRole == models.AttemptRoleOwner
}

// loadAttemptAccess resolves the caller and verifies read access to an attempt.
func (h *Handlers) loadAttemptAccess(ctx context.Context, eventID, attemptID int64) (attemptAccess, error) {
	u, err := h.requireUser(ctx)
	if err != nil {
		return attemptAccess{}, err
	}
	attempt, err := database.GetAttemptInEvent(ctx, h.db, eventID, attemptID)
	if err != nil {
		return attemptAccess{}, h.eventError(err)
	}
	eventRole, err := database.GetEventMemberRole(ctx, h.db, eventID, u.ID)
	if err != nil {
		return attemptAccess{}, h.eventError(err)
	}
	if eventRole == "" && u.Role != models.RoleAdmin {
		return attemptAccess{}, huma.NewError(http.StatusForbidden, "you are not a member of this event")
	}
	isHost := eventRole == models.EventRoleHost || u.Role == models.RoleAdmin
	myRole, err := database.GetAttemptMemberRole(ctx, h.db, attemptID, u.ID)
	if err != nil {
		return attemptAccess{}, h.eventError(err)
	}
	if !isHost && myRole == "" {
		return attemptAccess{}, huma.NewError(http.StatusForbidden, "you are not a member of this attempt")
	}
	return attemptAccess{user: u, attempt: attempt, isHost: isHost, myRole: myRole}, nil
}

// loadAttemptEditor requires an attempt member editing a live event.
func (h *Handlers) loadAttemptEditor(ctx context.Context, eventID, attemptID int64) (attemptAccess, error) {
	access, err := h.loadAttemptAccess(ctx, eventID, attemptID)
	if err != nil {
		return attemptAccess{}, err
	}
	if !access.isMine() {
		return attemptAccess{}, huma.NewError(http.StatusForbidden, "only attempt members can edit the workspace")
	}
	if access.attempt.EventStatus != string(models.EventStatusLive) {
		return attemptAccess{}, huma.NewError(http.StatusConflict, "the event is not live")
	}
	return access, nil
}

// DeploymentsHandler serves published event deployments under /deploy/<id>/.
func (h *Handlers) DeploymentsHandler() http.Handler {
	return h.deployments.Handler()
}

// EventStream is the raw SSE endpoint that pushes collaboration updates. It
// is registered on the root mux because huma does not model streaming.
func (h *Handlers) EventStream(w http.ResponseWriter, r *http.Request) {
	eventID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeProblem(w, http.StatusBadRequest, "invalid event id")
		return
	}
	attemptID, ok := parsePositiveID(r.PathValue("attemptId"))
	if !ok {
		writeProblem(w, http.StatusBadRequest, "invalid attempt id")
		return
	}
	u, err := middleware.CurrentUser(r.Context(), h.db)
	if err != nil {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return
	}
	attempt, err := database.GetAttemptInEvent(r.Context(), h.db, eventID, attemptID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "attempt not found")
		return
	}
	eventRole, err := database.GetEventMemberRole(r.Context(), h.db, eventID, u.ID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "could not authorize the stream")
		return
	}
	if eventRole == "" && u.Role != models.RoleAdmin {
		writeProblem(w, http.StatusForbidden, "you are not a member of this event")
		return
	}
	isHost := eventRole == models.EventRoleHost || u.Role == models.RoleAdmin
	myRole, err := database.GetAttemptMemberRole(r.Context(), h.db, attemptID, u.ID)
	if err != nil || (!isHost && myRole == "") {
		writeProblem(w, http.StatusForbidden, "you do not have access to this attempt")
		return
	}
	if err := h.ensureAttemptRoom(r.Context(), attempt); err != nil {
		writeProblem(w, http.StatusBadGateway, "could not load the workspace")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeProblem(w, http.StatusInternalServerError, "streaming is not available")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, unsubscribe := h.collab.Subscribe(attemptID, collab.User{ID: u.ID, Username: u.Username})
	defer unsubscribe()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Type, data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// eventOutput reloads an event for the given caller and overlays live dirty
// state from the collaboration rooms.
func (h *Handlers) eventOutput(ctx context.Context, id int64, u models.User) (*EventOutput, error) {
	role, err := database.GetEventMemberRole(ctx, h.db, id, u.ID)
	if err != nil {
		return nil, h.eventError(err)
	}
	isHost := role == models.EventRoleHost || u.Role == models.RoleAdmin
	event, err := database.GetEventDetail(ctx, h.db, id, h.cfg.PublicBaseURL, u.ID, isHost, isHost)
	if err != nil {
		return nil, h.eventError(err)
	}
	for ti := range event.Tasks {
		for ai := range event.Tasks[ti].Attempts {
			event.Tasks[ti].Attempts[ai].Dirty = h.collab.Dirty(event.Tasks[ti].Attempts[ai].ID)
		}
	}
	if !isHost {
		event.Members = []models.EventMember{}
	}
	resp := &EventOutput{}
	resp.Body.Event = event
	return resp, nil
}

// attemptOutput reloads one attempt for the given caller.
func (h *Handlers) attemptOutput(ctx context.Context, eventID, attemptID int64, u models.User, isHost bool) (*AttemptOutput, error) {
	attempt, err := database.GetAttemptView(ctx, h.db, attemptID, u.ID, isHost, h.cfg.PublicBaseURL)
	if err != nil {
		return nil, h.eventError(err)
	}
	attempt.Dirty = h.collab.Dirty(attemptID)
	resp := &AttemptOutput{}
	resp.Body.Attempt = attempt
	return resp, nil
}

// requireEventUser resolves the caller and verifies event access. The role is
// the caller's membership role; admins are treated as hosts.
func (h *Handlers) requireEventUser(ctx context.Context, eventID int64) (models.User, models.EventMemberRole, bool, error) {
	u, err := h.requireUser(ctx)
	if err != nil {
		return models.User{}, "", false, err
	}
	role, err := database.GetEventMemberRole(ctx, h.db, eventID, u.ID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return models.User{}, "", false, huma.NewError(http.StatusNotFound, "event not found")
		}
		log.Printf("event access: %v", err)
		return models.User{}, "", false, huma.NewError(http.StatusInternalServerError, "something went wrong")
	}
	if u.Role == models.RoleAdmin {
		role = models.EventRoleHost
	}
	if role == "" {
		return models.User{}, "", false, huma.NewError(http.StatusForbidden, "you are not a member of this event")
	}
	return u, role, role == models.EventRoleHost, nil
}

// requireEventHost requires the caller to be the host (or an admin).
func (h *Handlers) requireEventHost(ctx context.Context, eventID int64) (models.User, error) {
	u, _, isHost, err := h.requireEventUser(ctx, eventID)
	if err != nil {
		return models.User{}, err
	}
	if !isHost {
		return models.User{}, huma.NewError(http.StatusForbidden, "only the host can do this")
	}
	return u, nil
}

// workspaceOutput assembles the live workspace view of an attempt.
func (h *Handlers) workspaceOutput(ctx context.Context, attempt database.AttemptRow) (*WorkspaceOutput, error) {
	files, err := h.attemptSnapshotFiles(attempt)
	if err != nil {
		return nil, huma.NewError(http.StatusBadGateway, "could not read the workspace")
	}
	text := map[string]string{}
	nonText := []WorkspaceFile{}
	for path, data := range files {
		switch {
		case repos.IsBinary(data):
			nonText = append(nonText, WorkspaceFile{Path: path, Size: int64(len(data)), Binary: true})
		case len(data) > maxEventFileBytes:
			nonText = append(nonText, WorkspaceFile{Path: path, Size: int64(len(data)), Truncated: true})
		default:
			text[path] = string(data)
		}
	}
	docs := h.collab.LoadWorkspace(attempt.ID, text)
	resp := &WorkspaceOutput{}
	resp.Body.Docs = docs
	resp.Body.Files = nonText
	resp.Body.Deleted = h.collab.Deleted(attempt.ID)
	resp.Body.Dirty = h.collab.Dirty(attempt.ID)
	return resp, nil
}

// ensureAttemptRoom loads the git workspace into the live room once.
func (h *Handlers) ensureAttemptRoom(ctx context.Context, attempt database.AttemptRow) error {
	if h.collab.Loaded(attempt.ID) {
		return nil
	}
	files, err := h.attemptSnapshotFiles(attempt)
	if err != nil {
		return err
	}
	text := map[string]string{}
	for path, data := range files {
		if repos.IsBinary(data) || len(data) > maxEventFileBytes {
			continue
		}
		text[path] = string(data)
	}
	h.collab.LoadWorkspace(attempt.ID, text)
	return nil
}

// reloadAttemptRoom replaces the live room after an external push.
func (h *Handlers) reloadAttemptRoom(ctx context.Context, attempt database.AttemptRow) {
	files, err := h.attemptSnapshotFiles(attempt)
	if err != nil {
		log.Printf("reload attempt %d: %v", attempt.ID, err)
		return
	}
	text := map[string]string{}
	for path, data := range files {
		if repos.IsBinary(data) || len(data) > maxEventFileBytes {
			continue
		}
		text[path] = string(data)
	}
	h.collab.ReplaceWorkspace(attempt.ID, text)
}

// attemptSnapshotFiles returns every file of the attempt's main branch.
func (h *Handlers) attemptSnapshotFiles(attempt database.AttemptRow) (map[string][]byte, error) {
	dir, err := h.gitRepoDir(attempt.RepoPath)
	if err != nil {
		return nil, err
	}
	return repos.SnapshotFiles(dir)
}

// seedEventTask fills an empty attempt from the task definition, falling back
// to a starter app. It never overwrites an existing main branch.
func (h *Handlers) ensureAttemptWorkspace(ctx context.Context, author models.User, attempt database.AttemptRow) error {
	dir, err := h.gitRepoDir(attempt.RepoPath)
	if err != nil {
		return err
	}
	if _, ok := repoMainSHA(dir); ok {
		return nil
	}

	assessmentRepo := filepath.Join(h.cfg.GitReposDir, filepath.FromSlash(models.AssessmentRepoID(attempt.AssessmentID)+".git"))
	if sha, ok := repoMainSHA(assessmentRepo); ok {
		tmp, err := os.MkdirTemp("", "attempt-seed-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		if err := assessment.Materialize(ctx, assessmentRepo, sha, "", tmp); err != nil {
			return fmt.Errorf("materialize the task definition: %w", err)
		}
		if _, err := repos.CommitDir(dir, tmp, "Initial workspace from the task repository", gitIdentity(author)); err != nil {
			return err
		}
	} else {
		task, err := database.GetAssessmentDetail(ctx, h.db, attempt.AssessmentID)
		if err != nil {
			return err
		}
		files := starterWorkspace(task.Title, task.Description)
		if _, err := repos.CommitFiles(dir, files, "Create the starter workspace", gitIdentity(author)); err != nil {
			return err
		}
	}
	if err := h.deployAttempt(ctx, attempt); err != nil {
		log.Printf("seed attempt %d: deploy: %v", attempt.ID, err)
	}
	return nil
}

// deployAttempt publishes the main branch of one attempt.
func (h *Handlers) deployAttempt(ctx context.Context, attempt database.AttemptRow) error {
	dir, err := h.gitRepoDir(attempt.RepoPath)
	if err != nil {
		return err
	}
	sha, ok := repoMainSHA(dir)
	if !ok {
		return errors.New("the workspace has no main branch yet")
	}
	if err := database.SetAttemptDeploymentDeploying(ctx, h.db, attempt.ID); err != nil {
		log.Printf("deploy attempt %d: mark deploying: %v", attempt.ID, err)
	}
	if err := h.deployments.Deploy(ctx, attempt.PublicID, dir, sha); err != nil {
		if dbErr := database.SetAttemptDeploymentResult(ctx, h.db, attempt.ID, sha, "failed", err.Error()); dbErr != nil {
			log.Printf("deploy attempt %d: record failure: %v", attempt.ID, dbErr)
		}
		return err
	}
	if err := database.SetAttemptDeploymentResult(ctx, h.db, attempt.ID, sha, "live", ""); err != nil {
		return err
	}
	log.Printf("deploy attempt %d: live at /deploy/%s/ (%s)", attempt.ID, attempt.PublicID, shortSHA(sha))
	return nil
}

func (h *Handlers) eventError(err error) error {
	switch {
	case errors.Is(err, database.ErrNotFound):
		return huma.NewError(http.StatusNotFound, "event or attempt not found")
	case errors.Is(err, database.EventStatusError):
		return huma.NewError(http.StatusConflict, "the event is not in a state that allows this")
	default:
		log.Printf("event: %v", err)
		return huma.NewError(http.StatusInternalServerError, "something went wrong")
	}
}

// gitIdentity builds the commit signature for a user.
func gitIdentity(u models.User) gitobject.Signature {
	email := u.Email
	if email == "" {
		email = u.Username + "@event.local"
	}
	return gitobject.Signature{Name: u.Username, Email: email, When: time.Now()}
}

// repoMainSHA returns the commit main points at, or false when the repository
// has no main branch yet.
func repoMainSHA(dir string) (string, bool) {
	repo, err := repos.OpenDir(dir)
	if err != nil {
		return "", false
	}
	sha, err := repo.Revision("main")
	if err != nil {
		return "", false
	}
	return sha, true
}

// starterWorkspace is the app an attempt gets when its task definition
// repository was never pushed to.
func starterWorkspace(title, description string) map[string][]byte {
	readme := title + "\n\n" + description + "\n"
	page := `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>` + html.EscapeString(title) + `</title>
    <style>
      body { font-family: system-ui, sans-serif; max-width: 42rem; margin: 4rem auto; padding: 0 1.5rem; line-height: 1.6; }
      h1 { margin-bottom: .25rem; }
      p { color: #555; }
      code { background: #eee; padding: .15rem .35rem; border-radius: .25rem; }
    </style>
  </head>
  <body>
    <h1>` + html.EscapeString(title) + `</h1>
    <p>` + html.EscapeString(description) + `</p>
    <p>Edit this app in the browser, commit it, or push to <code>main</code>. Only commits deploy.</p>
    <script src="app.js"></script>
  </body>
</html>
`
	app := `// Only what is committed to main is deployed.\nconsole.log("starter app");\n`
	return map[string][]byte{
		"assessment.json":   []byte(`{"kind":"unbound"}`),
		"README.md":         []byte(readme),
		"deploy/index.html": []byte(page),
		"deploy/app.js":     []byte(app),
	}
}

func shortSHA(sha string) string {
	if len(sha) <= 8 {
		return sha
	}
	return sha[:8]
}
