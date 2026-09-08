package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// HealthOutput reports service liveness.
type HealthOutput struct {
	Body struct {
		Status string `json:"status" doc:"'ok' when the service and database are healthy"`
	}
}

func (h *Handlers) registerHealth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "getHealth",
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		Summary:     "Check service health",
	}, func(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
		if err := h.db.PingContext(ctx); err != nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "database unavailable")
		}
		resp := &HealthOutput{}
		resp.Body.Status = "ok"
		return resp, nil
	})
}
