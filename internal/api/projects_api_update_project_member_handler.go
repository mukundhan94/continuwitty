package api

import "net/http"

func updateProjectMemberHandler(service ProjectService) http.HandlerFunc {
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
		payload, err := decodeProjectMemberUpdatePayload(request)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		updated, err := service.UpdateProjectMember(
			request.Context(),
			ProjectMemberUpdateRouteRequest{
				ActorUserID: actorUserID,
				ActorRole:   actorRole,
				ProjectID:   projectID,
				UserID:      userID,
				Payload:     payload,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, updated)
	}
}
