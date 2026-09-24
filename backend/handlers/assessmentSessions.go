package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
	"backend/models"
	"backend/sessions"
)

// assessmentIDInput is the request of every endpoint scoped to one
// assessment through its id.
type assessmentIDInput struct {
	ID int64 `path:"id" example:"1" doc:"Assessment id"`
}

// solveSessionBody is the session view the solve page works with.
type solveSessionBody struct {
	AssessmentID     int64  `json:"assessmentId"`
	ChapterID        int64  `json:"chapterId"`
	ChapterTitle     string `json:"chapterTitle"`
	TaskFile         string `json:"taskFile" doc:"Path of the task file within the repository"`
	FileName         string `json:"fileName" doc:"Name of the task file for display"`
	Content          string `json:"content" doc:"Current contents of the task file"`
	Deadline         string `json:"deadline" doc:"When the session ends, RFC 3339"`
	Status           string `json:"status" doc:"Session status"`
	TimeLimitMinutes *int   `json:"timeLimitMinutes" doc:"Assessment time limit, null when unlimited"`
}

// SolveSessionOutput is the response of the session endpoints.
type SolveSessionOutput struct {
	Body struct {
		Session solveSessionBody `json:"session"`
	}
}

// SolveTestsOutput is the response of the run-tests endpoint.
type SolveTestsOutput struct {
	Body struct {
		Results   []sessions.TestResult `json:"results"`
		Raw       string                `json:"raw" doc:"Raw harness output"`
		AllPassed bool                  `json:"allPassed"`
	}
}

