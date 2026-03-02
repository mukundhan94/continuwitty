package api

import (
	"context"
	"net/http"

	"engram/internal/admin"

	"github.com/google/uuid"
)

func deleteMemoryAdminEngramRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return engramActorPathPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, actor AdminActor, engramID uuid.UUID, payload admin.EngramDeleteRequest) (any, error) {
			return service.DeleteEngram(ctx, engramID, actor.UserID, payload)
		},
	)
}
