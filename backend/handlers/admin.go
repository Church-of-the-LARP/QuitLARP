package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
	"backend/models"
)

type AdminSessionsOutput struct {
	Body struct {
		Sessions []adminSession `json:"sessions"`
	}
}

type adminSession struct {
	ID              string `json:"id"`
	UserID          int64  `json:"userId"`
	AssessmentID    int64  `json:"assessmentId"`
	AssessmentTitle string `json:"assessmentTitle"`
	Deadline        string `json:"deadline"`
	CreatedAt       string `json:"createdAt"`
	Runtime         string `json:"runtime"`
}

type AdminSessionInput struct {
	ID string `path:"id" doc:"Live assessment session ID"`
}

type AdminSubmissionsOutput struct {
	Body struct {
		Submissions []adminSubmission `json:"submissions"`
	}
}

type adminSubmission struct {
	ID           int64             `json:"id"`
	AssessmentID int64             `json:"assessmentId"`
	Assessment   string            `json:"assessmentTitle"`
	UserID       int64             `json:"userId"`
	Author       *models.AuthorRef `json:"author"`
	Public       bool              `json:"public"`
	CommitSHA    string            `json:"commitSha"`
	URL          string            `json:"url"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}

func (h *Handlers) registerAdmin(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listAdminSessions",
		Method:      http.MethodGet,
		Path:        "/api/v1/admin/sessions",
		Summary:     "List live assessment sessions",
	}, func(ctx context.Context, _ *struct{}) (*AdminSessionsOutput, error) {
		if _, err := h.requireUser(ctx, models.RoleAdmin); err != nil {
			return nil, err
		}
		out := &AdminSessionsOutput{}
		for _, session := range h.sessions.AdminSnapshot() {
			out.Body.Sessions = append(out.Body.Sessions, adminSession{
				ID: session.ID, UserID: session.UserID, AssessmentID: session.AssessmentID,
				AssessmentTitle: session.AssessmentTitle,
				Deadline:        session.Deadline.UTC().Format("2006-01-02T15:04:05Z07:00"),
				CreatedAt:       session.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
				Runtime:         "runsc (gVisor)",
			})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "endAdminSession",
		Method:      http.MethodDelete,
		Path:        "/api/v1/admin/sessions/{id}",
		Summary:     "End a live assessment session",
	}, func(ctx context.Context, input *AdminSessionInput) (*struct{}, error) {
		if _, err := h.requireUser(ctx, models.RoleAdmin); err != nil {
			return nil, err
		}
		if err := h.sessions.AdminAbandon(input.ID); err != nil {
			return nil, huma.NewError(http.StatusNotFound, "live session not found")
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listAdminSubmissions",
		Method:      http.MethodGet,
		Path:        "/api/v1/admin/submissions",
		Summary:     "List submitted assessments",
	}, func(ctx context.Context, _ *struct{}) (*AdminSubmissionsOutput, error) {
		if _, err := h.requireUser(ctx, models.RoleAdmin); err != nil {
			return nil, err
		}
		rows, err := database.ListAllSolutions(ctx, h.db)
		if err != nil {
			log.Printf("list admin submissions: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list submissions")
		}
		out := &AdminSubmissionsOutput{}
		for _, row := range rows {
			out.Body.Submissions = append(out.Body.Submissions, adminSubmission{
				ID: row.ID, AssessmentID: row.AssessmentID, Assessment: row.AssessmentTitle,
				UserID: row.UserID, Author: row.Author, Public: row.Public,
				CommitSHA: row.CommitSHA,
				URL:       h.cfg.PublicBaseURL + "/git/" + models.SolutionRepoID(row.AssessmentID, row.UserID) + ".git",
				CreatedAt: row.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt: row.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			})
		}
		return out, nil
	})
}