// solutionBody is the solution view shared by the solution endpoints.
type solutionBody struct {
	ID           int64             `json:"id"`
	AssessmentID int64             `json:"assessmentId"`
	UserID       int64             `json:"userId"`
	Public       bool              `json:"public"`
	CommitSHA    string            `json:"commitSha"`
	URL          string            `json:"url" doc:"Clone URL on the local git server"`
	Author       *models.AuthorRef `json:"author"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// SolveSolutionOutput is the response of the endpoints returning one solution.
type SolveSolutionOutput struct {
	Body struct {
		Solution solutionBody `json:"solution"`
	}
}

// SolveSolutionListOutput is the response of the author's solution listing.
type SolveSolutionListOutput struct {
	Body struct {
		Solutions []solutionBody `json:"solutions"`
	}
}

func (h *Handlers) registerAssessmentSessions(api huma.API) {
	// POST /api/v1/assessments/{id}/sessions
	huma.Register(api, huma.Operation{
		OperationID: "startAssessmentSession",
		Method:      http.MethodPost,
		Path:        "/api/v1/assessments/{id}/sessions",
		Summary:     "Start solving an assessment",
		Description: "Requires authentication. Builds the assessment environment when needed, starts one sandboxed container for this user and returns the task file. Calling it again returns the running session.",
	}, func(ctx context.Context, input *assessmentIDInput) (*SolveSessionOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		session, err := h.sessions.Start(ctx, u, input.ID)
		if err != nil {
			return nil, sessionError(err)
		}
		return solveSessionOutput(session), nil
	})

	// GET /api/v1/assessments/{id}/sessions/current
	huma.Register(api, huma.Operation{
		OperationID: "getAssessmentSession",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments/{id}/sessions/current",
		Summary:     "Get the running solving session",
		Description: "Requires authentication. Returns the caller's running session for the assessment, or 404 when none is running.",
	}, func(ctx context.Context, input *assessmentIDInput) (*SolveSessionOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		session, ok := h.sessions.Current(u.ID, input.ID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "no solving session is running for this assessment")
		}
		return solveSessionOutput(session), nil
	})

	// PUT /api/v1/assessments/{id}/sessions/current/file
	huma.Register(api, huma.Operation{
		OperationID: "writeAssessmentSessionFile",
		Method:      http.MethodPut,
		Path:        "/api/v1/assessments/{id}/sessions/current/file",
		Summary:     "Write the task file",
		Description: "Requires authentication. Writes the editor contents into the session's container. Only the chapter's declared task file can be written.",
	}, func(ctx context.Context, input *writeTaskFileInput) (*SolveSessionOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		session, ok := h.sessions.Current(u.ID, input.ID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "no solving session is running for this assessment")
		}
		if err := h.sessions.WriteFile(session, input.Body.Content); err != nil {
			return nil, sessionError(err)
		}
		return solveSessionOutput(session), nil
	})

	// POST /api/v1/assessments/{id}/sessions/current/tests
	huma.Register(api, huma.Operation{
		OperationID: "runAssessmentSessionTests",
		Method:      http.MethodPost,
		Path:        "/api/v1/assessments/{id}/sessions/current/tests",
		Summary:     "Run the chapter checks",
		Description: "Requires authentication. Runs the chapter's compiled checks inside the sandbox against the current task file and returns the results.",
	}, func(ctx context.Context, input *assessmentIDInput) (*SolveTestsOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		session, ok := h.sessions.Current(u.ID, input.ID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "no solving session is running for this assessment")
		}
		report, err := h.sessions.RunTests(ctx, session)
		if err != nil {
			return nil, sessionError(err)
		}
		resp := &SolveTestsOutput{}
		resp.Body.Results = report.Results
		resp.Body.Raw = report.Raw
		resp.Body.AllPassed = report.AllPassed
		return resp, nil
	})

	// POST /api/v1/assessments/{id}/sessions/current/submit
	huma.Register(api, huma.Operation{
		OperationID: "submitAssessmentSession",
		Method:      http.MethodPost,
		Path:        "/api/v1/assessments/{id}/sessions/current/submit",
		Summary:     "Submit the solution",
		Description: "Requires authentication. Persists the current task file as the caller's solution repository, ends the sandbox and returns the stored solution.",
	}, func(ctx context.Context, input *submitSolutionInput) (*SolveSolutionOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		session, ok := h.sessions.Current(u.ID, input.ID)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "no solving session is running for this assessment")
		}
		solution, err := h.sessions.Submit(ctx, u, session, input.Body.Public)
		if err != nil {
			return nil, sessionError(err)
		}
		resp := &SolveSolutionOutput{}
		resp.Body.Solution = h.solutionBody(solution)
		return resp, nil
	})

	// DELETE /api/v1/assessments/{id}/sessions/current
	huma.Register(api, huma.Operation{
		OperationID: "abandonAssessmentSession",
		Method:      http.MethodDelete,
		Path:        "/api/v1/assessments/{id}/sessions/current",
		Summary:     "End the solving session",
		Description: "Requires authentication. Ends the caller's session and removes its container without persisting anything. Ending a session that is not running is a no-op.",
	}, func(ctx context.Context, input *assessmentIDInput) (*struct{}, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		session, ok := h.sessions.Current(u.ID, input.ID)
		if !ok {
			return nil, nil
		}
		if err := h.sessions.Abandon(session); err != nil {
			log.Printf("abandon session %s: %v", session.ID, err)
		}
		return nil, nil
	})

	// GET /api/v1/assessments/{id}/solutions
	huma.Register(api, huma.Operation{
		OperationID: "listAssessmentSolutions",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments/{id}/solutions",
		Summary:     "List the solutions of an assessment",
		Description: "The author of the assessment or an admin/superadmin may list every stored solution with its solver and visibility.",
	}, func(ctx context.Context, input *assessmentIDInput) (*SolveSolutionListOutput, error) {
		if err := h.requireManageableAssessment(ctx, input.ID); err != nil {
			return nil, err
		}
		list, err := database.ListSolutions(ctx, h.db, input.ID)
		if err != nil {
			log.Printf("list solutions: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list solutions")
		}
		resp := &SolveSolutionListOutput{}
		resp.Body.Solutions = make([]solutionBody, 0, len(list))
		for _, solution := range list {
			resp.Body.Solutions = append(resp.Body.Solutions, h.solutionBody(solution))
		}
		return resp, nil
	})

	// GET /api/v1/assessments/{id}/solutions/current
	huma.Register(api, huma.Operation{
		OperationID: "getMyAssessmentSolution",
		Method:      http.MethodGet,
		Path:        "/api/v1/assessments/{id}/solutions/current",
		Summary:     "Get my solution",
		Description: "Requires authentication. Returns the caller's own stored solution for the assessment, or 404 when there is none.",
	}, func(ctx context.Context, input *assessmentIDInput) (*SolveSolutionOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		solution, err := database.GetSolution(ctx, h.db, input.ID, u.ID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "you have not submitted a solution for this assessment")
			}
			log.Printf("get solution: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not load the solution")
		}
		resp := &SolveSolutionOutput{}
		resp.Body.Solution = h.solutionBody(solution)
		return resp, nil
	})

	// PATCH /api/v1/assessments/{id}/solutions/current
	huma.Register(api, huma.Operation{
		OperationID: "updateMyAssessmentSolution",
		Method:      http.MethodPatch,
		Path:        "/api/v1/assessments/{id}/solutions/current",
		Summary:     "Change my solution's visibility",
		Description: "Requires authentication. The solver decides whether their solution is public; public solutions may be read by any signed-in user.",
	}, func(ctx context.Context, input *patchMySolutionInput) (*SolveSolutionOutput, error) {
		u, err := h.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		solution, err := database.GetSolution(ctx, h.db, input.ID, u.ID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, "you have not submitted a solution for this assessment")
			}
			log.Printf("get solution: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not load the solution")
		}
		updated, err := database.SetSolutionPublic(ctx, h.db, solution.ID, input.Body.Public)
		if err != nil {
			log.Printf("set solution visibility: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not update the solution")
		}
		resp := &SolveSolutionOutput{}
		resp.Body.Solution = h.solutionBody(updated)
		return resp, nil
	})
}

// writeTaskFileInput is the request of PUT
// /api/v1/assessments/{id}/sessions/current/file.
type writeTaskFileInput struct {
	ID   int64 `path:"id" example:"1" doc:"Assessment id"`
	Body struct {
		Content string `json:"content" doc:"Full contents of the task file"`
	}
}

// submitSolutionInput is the request of POST
// /api/v1/assessments/{id}/sessions/current/submit.
type submitSolutionInput struct {
	ID   int64 `path:"id" example:"1" doc:"Assessment id"`
	Body struct {
		Public bool `json:"public" doc:"Whether every signed-in user may read the solution"`
	}
}

// patchMySolutionInput is the request of PATCH
// /api/v1/assessments/{id}/solutions/current.
type patchMySolutionInput struct {
	ID   int64 `path:"id" example:"1" doc:"Assessment id"`
	Body struct {
		Public bool `json:"public" doc:"Whether every signed-in user may read the solution"`
	}
}

func solveSessionOutput(session *sessions.Session) *SolveSessionOutput {
	resp := &SolveSessionOutput{}
	resp.Body.Session = solveSessionBody{
		AssessmentID: session.AssessmentID,
		ChapterID:    session.ChapterID,
		ChapterTitle: session.ChapterTitle,
		TaskFile:     session.TaskFile,
		FileName:     path.Base(session.TaskFile),
		Content:      session.Content,
		Deadline:     session.Deadline.UTC().Format(time.RFC3339),
		Status:       "running",
	}
	if session.TimeLimit > 0 {
		limit := session.TimeLimit
		resp.Body.Session.TimeLimitMinutes = &limit
	}
	return resp
}

func (h *Handlers) solutionBody(solution models.Solution) solutionBody {
	return solutionBody{
		ID:           solution.ID,
		AssessmentID: solution.AssessmentID,
		UserID:       solution.UserID,
		Public:       solution.Public,
		CommitSHA:    solution.CommitSHA,
		URL:          h.cfg.PublicBaseURL + "/git/" + models.SolutionRepoID(solution.AssessmentID, solution.UserID) + ".git",
		Author:       solution.Author,
		CreatedAt:    solution.CreatedAt,
		UpdatedAt:    solution.UpdatedAt,
	}
}

// sessionError maps session and environment failures to huma errors.
func sessionError(err error) error {
	switch {
	case errors.Is(err, sessions.ErrAssessmentGone), errors.Is(err, database.ErrNotFound):
		return huma.NewError(http.StatusNotFound, "assessment not found")
	case errors.Is(err, sessions.ErrUnsupported):
		return huma.NewError(http.StatusUnprocessableEntity, "this assessment kind is not supported yet")
	case errors.Is(err, sessions.ErrNotRunnable):
		return huma.NewError(http.StatusUnprocessableEntity, "this assessment has no solvable chapter")
	case errors.Is(err, sessions.ErrNotSynced):
		return huma.NewError(http.StatusConflict, "the assessment repository has not been pushed to yet")
	case errors.Is(err, sessions.ErrSessionGone):
		return huma.NewError(http.StatusGone, "the solving session has ended; start a new one")
	case errors.Is(err, sessions.ErrFileTooLarge):
		return huma.NewError(http.StatusUnprocessableEntity, "the file is larger than the editor allows")
	case errors.Is(err, sessions.ErrTaskFileOnly):
		return huma.NewError(http.StatusUnprocessableEntity, "only the declared task file can be written")
	default:
		log.Printf("solve session: %v", err)
		return huma.NewError(http.StatusBadGateway, "the environment operation failed: "+err.Error())
	}
}
