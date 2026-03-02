package api

import (
	"net/http"
	"strings"

	"engram/internal/models"
)

func decodeProjectMemberUpdatePayload(request *http.Request) (models.ProjectMemberUpdateRequest, error) {
	payload := models.ProjectMemberUpdateRequest{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return models.ProjectMemberUpdateRequest{}, err
	}
	role, err := models.ParseProjectMemberRole(strings.TrimSpace(string(payload.Role)))
	if err != nil {
		return models.ProjectMemberUpdateRequest{}, errProjectMemberRoleInvalid
	}
	payload.Role = role
	return payload, nil
}
