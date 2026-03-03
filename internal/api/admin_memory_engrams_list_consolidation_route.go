package api

import (
	"context"
	"net/http"

	"engram/internal/admin"
)

func listMemoryAdminEngramConsolidationRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	return adminActorListRoute(
		requireAdminActor,
		parseMemoryAdminConsolidationSuggestionListRequest,
		func(ctx context.Context, request admin.EngramConsolidationSuggestionListRequest) (any, error) {
			return service.ListEngramConsolidationSuggestions(ctx, request)
		},
	)
}
