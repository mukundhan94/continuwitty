package api

import (
	"net/http"

	"engram/internal/admin"
	"engram/internal/models"
)

func resolveMemoryAdminEngramContradictionRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	type payload struct {
		ProjectID *string `json:"project_id,omitempty"`
		Status    string  `json:"status"`
	}
	return adminActorRoute(requireAdminActor, func(writer http.ResponseWriter, request *http.Request, actor AdminActor) {
		alertID, ok := parsePathUUID(writer, request, "alert_id")
		if !ok {
			return
		}
		var decoded payload
		if !decodeJSONAllowEmpty(writer, request, &decoded) {
			return
		}
		status, parseErr := models.ParseContradictionAlertStatus(decoded.Status)
		if parseErr != nil {
			writeInvalidParameter(writer, "status")
			return
		}
		if status == models.ContradictionAlertStatusOpen {
			writeInvalidParameter(writer, "status")
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.ResolveEngramContradictionAlert(
				request.Context(),
				alertID,
				actor.UserID,
				admin.EngramContradictionAlertResolveRequest{
					ProjectID: decoded.ProjectID,
					Status:    status,
				},
			)
		})
	})
}
