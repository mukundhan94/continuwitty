package api

import (
	"context"
	"net/http"

	"engram/internal/admin"
)

func refreshMemoryAdminEngramContradictionRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	type payload struct {
		ProjectID *string `json:"project_id,omitempty"`
	}
	return adminActorPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, decoded payload) (any, error) {
			return service.RefreshEngramContradictionAlerts(
				ctx,
				admin.EngramContradictionAlertRefreshRequest{
					ProjectID: decoded.ProjectID,
				},
			)
		},
	)
}
