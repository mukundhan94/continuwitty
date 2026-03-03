package api

import (
	"context"
	"net/http"

	"engram/internal/admin"
	"engram/internal/models"
)

func listMemoryAdminEngramContradictionRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	return adminActorListRoute(
		requireAdminActor,
		parseMemoryAdminContradictionAlertListRequest,
		func(ctx context.Context, request admin.EngramContradictionAlertListRequest) (any, error) {
			return service.ListEngramContradictionAlerts(ctx, request)
		},
	)
}

func parseMemoryAdminContradictionAlertListRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (admin.EngramContradictionAlertListRequest, bool) {
	projectID, status, limit, offset, ok := parseMemoryAdminProjectStatusListRequest(
		writer,
		request,
		func(w http.ResponseWriter, r *http.Request, key string) (*models.ContradictionAlertStatus, bool) {
			return parseOptionalStatusQuery(w, r, key, models.ParseContradictionAlertStatus)
		},
	)
	if !ok {
		return admin.EngramContradictionAlertListRequest{}, false
	}
	return admin.EngramContradictionAlertListRequest{
		ProjectID: projectID,
		Status:    status,
		Limit:     limit,
		Offset:    offset,
	}, true
}
