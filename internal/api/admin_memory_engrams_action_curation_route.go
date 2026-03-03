package api

import (
	"net/http"

	"engram/internal/admin"
	"engram/internal/models"
)

func actionMemoryAdminEngramCurationRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	type payload struct {
		ProjectID *string `json:"project_id,omitempty"`
		Status    string  `json:"status"`
	}
	return adminActorRoute(requireAdminActor, func(writer http.ResponseWriter, request *http.Request, actor AdminActor) {
		suggestionID, ok := parsePathUUID(writer, request, "suggestion_id")
		if !ok {
			return
		}
		var decoded payload
		if !decodeJSONAllowEmpty(writer, request, &decoded) {
			return
		}
		status, parseErr := models.ParseMemoryCurationSuggestionStatus(decoded.Status)
		if parseErr != nil || status == models.MemoryCurationSuggestionStatusSuggested {
			writeInvalidParameter(writer, "status")
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.ActionMemoryCurationSuggestion(
				request.Context(),
				suggestionID,
				actor.UserID,
				admin.MemoryCurationSuggestionActionRequest{
					ProjectID: decoded.ProjectID,
					Status:    status,
				},
			)
		})
	})
}
