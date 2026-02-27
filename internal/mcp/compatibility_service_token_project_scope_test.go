package mcp

import (
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceTokenProjectScopeRejectsSessionScopedProjectOutsideAllowlist(t *testing.T) {
	sessionID := uuid.MustParse("40010000-0000-0000-0000-000000000400")
	service := NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			SessionGet: &fakeSessionGetService{
				session: &models.ChatSessionRecord{
					SessionID: sessionID,
					ProjectID: "project-session",
				},
			},
		},
	)
	request := directToolRequest(
		"40010000-0000-0000-0000-000000000401",
		"chat.get_session",
		map[string]any{"session_id": sessionID.String()},
	)
	request.TokenAuth = &models.MCPTokenAuthContext{
		Scope:             models.MCPTokenScopeRead,
		AllowedProjectIDs: []string{"project-allowed"},
	}

	frame := runCompatibilityRequestWithService(t, service, request)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32003)
	data := mapFromMap(t, errorPayload, "data")
	if data["project_id"] != "project-session" {
		t.Fatalf("expected denied session project_id in error payload")
	}
}

func TestCompatibilityServiceTokenProjectScopeAllowsSessionScopedToolWhenProjectMatches(t *testing.T) {
	sessionID := uuid.MustParse("40020000-0000-0000-0000-000000000400")
	service := NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			SessionGet: &fakeSessionGetService{
				session: &models.ChatSessionRecord{
					SessionID: sessionID,
					ProjectID: "project-session",
				},
			},
		},
	)
	request := directToolRequest(
		"40020000-0000-0000-0000-000000000401",
		"chat.get_session",
		map[string]any{"session_id": sessionID.String()},
	)
	request.TokenAuth = &models.MCPTokenAuthContext{
		Scope:             models.MCPTokenScopeRead,
		AllowedProjectIDs: []string{"project-session"},
	}

	frame := runCompatibilityRequestWithService(t, service, request)
	session := chatSessionFromFrame(t, frame, false)
	if session.SessionID != sessionID {
		t.Fatalf("expected session payload when token project matches")
	}
}

func TestCompatibilityServiceTokenProjectScopeRejectsEngramScopedProjectOutsideAllowlist(t *testing.T) {
	engramID := uuid.MustParse("40030000-0000-0000-0000-000000000400")
	ownerUserID := uuid.MustParse("40030000-0000-0000-0000-000000000401")
	service := NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			EngramGet: &fakeEngramGetService{
				engram: &models.AdminEngramRecord{
					EngramID:    engramID,
					ProjectID:   "project-engram",
					OwnerUserID: &ownerUserID,
				},
			},
		},
	)
	request := directToolRequest(
		"40030000-0000-0000-0000-000000000402",
		"engram.get",
		map[string]any{"engram_id": engramID.String()},
	)
	request.TokenAuth = &models.MCPTokenAuthContext{
		Scope:             models.MCPTokenScopeRead,
		AllowedProjectIDs: []string{"project-allowed"},
	}

	frame := runCompatibilityRequestWithService(t, service, request)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32003)
	data := mapFromMap(t, errorPayload, "data")
	if data["project_id"] != "project-engram" {
		t.Fatalf("expected denied engram project_id in error payload")
	}
}
