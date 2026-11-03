package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
	"backend/models"
)

// ScheduledListOutput is the response of GET /api/v1/scheduled-assessments.
type ScheduledListOutput struct {
	Body struct {
		Scheduled []models.ScheduledAssessment `json:"scheduled"`
	}
}

// ScheduledOutput is the response of POST /api/v1/scheduled-assessments.
type ScheduledOutput struct {
	Body struct {
		Scheduled models.ScheduledAssessment `json:"scheduled"`
	}
}

func (h *Handlers) registerSchedule(api huma.API) {
	// GET /api/v1/scheduled-assessments — the caller's upcoming tests.
	huma.Register(api, huma.Operation{
		OperationID: "listScheduledAssessments",
		Method:      http.MethodGet,
		Path:        "/api/v1/scheduled-assessments",
		Summary:     "List upcoming assessments",
		Description: "Requires authentication. Returns the caller's scheduled assessments that have not started yet, soonest first.",
	}, func(ctx context.Context, input *struct {
		Limit int `query:"limit" example:"20" doc:"Maximum number of entries (1-100, default 20)"`
	}) (*ScheduledListOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		limit := input.Limit
		if limit == 0 {
			limit = 20
		}
		if limit < 1 || limit > 100 {
			return nil, unprocessable("limit must be between 1 and 100")
		}
		items, err := database.ListUpcomingAssessments(ctx, h.db, u.ID, limit)
		if err != nil {
			log.Printf("list upcoming assessments: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list upcoming assessments")
		}
		resp := &ScheduledListOutput{}
		resp.Body.Scheduled = items
		return resp, nil
	})

	// POST /api/v1/scheduled-assessments — plan (or reschedule) a test.
	huma.Register(api, huma.Operation{
		OperationID: "scheduleAssessment",
		Method:      http.MethodPost,
		Path:        "/api/v1/scheduled-assessments",
		Summary:     "Schedule an assessment",
		Description: "Requires authentication. Plans an assessment for a future time; scheduling the same assessment again moves the existing entry.",
	}, func(ctx context.Context, input *struct {
		Body struct {
			AssessmentID int64     `json:"assessmentId" example:"1" doc:"Assessment to schedule"`
			ScheduledAt  time.Time `json:"scheduledAt" doc:"When to take it (RFC 3339); must be in the future"`
		}
	}) (*ScheduledOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if !input.Body.ScheduledAt.After(time.Now()) {
			return nil, unprocessable("scheduledAt must be in the future")
		}
		got, err := database.SaveScheduledAssessment(ctx, h.db, u.ID, input.Body.AssessmentID, input.Body.ScheduledAt)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "assessment not found")
			}
			log.Printf("schedule assessment: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not schedule assessment")
		}
		resp := &ScheduledOutput{}
		resp.Body.Scheduled = got
		return resp, nil
	})

	// DELETE /api/v1/scheduled-assessments/{id} — drop an entry.
	huma.Register(api, huma.Operation{
		OperationID: "unscheduleAssessment",
		Method:      http.MethodDelete,
		Path:        "/api/v1/scheduled-assessments/{id}",
		Summary:     "Remove a scheduled assessment",
		Description: "Requires authentication. Removes one of the caller's own schedule entries.",
	}, func(ctx context.Context, input *struct {
		ID int64 `path:"id" example:"1" doc:"Schedule entry id"`
	}) (*struct{}, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := database.DeleteScheduledAssessment(ctx, h.db, u.ID, input.ID); err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "scheduled assessment not found")
			}
			log.Printf("unschedule assessment: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not remove scheduled assessment")
		}
		return nil, nil
	})
}
