package api

import (
	"net/http"

	"engram/internal/admin"
	"engram/internal/models"
)

func actionMemoryAdminEngramConsolidationRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	type payload struct {
		Status string `json:"status"`
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
		status, parseErr := models.ParseConsolidationSuggestionStatus(decoded.Status)
		if parseErr != nil {
			writeInvalidParameter(writer, "status")
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.ActionEngramConsolidationSuggestion(
				request.Context(),
				suggestionID,
				actor.UserID,
				admin.EngramConsolidationSuggestionActionRequest{Status: status},
			)
		})
	})
}
