package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
	"backend/models"
)

// ChapterOutput is the response of the single-chapter endpoints.
type ChapterOutput struct {
	Body struct {
		Chapter models.Chapter `json:"chapter"`
	}
}

// AddChapterInput is the request of POST /api/v1/assessments/{id}/chapters.
type AddChapterInput struct {
	AssessmentID int64 `path:"id" example:"1" doc:"Assessment id to extend"`
	Body         models.ChapterDraft
}

// PatchChapterInput is the request of
// PATCH /api/v1/assessments/{id}/chapters/{chapterId}.
type PatchChapterInput struct {
	AssessmentID int64 `path:"id" example:"1" doc:"Assessment id"`
	ChapterID    int64 `path:"chapterId" example:"1" doc:"Chapter id to update"`
	Body         struct {
		Title            *string `json:"title,omitempty" doc:"Chapter title"`
		Description      *string `json:"description,omitempty" doc:"Chapter body text"`
		TimeLimitMinutes *int    `json:"timeLimitMinutes,omitempty" example:"30" doc:"Time limit for this chapter, in minutes"`
		Position         *int    `json:"position,omitempty" example:"2" doc:"1-based ordering hint within the assessment"`
	}
}

// DeleteChapterInput is the request of
// DELETE /api/v1/assessments/{id}/chapters/{chapterId}.
type DeleteChapterInput struct {
	AssessmentID int64 `path:"id" example:"1" doc:"Assessment id"`
	ChapterID    int64 `path:"chapterId" example:"1" doc:"Chapter id to delete"`
}

func (h *Handlers) registerAssessmentChapters(api huma.API) {
	// POST /api/v1/assessments/{id}/chapters — author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "addAssessmentChapter",
		Method:      http.MethodPost,
		Path:        "/api/v1/assessments/{id}/chapters",
		Summary:     "Add a chapter to an assessment",
		Description: "The author of the assessment or an admin/superadmin may append a chapter; it is placed after the current last one.",
	}, func(ctx context.Context, input *AddChapterInput) (*ChapterOutput, error) {
		if _, err := h.requireManageableAssessment(ctx, input.AssessmentID); err != nil {
			return nil, err
		}
		draft := models.ChapterDraft{
			Title:            strings.TrimSpace(input.Body.Title),
			Description:      strings.TrimSpace(input.Body.Description),
			TimeLimitMinutes: input.Body.TimeLimitMinutes,
		}
		if draft.Title == "" || draft.Description == "" {
			return nil, unprocessable("title and description are required")
		}
		if draft.TimeLimitMinutes <= 0 {
			return nil, unprocessable("timeLimitMinutes must be greater than zero")
		}

		got, err := database.AddChapter(ctx, h.db, input.AssessmentID, draft)
		if err != nil {
			if database.IsForeignKeyViolation(err) {
				return nil, huma.NewError(http.StatusNotFound, "assessment not found")
			}
			log.Printf("add chapter: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not add chapter")
		}
		resp := &ChapterOutput{}
		resp.Body.Chapter = got
		return resp, nil
	})

	// PATCH /api/v1/assessments/{id}/chapters/{chapterId} — author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "patchAssessmentChapter",
		Method:      http.MethodPatch,
		Path:        "/api/v1/assessments/{id}/chapters/{chapterId}",
		Summary:     "Update a chapter",
		Description: "The author of the assessment or an admin/superadmin may change the chapter title, description, time limit and position.",
	}, func(ctx context.Context, input *PatchChapterInput) (*ChapterOutput, error) {
		ch, err := database.GetChapter(ctx, h.db, input.ChapterID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) || ch.AssessmentID != input.AssessmentID {
				return nil, huma.NewError(http.StatusNotFound, "chapter not found")
			}
			log.Printf("chapter lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update chapter")
		}
		if _, err := h.requireManageableAssessment(ctx, ch.AssessmentID); err != nil {
			return nil, err
		}

		patch := database.ChapterPatch{}
		if input.Body.Title != nil {
			v := strings.TrimSpace(*input.Body.Title)
			if v == "" {
				return nil, unprocessable("title cannot be blank")
			}
			patch.Title = &v
		}
		if input.Body.Description != nil {
			v := strings.TrimSpace(*input.Body.Description)
			if v == "" {
				return nil, unprocessable("description cannot be blank")
			}
			patch.Description = &v
		}
		if input.Body.TimeLimitMinutes != nil {
			if *input.Body.TimeLimitMinutes <= 0 {
				return nil, unprocessable("timeLimitMinutes must be greater than zero")
			}
			patch.TimeLimitMinutes = input.Body.TimeLimitMinutes
		}
		if input.Body.Position != nil {
			if *input.Body.Position < 1 {
				return nil, unprocessable("position must be 1 or greater")
			}
			patch.Position = input.Body.Position
		}

		got, err := database.UpdateChapter(ctx, h.db, input.ChapterID, patch)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "chapter not found")
			}
			log.Printf("patch chapter: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update chapter")
		}
		resp := &ChapterOutput{}
		resp.Body.Chapter = got
		return resp, nil
	})

	// DELETE /api/v1/assessments/{id}/chapters/{chapterId} — author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "deleteAssessmentChapter",
		Method:      http.MethodDelete,
		Path:        "/api/v1/assessments/{id}/chapters/{chapterId}",
		Summary:     "Delete a chapter",
		Description: "The author of the assessment or an admin/superadmin may remove a chapter.",
	}, func(ctx context.Context, input *DeleteChapterInput) (*struct{}, error) {
		ch, err := database.GetChapter(ctx, h.db, input.ChapterID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) || ch.AssessmentID != input.AssessmentID {
				return nil, huma.NewError(http.StatusNotFound, "chapter not found")
			}
			log.Printf("chapter lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not delete chapter")
		}
		if _, err := h.requireManageableAssessment(ctx, ch.AssessmentID); err != nil {
			return nil, err
		}
		if err := database.DeleteChapter(ctx, h.db, input.ChapterID); err != nil {
			if errors.Is(err, database.ErrLastChapter) {
				return nil, unprocessable("cannot delete an assessment's only chapter")
			}
			log.Printf("delete chapter: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not delete chapter")
		}
		return nil, nil
	})
}
