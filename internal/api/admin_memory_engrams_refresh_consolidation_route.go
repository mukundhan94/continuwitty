package api

import (
	"context"
	"net/http"

	"engram/internal/admin"
)

func refreshMemoryAdminEngramConsolidationRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	type payload struct {
		ProjectID    *string `json:"project_id,omitempty"`
		MinGroupSize *int    `json:"min_group_size,omitempty"`
	}
	return adminActorPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, decoded payload) (any, error) {
			return service.RefreshEngramConsolidationSuggestions(
				ctx,
				admin.EngramConsolidationSuggestionRefreshRequest{
					ProjectID:    decoded.ProjectID,
					MinGroupSize: decoded.MinGroupSize,
				},
			)
		},
	)
}
