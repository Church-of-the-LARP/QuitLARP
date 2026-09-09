package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"backend/database"
)

func (h *Handlers) registerTags(api huma.API) {
	// GET /api/v1/tags — public.
	huma.Register(api, huma.Operation{
		OperationID: "listTags",
		Method:      http.MethodGet,
		Path:        "/api/v1/tags",
		Summary:     "List tags",
		Description: "Public. Returns every tag with the number of assessments carrying it. Tags are created automatically when an assessment uses a new name, so there is no dedicated create endpoint.",
	}, func(ctx context.Context, _ *struct{}) (*TagListOutput, error) {
		tags, err := database.ListTags(ctx, h.db)
		if err != nil {
			log.Printf("list tags: %v", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not list tags")
		}
		resp := &TagListOutput{}
		resp.Body.Tags = tags
		return resp, nil
	})
}
