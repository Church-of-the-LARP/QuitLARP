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

// ---- shared helpers -------------------------------------------------------

func unprocessable(msg string) error {
	return huma.NewError(http.StatusUnprocessableEntity, msg)
}

// normalizeDraft validates an assessment payload and returns a cleaned copy:
// whitespace is trimmed, unknown tags are fine (they are created on the fly)
// and duplicate/blank tag names are dropped.
func normalizeDraft(in models.AssessmentDraft) (models.AssessmentDraft, error) {
	d := in
	d.Title = strings.TrimSpace(d.Title)
	d.Description = strings.TrimSpace(d.Description)
	d.Template.FileName = strings.TrimSpace(d.Template.FileName)

	if d.Title == "" {
		return d, unprocessable("title is required")
	}
	if d.Description == "" {
		return d, unprocessable("description is required")
	}
	if !d.Difficulty.Valid() {
		return d, unprocessable("difficulty must be one of: easy, medium, hard")
	}
	if d.TimeLimitMinutes <= 0 {
		return d, unprocessable("timeLimitMinutes must be greater than zero")
	}
	if d.Template.FileName == "" {
		return d, unprocessable("template.fileName is required")
	}

	seen := map[string]bool{}
	tags := []string{}
	for _, raw := range d.Tags {
		name := strings.TrimSpace(raw)
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		tags = append(tags, name)
	}
	d.Tags = tags

	chapters := make([]models.ChapterDraft, 0, len(d.Chapters))
	for _, ch := range d.Chapters {
		ch.Title = strings.TrimSpace(ch.Title)
		ch.Description = strings.TrimSpace(ch.Description)
		if ch.Title == "" || ch.Description == "" || ch.TimeLimitMinutes <= 0 {
			return d, unprocessable("each chapter needs a title, a description and a positive timeLimitMinutes")
		}
		chapters = append(chapters, ch)
	}
	d.Chapters = chapters

	tests := make([]models.TestDraft, 0, len(d.Tests))
	for _, t := range d.Tests {
		t.Name = strings.TrimSpace(t.Name)
		t.Description = strings.TrimSpace(t.Description)
		t.File.FileName = strings.TrimSpace(t.File.FileName)
		if t.Name == "" || t.Description == "" || t.File.FileName == "" {
			return d, unprocessable("each test needs a name, a description and a file name")
		}
		tests = append(tests, t)
	}
	d.Tests = tests

	return d, nil
}

// requireManageableAssessment checks the caller is allowed to write to the
// assessment: its author, or an admin/superadmin. The assessment must exist.
func (h *Handlers) requireManageableAssessment(ctx context.Context, id int64) error {
	authorID, err := database.GetAssessmentAuthorID(ctx, h.db, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return huma.NewError(http.StatusNotFound, "assessment not found")
		}
		log.Printf("assessment author lookup: %v", err)
		return huma.NewError(http.StatusInternalServerError, "something went wrong")
	}
	u, err := h.requireUser(ctx)
	if err != nil {
		return err
	}
	if (authorID != nil && *authorID == u.ID) ||
		u.Role == models.RoleAdmin || u.Role == models.RoleSuperadmin {
		return nil
	}
	return huma.NewError(http.StatusForbidden, "you do not have permission to do this")
}

// ---- response/request types ----------------------------------------------

// AssessmentOutput is the response of the single-assessment endpoints.
type AssessmentOutput struct {
	Body struct {
		Assessment models.Assessment `json:"assessment"`
	}
}

// AssessmentRepoOutput is the response of GET /api/v1/assessments/{id}/repo.
type AssessmentRepoOutput struct {
	Body struct {
		RepoID string `json:"repoId" doc:"Repository path on the local git server"`
		URL    string `json:"url" doc:"Clone URL of the repository that backs the assessment"`
	}
}

// AssessmentListOutput is the response of GET /api/v1/assessments.
type AssessmentListOutput struct {
	Body struct {
		Assessments []models.AssessmentSummary `json:"assessments"`
		Total       int64                      `json:"total" doc:"Total number of assessments matching the filters"`
	}
}

// ListAssessmentsInput is the request of GET /api/v1/assessments.
type ListAssessmentsInput struct {
	Difficulty string `query:"difficulty" doc:"Filter by difficulty rating"`
	Tag        string `query:"tag" doc:"Filter by tag name (case-insensitive)"`
	Q          string `query:"q" doc:"Search term matched against title and description"`
	Offset     int    `query:"offset" example:"0" doc:"Number of results to skip"`
	Limit      int    `query:"limit" example:"50" doc:"Maximum number of results (1-100, default 50)"`
}

