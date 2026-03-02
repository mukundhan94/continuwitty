package api

import (
	"net/http"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
)

func decodeProjectMemberCreatePayload(request *http.Request) (models.ProjectMemberCreateRequest, error) {
	payload := models.ProjectMemberCreateRequest{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return models.ProjectMemberCreateRequest{}, err
	}
	if payload.UserID == uuid.Nil {
		return models.ProjectMemberCreateRequest{}, errProjectMemberUserInvalid
	}
	role, err := models.ParseProjectMemberRole(strings.TrimSpace(string(payload.Role)))
	if err != nil {
		return models.ProjectMemberCreateRequest{}, errProjectMemberRoleInvalid
	}
	payload.Role = role
	return payload, nil
}
