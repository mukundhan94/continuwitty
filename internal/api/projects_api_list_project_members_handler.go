package api

import "net/http"

func listProjectMembersHandler(service ProjectService) http.HandlerFunc {
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
		includeRevoked, limit, offset, err := parseProjectMemberListQuery(request)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid query parameters"})
			return
		}
		records, err := service.ListProjectMembers(
			request.Context(),
			ProjectMemberListRouteRequest{
				ActorUserID:    actorUserID,
				ActorRole:      actorRole,
				ProjectID:      projectID,
				IncludeRevoked: includeRevoked,
				Limit:          limit,
				Offset:         offset,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, records)
	}
}