// CreateAssessmentInput is the request of POST /api/v1/assessments.
type CreateAssessmentInput struct {
	Body models.AssessmentDraft
}

// PatchAssessmentInput is the request of PATCH /api/v1/assessments/{id}.
type PatchAssessmentInput struct {
	ID   int64 `path:"id" example:"1" doc:"Assessment id to update"`
	Body struct {
		Title            *string            `json:"title,omitempty" doc:"Assessment title"`
		Description      *string            `json:"description,omitempty" doc:"What the candidate has to build"`
		Difficulty       *models.Difficulty `json:"difficulty,omitempty" enum:"easy,medium,hard" doc:"Difficulty rating"`
		TimeLimitMinutes *int               `json:"timeLimitMinutes,omitempty" example:"120" doc:"Time limit for the whole assessment, in minutes"`
		Template         *models.CodeFile   `json:"template,omitempty" doc:"Base template file the candidate starts from"`
	}
}

// ReplaceAssessmentTagsInput is the request of PUT
// /api/v1/assessments/{id}/tags.
type ReplaceAssessmentTagsInput struct {
	ID   int64 `path:"id" example:"1" doc:"Assessment id"`
	Body struct {
		Tags []string `json:"tags" doc:"Tag labels; unknown ones are created automatically"`
	}
}

// TagListOutput is the response of GET /api/v1/tags.
type TagListOutput struct {
	Body struct {
		Tags []models.TagCount `json:"tags"`
	}
}

// ---- routes ---------------------------------------------------------------

