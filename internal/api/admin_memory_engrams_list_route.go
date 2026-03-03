package api

import (
	"context"
	"net/http"

	"engram/internal/admin"
)

func listMemoryAdminEngramsRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return adminActorListRoute(
		requireAdminActor,
		parseMemoryAdminEngramListRequest,
		func(ctx context.Context, request admin.MemoryAdminEngramListRequest) (any, error) {
			return service.ListEngrams(ctx, request)
		},
	)
}
