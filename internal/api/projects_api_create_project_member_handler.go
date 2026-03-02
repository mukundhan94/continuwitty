package api

import "net/http"

func createProjectMemberHandler(service ProjectService) http.HandlerFunc {
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
		payload, err := decodeProjectMemberCreatePayload(request)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		created, err := service.AddProjectMember(
			request.Context(),
			ProjectMemberCreateRouteRequest{
				ActorUserID: actorUserID,
				ActorRole:   actorRole,
				ProjectID:   projectID,
				Payload:     payload,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	}
}
