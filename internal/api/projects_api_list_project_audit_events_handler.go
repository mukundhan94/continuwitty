package api

import "net/http"

func listProjectAuditEventsHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		projectID, ok := projectIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid project_id"})
			return
		}
		limit, offset, err := parseProjectAuditListQuery(request)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid query parameters"})
			return
		}
		events, err := service.ListProjectAuditEvents(
			request.Context(),
			ProjectAuditListRouteRequest{
				ActorUserID: actorUserID,
				ActorRole:   actorRole,
				ProjectID:   projectID,
				Limit:       limit,
				Offset:      offset,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, events)
	}
}
