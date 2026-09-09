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

// TestOutput is the response of the hidden-test endpoints. The testing file
// is only ever returned here, never by the public assessment detail.
type TestOutput struct {
	Body struct {
		Test models.Test `json:"test"`
	}
}

// AddTestInput is the request of POST /api/v1/assessments/{id}/tests.
type AddTestInput struct {
	AssessmentID int64 `path:"id" example:"1" doc:"Assessment id to extend"`
	Body         models.TestDraft
}

// TestPathInput carries the two path ids of the nested test endpoints.
type TestPathInput struct {
	AssessmentID int64 `path:"id" example:"1" doc:"Assessment id"`
	TestID       int64 `path:"testId" example:"1" doc:"Test id"`
}

// PatchTestInput is the request of PATCH .../tests/{testId}.
type PatchTestInput struct {
	AssessmentID int64 `path:"id" example:"1" doc:"Assessment id"`
	TestID       int64 `path:"testId" example:"1" doc:"Test id to update"`
	Body         struct {
		Name        *string          `json:"name,omitempty" doc:"Test name"`
		Description *string          `json:"description,omitempty" doc:"What this test verifies"`
		File        *models.CodeFile `json:"file,omitempty" doc:"The hidden testing file"`
	}
}

func (h *Handlers) registerAssessmentTests(api huma.API) {
	// POST /api/v1/assessments/{id}/tests — author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "addAssessmentTest",
		Method:      http.MethodPost,
		Path:        "/api/v1/assessments/{id}/tests",
		Summary:     "Add a hidden test to an assessment",
		Description: "The author of the assessment or an admin/superadmin may add a hidden test with its testing file. Candidates only ever see the name and description.",
	}, func(ctx context.Context, input *AddTestInput) (*TestOutput, error) {
		if err := h.requireManageableAssessment(ctx, input.AssessmentID); err != nil {
			return nil, err
		}
		draft := models.TestDraft{
			Name:        strings.TrimSpace(input.Body.Name),
			Description: strings.TrimSpace(input.Body.Description),
			File: models.CodeFile{
				FileName: strings.TrimSpace(input.Body.File.FileName),
				Content:  input.Body.File.Content,
			},
		}
		if draft.Name == "" || draft.Description == "" {
			return nil, unprocessable("name and description are required")
		}
		if draft.File.FileName == "" {
			return nil, unprocessable("file.fileName is required")
		}

		got, err := database.AddTest(ctx, h.db, input.AssessmentID, draft)
		if err != nil {
			if database.IsForeignKeyViolation(err) {
				return nil, huma.NewError(http.StatusNotFound, "assessment not found")
			}
			log.Printf("add test: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not add test")
		}
		resp := &TestOutput{}
		resp.Body.Test = got
		return resp, nil
	})

	// GET /api/v1/assessments/{id}/tests/{testId} — author or admin only,
	// because this is the endpoint that reveals the hidden testing file.
	huma.Register(api, huma.Operation{
		OperationID: "getAssessmentTest",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments/{id}/tests/{testId}",
		Summary:     "Get a hidden test (author or admin)",
		Description: "Restricted to the author of the assessment or an admin/superadmin: returns the hidden testing file. Everyone else may only see the test summaries on the assessment detail.",
	}, func(ctx context.Context, input *TestPathInput) (*TestOutput, error) {
		t, err := database.GetTest(ctx, h.db, input.TestID)
		if err != nil || t.AssessmentID != input.AssessmentID {
			if errors.Is(err, database.ErrNotFound) || t.AssessmentID != input.AssessmentID {
				return nil, huma.NewError(http.StatusNotFound, "test not found")
			}
			log.Printf("test lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not load test")
		}
		if err := h.requireManageableAssessment(ctx, t.AssessmentID); err != nil {
			return nil, err
		}
		resp := &TestOutput{}
		resp.Body.Test = t
		return resp, nil
	})

	// PATCH /api/v1/assessments/{id}/tests/{testId} — author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "patchAssessmentTest",
		Method:      http.MethodPatch,
		Path:        "/api/v1/assessments/{id}/tests/{testId}",
		Summary:     "Update a hidden test",
		Description: "The author of the assessment or an admin/superadmin may change the test name, description or hidden testing file.",
	}, func(ctx context.Context, input *PatchTestInput) (*TestOutput, error) {
		t, err := database.GetTest(ctx, h.db, input.TestID)
		if err != nil || t.AssessmentID != input.AssessmentID {
			if errors.Is(err, database.ErrNotFound) || t.AssessmentID != input.AssessmentID {
				return nil, huma.NewError(http.StatusNotFound, "test not found")
			}
			log.Printf("test lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update test")
		}
		if err := h.requireManageableAssessment(ctx, t.AssessmentID); err != nil {
			return nil, err
		}

		patch := database.TestPatch{}
		if input.Body.Name != nil {
			v := strings.TrimSpace(*input.Body.Name)
			if v == "" {
				return nil, unprocessable("name cannot be blank")
			}
			patch.Name = &v
		}
		if input.Body.Description != nil {
			v := strings.TrimSpace(*input.Body.Description)
			if v == "" {
				return nil, unprocessable("description cannot be blank")
			}
			patch.Description = &v
		}
		if input.Body.File != nil {
			if strings.TrimSpace(input.Body.File.FileName) == "" {
				return nil, unprocessable("file.fileName is required")
			}
			patch.File = input.Body.File
		}

		got, err := database.UpdateTest(ctx, h.db, input.TestID, patch)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "test not found")
			}
			log.Printf("patch test: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update test")
		}
		resp := &TestOutput{}
		resp.Body.Test = got
		return resp, nil
	})

	// DELETE /api/v1/assessments/{id}/tests/{testId} — author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "deleteAssessmentTest",
		Method:      http.MethodDelete,
		Path:        "/api/v1/assessments/{id}/tests/{testId}",
		Summary:     "Delete a hidden test",
		Description: "The author of the assessment or an admin/superadmin may remove a hidden test.",
	}, func(ctx context.Context, input *TestPathInput) (*struct{}, error) {
		t, err := database.GetTest(ctx, h.db, input.TestID)
		if err != nil || t.AssessmentID != input.AssessmentID {
			if errors.Is(err, database.ErrNotFound) || t.AssessmentID != input.AssessmentID {
				return nil, huma.NewError(http.StatusNotFound, "test not found")
			}
			log.Printf("test lookup: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not delete test")
		}
		if err := h.requireManageableAssessment(ctx, t.AssessmentID); err != nil {
			return nil, err
		}
		if err := database.DeleteTest(ctx, h.db, input.TestID); err != nil {
			log.Printf("delete test: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not delete test")
		}
		return nil, nil
	})
}
