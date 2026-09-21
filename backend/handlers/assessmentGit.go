package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path/filepath"

	"github.com/danielgtaylor/huma/v2"

	"backend/models"
	"backend/repos"
)

// AssessmentGitEntryOutput is one row of a repository directory listing.
type AssessmentGitEntryOutput struct {
	Name string `json:"name"`
	Type string `json:"type" enum:"tree,blob" doc:"tree for a directory, blob for a file"`
	Size int64  `json:"size" doc:"Size in bytes; zero for directories"`
	Hash string `json:"hash" doc:"Object id of the entry"`
}

// AssessmentGitTreeOutput is the response of GET
// /api/v1/assessments/{id}/repo/tree.
type AssessmentGitTreeOutput struct {
	Body struct {
		Ref     string                     `json:"ref" doc:"Revision that was listed"`
		Path    string                     `json:"path" doc:"Directory path that was listed"`
		Entries []AssessmentGitEntryOutput `json:"entries"`
		Empty   bool                       `json:"empty" doc:"True when the repository has no commits yet"`
	}
}

// AssessmentGitFileOutput is the response of GET
// /api/v1/assessments/{id}/repo/file.
type AssessmentGitFileOutput struct {
	Body struct {
		Ref       string `json:"ref" doc:"Revision the file was read from"`
		Path      string `json:"path" doc:"File path within the repository"`
		Size      int64  `json:"size" doc:"Size of the file in bytes"`
		Hash      string `json:"hash" doc:"Object id of the file"`
		Binary    bool   `json:"binary" doc:"True when the file looks binary; content is then empty"`
		Truncated bool   `json:"truncated" doc:"True when the file was larger than the display limit; content is then empty"`
		Content   string `json:"content" doc:"UTF-8 file content"`
	}
}

func (h *Handlers) registerAssessmentGit(api huma.API) {
	// GET /api/v1/assessments/{id}/repo/tree - author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "getAssessmentRepoTree",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments/{id}/repo/tree",
		Summary:     "List a directory of an assessment repository",
		Description: "The author of the assessment or an admin/superadmin may browse the repository that backs it; a repository with no commits yet returns an empty listing.",
	}, func(ctx context.Context, input *assessmentGitTreeInput) (*AssessmentGitTreeOutput, error) {
		if err := h.requireManageableAssessment(ctx, input.ID); err != nil {
			return nil, err
		}
		repo, err := h.openAssessmentRepo(input.ID)
		if err != nil {
			if errors.Is(err, repos.ErrRepoMissing) {
				return emptyGitTree(), nil
			}
			return nil, gitBrowseError(err)
		}
		tree, err := repo.Tree(input.Ref, input.Path)
		if err != nil {
			return nil, gitBrowseError(err)
		}
		resp := &AssessmentGitTreeOutput{}
		resp.Body.Ref = tree.Ref
		resp.Body.Path = tree.Path
		resp.Body.Empty = tree.Empty
		resp.Body.Entries = make([]AssessmentGitEntryOutput, 0, len(tree.Entries))
		for _, e := range tree.Entries {
			resp.Body.Entries = append(resp.Body.Entries, AssessmentGitEntryOutput{
				Name: e.Name,
				Type: e.Type,
				Size: e.Size,
				Hash: e.Hash,
			})
		}
		return resp, nil
	})

	// GET /api/v1/assessments/{id}/repo/file - author or admin.
	huma.Register(api, huma.Operation{
		OperationID: "getAssessmentRepoFile",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments/{id}/repo/file",
		Summary:     "Read a file from an assessment repository",
		Description: "The author of the assessment or an admin/superadmin may read a file in the repository that backs it; binary and oversized files return metadata only.",
	}, func(ctx context.Context, input *assessmentGitFileInput) (*AssessmentGitFileOutput, error) {
		if err := h.requireManageableAssessment(ctx, input.ID); err != nil {
			return nil, err
		}
		repo, err := h.openAssessmentRepo(input.ID)
		if err != nil {
			return nil, gitBrowseError(err)
		}
		file, err := repo.File(input.Ref, input.Path)
		if err != nil {
			return nil, gitBrowseError(err)
		}
		resp := &AssessmentGitFileOutput{}
		resp.Body.Ref = file.Ref
		resp.Body.Path = file.Path
		resp.Body.Size = file.Size
		resp.Body.Hash = file.Hash
		resp.Body.Binary = file.Binary
		resp.Body.Truncated = file.Truncated
		resp.Body.Content = file.Content
		return resp, nil
	})
}

// assessmentGitTreeInput is the request of GET
// /api/v1/assessments/{id}/repo/tree.
type assessmentGitTreeInput struct {
	ID   int64  `path:"id" example:"1" doc:"Assessment id"`
	Ref  string `query:"ref" doc:"Branch, tag or commit to browse; defaults to the repository HEAD"`
	Path string `query:"path" doc:"Directory path within the repository; defaults to the root"`
}

// assessmentGitFileInput is the request of GET
// /api/v1/assessments/{id}/repo/file.
type assessmentGitFileInput struct {
	ID   int64  `path:"id" example:"1" doc:"Assessment id"`
	Ref  string `query:"ref" doc:"Branch, tag or commit to read from; defaults to the repository HEAD"`
	Path string `query:"path" doc:"File path within the repository"`
}

// openAssessmentRepo opens the assessment repository for reading; it never
// creates it.
func (h *Handlers) openAssessmentRepo(id int64) (*repos.Repo, error) {
	repoID := models.AssessmentRepoID(id)
	if len(repoID) > 255 || !repoPathPattern.MatchString(repoID) {
		return nil, errInvalidRepo
	}
	dir := filepath.Join(h.cfg.GitReposDir, filepath.FromSlash(repoID+".git"))
	return repos.OpenDir(dir)
}

func emptyGitTree() *AssessmentGitTreeOutput {
	resp := &AssessmentGitTreeOutput{}
	resp.Body.Entries = []AssessmentGitEntryOutput{}
	resp.Body.Empty = true
	return resp
}

// gitBrowseError maps a read failure to a huma error.
func gitBrowseError(err error) error {
	switch {
	case errors.Is(err, repos.ErrRepoMissing):
		return huma.NewError(http.StatusNotFound, "the repository has not been created yet")
	case errors.Is(err, repos.ErrRefNotFound):
		return huma.NewError(http.StatusNotFound, "no such branch, tag or commit")
	case errors.Is(err, repos.ErrPathNotFound):
		return huma.NewError(http.StatusNotFound, "no such file or directory in this revision")
	case errors.Is(err, errInvalidRepo):
		log.Printf("assessment repository name: %v", err)
		return huma.NewError(http.StatusInternalServerError, "something went wrong")
	default:
		log.Printf("read assessment repository: %v", err)
		return huma.NewError(http.StatusInternalServerError, "could not read the repository")
	}
}
