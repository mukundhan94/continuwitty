package api

import (
	"net/http"

	"engram/internal/admin"
)

func refreshMemoryAdminEngramLinkCurationRoute(
	service MemoryAdminService,
	requireAdminActor RequireAdminActor,
) http.HandlerFunc {
	type payload struct {
		IncludeArchived   *bool    `json:"include_archived,omitempty"`
		Limit             *int     `json:"limit,omitempty"`
		StaleAfterDays    *int     `json:"stale_after_days,omitempty"`
		LowValueThreshold *float64 `json:"low_value_threshold,omitempty"`
	}
	return adminActorRoute(requireAdminActor, func(writer http.ResponseWriter, request *http.Request, actor AdminActor) {
		engramID, ok := parsePathUUID(writer, request, "engram_id")
		if !ok {
			return
		}
		var decoded payload
		if !decodeJSONAllowEmpty(writer, request, &decoded) {
			return
		}
		refreshRequest := admin.EngramLinkCurationSuggestionRefreshRequest{
			SourceEngramID: engramID,
		}
		if decoded.IncludeArchived != nil {
			refreshRequest.IncludeArchived = *decoded.IncludeArchived
		}
		if decoded.Limit != nil {
			refreshRequest.Limit = *decoded.Limit
		}
		if decoded.StaleAfterDays != nil {
			refreshRequest.StaleAfterDays = *decoded.StaleAfterDays
		}
		if decoded.LowValueThreshold != nil {
			refreshRequest.LowValueThreshold = *decoded.LowValueThreshold
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.RefreshEngramLinkCurationSuggestions(
				request.Context(),
				actor.UserID,
				refreshRequest,
			)
		})
	})
}
