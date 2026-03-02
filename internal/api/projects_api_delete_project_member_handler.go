package api

import "net/http"

func deleteProjectMemberHandler(service ProjectService) http.HandlerFunc {
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
		userID, ok := userIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid user_id"})
			return
		}
		err := service.RemoveProjectMember(
			request.Context(),
			ProjectMemberDeleteRouteRequest{
				ActorUserID: actorUserID,
				ActorRole:   actorRole,
				ProjectID:   projectID,
				UserID:      userID,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]bool{"removed": true})
	}
}
