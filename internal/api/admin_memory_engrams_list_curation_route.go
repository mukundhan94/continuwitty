package api

import (
	"context"
	"net/http"

	"engram/internal/admin"
)

func listMemoryAdminEngramCurationRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	return adminActorListRoute(
		requireAdminActor,
		parseMemoryAdminCurationSuggestionListRequest,
		func(ctx context.Context, request admin.MemoryCurationSuggestionListRequest) (any, error) {
			return service.ListMemoryCurationSuggestions(ctx, request)
		},
	)
}
