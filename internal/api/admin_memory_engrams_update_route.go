package api

import (
	"context"
	"net/http"

	"engram/internal/admin"

	"github.com/google/uuid"
)

func updateMemoryAdminEngramRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return engramActorPathPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, actor AdminActor, engramID uuid.UUID, payload admin.EngramUpdateRequest) (any, error) {
			return service.UpdateEngram(ctx, engramID, actor.UserID, payload)
		},
	)
}
