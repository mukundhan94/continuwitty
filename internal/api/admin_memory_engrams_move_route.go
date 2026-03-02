package api

import (
	"context"
	"net/http"

	"engram/internal/admin"

	"github.com/google/uuid"
)

func moveMemoryAdminEngramRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return engramActorPathPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, actor AdminActor, engramID uuid.UUID, payload admin.EngramMoveRequest) (any, error) {
			return service.MoveEngram(
				ctx,
				engramID,
				admin.WriteActor{UserID: actor.UserID, Role: actor.Role},
				payload,
			)
		},
	)
}