func (h *Handlers) registerAssessments(api huma.API) {
	// GET /api/v1/assessments — public browse with filters.
	huma.Register(api, huma.Operation{
		OperationID: "listAssessments",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments",
		Summary:     "List assessments",
		Description: "Public. Filters by difficulty, tag and free-text search; hidden test files are never included.",
	}, func(ctx context.Context, input *ListAssessmentsInput) (*AssessmentListOutput, error) {
		if input.Offset < 0 {
			return nil, unprocessable("offset must be zero or greater")
		}
		limit := input.Limit
		if limit == 0 {
			limit = 50
		}
		if limit < 1 || limit > 100 {
			return nil, unprocessable("limit must be between 1 and 100")
		}

		filter := database.AssessmentFilter{
			Tag:    strings.TrimSpace(input.Tag),
			Q:      strings.TrimSpace(input.Q),
			Offset: input.Offset,
			Limit:  limit,
		}
		if input.Difficulty != "" {
			d, ok := models.ParseDifficulty(strings.TrimSpace(input.Difficulty))
			if !ok {
				return nil, unprocessable("difficulty must be one of: easy, medium, hard")
			}
			filter.Difficulty = &d
		}

		items, err := database.ListAssessments(ctx, h.db, filter)
		if err != nil {
			log.Printf("list assessments: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list assessments")
		}
		total, err := database.CountAssessments(ctx, h.db, filter)
		if err != nil {
			log.Printf("count assessments: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list assessments")
		}
		resp := &AssessmentListOutput{}
		resp.Body.Assessments = items
		resp.Body.Total = total
		return resp, nil
	})

	// POST /api/v1/assessments — any signed-in user authors assessments.
	huma.Register(api, huma.Operation{
		OperationID: "createAssessment",
		Method:      http.MethodPost,
		Path:        "/api/v1/assessments",
		Summary:     "Create an assessment",
		Description: "Requires authentication. Chapters and hidden tests may be included; unknown tag names are created automatically.",
	}, func(ctx context.Context, input *CreateAssessmentInput) (*AssessmentOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		draft, err := normalizeDraft(input.Body)
		if err != nil {
			return nil, err
		}
		got, err := database.CreateAssessment(ctx, h.db, u.ID, draft)
		if err != nil {
			log.Printf("create assessment: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not create assessment")
		}
		resp := &AssessmentOutput{}
		resp.Body.Assessment = got
		return resp, nil
	})

	// GET /api/v1/assessments/{id} — public detail.
	huma.Register(api, huma.Operation{
		OperationID: "getAssessment",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments/{id}",
		Summary:     "Get an assessment",
		Description: "Public. Returns the template and chapter content plus the hidden tests as name/description summaries — never their files.",
	}, func(ctx context.Context, input *struct {
		ID int64 `path:"id" example:"1" doc:"Assessment id"`
	}) (*AssessmentOutput, error) {
		got, err := database.GetAssessmentDetail(ctx, h.db, input.ID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "assessment not found")
			}
			log.Printf("get assessment: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not load assessment")
		}
		resp := &AssessmentOutput{}
		resp.Body.Assessment = got
		return resp, nil
	})

	// GET /api/v1/assessments/{id}/repo: author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "getAssessmentRepo",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments/{id}/repo",
		Summary:     "Get the git repository of an assessment",
		Description: "The author of the assessment or an admin/superadmin may read the repository that backs it; the link is where the assignment content gets pushed.",
	}, func(ctx context.Context, input *struct {
		ID int64 `path:"id" example:"1" doc:"Assessment id"`
	}) (*AssessmentRepoOutput, error) {
		if err := h.requireManageableAssessment(ctx, input.ID); err != nil {
			return nil, err
		}
		repoID := models.AssessmentRepoID(input.ID)
		resp := &AssessmentRepoOutput{}
		resp.Body.RepoID = repoID
		resp.Body.URL = h.cfg.PublicBaseURL + "/git/" + repoID + ".git"
		return resp, nil
	})

	// PATCH /api/v1/assessments/{id} — author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "patchAssessment",
		Method:      http.MethodPatch,
		Path:        "/api/v1/assessments/{id}",
		Summary:     "Update an assessment",
		Description: "The author of the assessment or an admin/superadmin may update its scalar fields and template. Chapters, tests and tags have their own endpoints.",
	}, func(ctx context.Context, input *PatchAssessmentInput) (*AssessmentOutput, error) {
		if err := h.requireManageableAssessment(ctx, input.ID); err != nil {
			return nil, err
		}
		patch := database.AssessmentPatch{}
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
		if input.Body.Difficulty != nil {
			if !input.Body.Difficulty.Valid() {
				return nil, unprocessable("difficulty must be one of: easy, medium, hard")
			}
			patch.Difficulty = input.Body.Difficulty
		}
		if input.Body.TimeLimitMinutes != nil {
			if *input.Body.TimeLimitMinutes <= 0 {
				return nil, unprocessable("timeLimitMinutes must be greater than zero")
			}
			patch.TimeLimitMinutes = input.Body.TimeLimitMinutes
		}
		if input.Body.Template != nil {
			if strings.TrimSpace(input.Body.Template.FileName) == "" {
				return nil, unprocessable("template.fileName is required")
			}
			patch.Template = input.Body.Template
		}

		got, err := database.UpdateAssessment(ctx, h.db, input.ID, patch)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "assessment not found")
			}
			log.Printf("patch assessment: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update assessment")
		}
		resp := &AssessmentOutput{}
		resp.Body.Assessment = got
		return resp, nil
	})

	// DELETE /api/v1/assessments/{id} — author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "deleteAssessment",
		Method:      http.MethodDelete,
		Path:        "/api/v1/assessments/{id}",
		Summary:     "Delete an assessment",
		Description: "The author of the assessment or an admin/superadmin may delete it; its chapters, tests and tag links are removed with it.",
	}, func(ctx context.Context, input *struct {
		ID int64 `path:"id" example:"1" doc:"Assessment id to delete"`
	}) (*struct{}, error) {
		if err := h.requireManageableAssessment(ctx, input.ID); err != nil {
			return nil, err
		}
		if err := database.DeleteAssessment(ctx, h.db, input.ID); err != nil {
			log.Printf("delete assessment: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not delete assessment")
		}
		return nil, nil
	})

	// PUT /api/v1/assessments/{id}/tags — author or admin, full replacement.
	huma.Register(api, huma.Operation{
		OperationID: "replaceAssessmentTags",
		Method:      http.MethodPut,
		Path:        "/api/v1/assessments/{id}/tags",
		Summary:     "Replace the tags of an assessment",
		Description: "The author of the assessment or an admin/superadmin may set its full tag list. Unknown tag names are created automatically.",
	}, func(ctx context.Context, input *ReplaceAssessmentTagsInput) (*AssessmentOutput, error) {
		if err := h.requireManageableAssessment(ctx, input.ID); err != nil {
			return nil, err
		}
		if err := database.ReplaceAssessmentTags(ctx, h.db, input.ID, input.Body.Tags); err != nil {
			log.Printf("replace tags: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not replace tags")
		}
		got, err := database.GetAssessmentDetail(ctx, h.db, input.ID)
		if err != nil {
			log.Printf("reload assessment: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not load assessment")
		}
		resp := &AssessmentOutput{}
		resp.Body.Assessment = got
		return resp, nil
	})
}
