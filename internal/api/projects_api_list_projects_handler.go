package api

import (
	"net/http"
)

func listProjectsHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		includeArchived, limit, offset, err := parseProjectListQuery(request)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid query parameters"})
			return
		}
		records, err := service.ListProjects(
			request.Context(),
			ProjectListRouteRequest{
				ActorUserID:     actorUserID,
				ActorRole:       actorRole,
				IncludeArchived: includeArchived,
				Limit:           limit,
				Offset:          offset,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, records)
	}
}
