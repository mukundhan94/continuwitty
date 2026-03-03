package api

import (
	"context"
	"net/http"

	"engram/internal/admin"
)

func refreshMemoryAdminEngramFreshnessRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	type payload struct {
		ProjectID    *string  `json:"project_id,omitempty"`
		HalfLifeDays *float64 `json:"half_life_days,omitempty"`
	}
	return adminActorPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, decoded payload) (any, error) {
			return service.RefreshEngramFreshness(
				ctx,
				admin.EngramFreshnessRefreshRequest{
					ProjectID:    decoded.ProjectID,
					HalfLifeDays: decoded.HalfLifeDays,
				},
			)
		},
	)
}
