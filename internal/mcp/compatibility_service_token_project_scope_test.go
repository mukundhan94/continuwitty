package mcp

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceTokenProjectScopeRejectsResolvedProjectOutsideAllowlist(t *testing.T) {
	sessionID := uuid.MustParse("40010000-0000-0000-0000-000000000400")
	engramID := uuid.MustParse("40030000-0000-0000-0000-000000000400")
	collectionID := uuid.MustParse("40035000-0000-0000-0000-000000000400")
	moveEngramID := uuid.MustParse("40050000-0000-0000-0000-000000000400")
	ownerUserID := uuid.MustParse("40030000-0000-0000-0000-000000000401")
	collectionOwnerUserID := uuid.MustParse("40035000-0000-0000-0000-000000000401")
	moveOwnerUserID := uuid.MustParse("40050000-0000-0000-0000-000000000401")
	testCases := []struct {
		name              string
		service           Service
		request           StreamCallRequest
		expectedProjectID string
	}{
		{
			name:    "session scoped tool",
			service: newSessionTokenScopeService(sessionID, "project-session", nil),
			request: tokenScopedDirectRequest(tokenScopedRequest{
				actorID:           "40010000-0000-0000-0000-000000000401",
				toolName:          "chat.get_session",
				params:            map[string]any{"session_id": sessionID.String()},
				scope:             models.MCPTokenScopeRead,
				allowedProjectIDs: []string{"project-allowed"},
			}),
			expectedProjectID: "project-session",
		},
		{
			name:    "engram scoped tool",
			service: newEngramTokenScopeService(engramID, ownerUserID, "project-engram", nil),
			request: tokenScopedDirectRequest(tokenScopedRequest{
				actorID:           "40030000-0000-0000-0000-000000000402",
				toolName:          "engram.get",
				params:            map[string]any{"engram_id": engramID.String()},
				scope:             models.MCPTokenScopeRead,
				allowedProjectIDs: []string{"project-allowed"},
			}),
			expectedProjectID: "project-engram",
		},
		{
			name: "collection scoped tool",
			service: newCollectionTokenScopeService(
				collectionID,
				collectionOwnerUserID,
				"project-collection",
			),
			request: tokenScopedDirectRequest(tokenScopedRequest{
				actorID:           "40035000-0000-0000-0000-000000000402",
				toolName:          "engram.collection_update",
				params:            map[string]any{"collection_id": collectionID.String()},
				scope:             models.MCPTokenScopeWrite,
				allowedProjectIDs: []string{"project-allowed"},
			}),
			expectedProjectID: "project-collection",
		},
		{
			name:    "move project target scope",
			service: newEngramTokenScopeService(moveEngramID, moveOwnerUserID, "project-source", nil),
			request: tokenScopedDirectRequest(tokenScopedRequest{
				actorID:  "40050000-0000-0000-0000-000000000402",
				toolName: "engram.move_project",
				params: map[string]any{
					"engram_id":         moveEngramID.String(),
					"target_project_id": "project-target",
				},
				scope:             models.MCPTokenScopeWrite,
				allowedProjectIDs: []string{"project-source"},
			}),
			expectedProjectID: "project-target",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertTokenScopeDeniedProject(t, testCase.service, testCase.request, testCase.expectedProjectID)
		})
	}
}

func TestCompatibilityServiceTokenProjectScopeAllowsSessionScopedToolWhenProjectMatches(t *testing.T) {
	sessionID := uuid.MustParse("40020000-0000-0000-0000-000000000400")
	service := newSessionTokenScopeService(sessionID, "project-session", nil)
	request := tokenScopedDirectRequest(tokenScopedRequest{
		actorID:           "40020000-0000-0000-0000-000000000401",
		toolName:          "chat.get_session",
		params:            map[string]any{"session_id": sessionID.String()},
		scope:             models.MCPTokenScopeRead,
		allowedProjectIDs: []string{"project-session"},
	})

	frame := runCompatibilityRequestWithService(t, service, request)
	session := chatSessionFromFrame(t, frame, false)
	if session.SessionID != sessionID {
		t.Fatalf("expected session payload when token project matches")
	}
}

func TestCompatibilityServiceTokenProjectScopeChatSaveAsEngramPrefersSessionProjectOverProjectInput(t *testing.T) {
	sessionID := uuid.MustParse("40040000-0000-0000-0000-000000000400")
	saveService := &fakeSessionSaveAsEngramService{
		response: &models.SaveSessionAsEngramResponse{
			EngramID:  uuid.MustParse("40040000-0000-0000-0000-000000000401"),
			SessionID: sessionID,
			CreatedAt: time.Now().UTC(),
		},
	}
	service := newSessionTokenScopeService(sessionID, "project-session", saveService)
	request := tokenScopedDirectRequest(tokenScopedRequest{
		actorID:  "40040000-0000-0000-0000-000000000402",
		toolName: "chat.save_as_engram",
		params: map[string]any{
			"session_id": sessionID.String(),
			"project_id": "project-other",
		},
		scope:             models.MCPTokenScopeWrite,
		allowedProjectIDs: []string{"project-session"},
	})

	frame := runCompatibilityRequestWithService(t, service, request)
	if errorPayload, isError := frame["error"].(map[string]any); isError {
		t.Fatalf("unexpected error frame: %#v", errorPayload)
	}
	saved := savedEngramFromFrame(t, frame, false)
	if saved.SessionID != sessionID {
		t.Fatalf("expected save-as-engram success using session-scoped project resolution")
	}
}

func TestCompatibilityServiceTokenProjectScopeEngramMoveProjectRejectsSourceProjectOutsideAllowlist(t *testing.T) {
	engramID := uuid.MustParse("40060000-0000-0000-0000-000000000400")
	ownerUserID := uuid.MustParse("40060000-0000-0000-0000-000000000401")
	moveService := &fakeEngramMoveService{moved: &models.AdminEngramRecord{EngramID: engramID}}
	service := newEngramTokenScopeService(engramID, ownerUserID, "project-source", moveService)
	request := tokenScopedDirectRequest(tokenScopedRequest{
		actorID:  "40060000-0000-0000-0000-000000000402",
		toolName: "engram.move_project",
		params: map[string]any{
			"engram_id":         engramID.String(),
			"target_project_id": "project-target",
		},
		scope:             models.MCPTokenScopeWrite,
		allowedProjectIDs: []string{"project-target"},
	})

	assertTokenScopeDeniedProject(t, service, request, "project-source")
	if moveService.call.EngramID != uuid.Nil {
		t.Fatalf("expected move service not to be called when source project is disallowed")
	}
}

func newSessionTokenScopeService(
	sessionID uuid.UUID,
	projectID string,
	saveService SessionSaveAsEngramService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			SessionGet: &fakeSessionGetService{
				session: &models.ChatSessionRecord{
					SessionID: sessionID,
					ProjectID: projectID,
				},
			},
			SessionSaveAsEngram: saveService,
		},
	)
}

func newEngramTokenScopeService(
	engramID uuid.UUID,
	ownerUserID uuid.UUID,
	projectID string,
	moveService EngramMoveService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			EngramGet: &fakeEngramGetService{
				engram: &models.AdminEngramRecord{
					EngramID:    engramID,
					ProjectID:   projectID,
					OwnerUserID: &ownerUserID,
				},
			},
			EngramMove: moveService,
		},
	)
}

func newCollectionTokenScopeService(
	collectionID uuid.UUID,
	ownerUserID uuid.UUID,
	projectID string,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			EngramCollectionGet: &fakeEngramCollectionGetService{
				collection: &models.EngramCollectionRecord{
					CollectionID: collectionID,
					ProjectID:    projectID,
					OwnerUserID:  ownerUserID,
				},
			},
		},
	)
}

type tokenScopedRequest struct {
	actorID           string
	toolName          string
	params            map[string]any
	scope             models.MCPTokenScope
	allowedProjectIDs []string
}

func tokenScopedDirectRequest(input tokenScopedRequest) StreamCallRequest {
	request := directToolRequest(input.actorID, input.toolName, input.params)
	request.TokenAuth = &models.MCPTokenAuthContext{
		Scope:             input.scope,
		AllowedProjectIDs: input.allowedProjectIDs,
	}
	return request
}

func assertTokenScopeDeniedProject(
	t *testing.T,
	service Service,
	request StreamCallRequest,
	expectedProjectID string,
) {
	t.Helper()
	frame := runCompatibilityRequestWithService(t, service, request)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32003)
	data := mapFromMap(t, errorPayload, "data")
	if data["project_id"] != expectedProjectID {
		t.Fatalf("expected denied project_id %q in error payload", expectedProjectID)
	}
}

type fakeEngramCollectionGetService struct {
	collection *models.EngramCollectionRecord
	err        error
}

func (service *fakeEngramCollectionGetService) GetCollection(
	_ context.Context,
	_ EngramCollectionGetRequest,
) (*models.EngramCollectionRecord, error) {
	if service.err != nil {
		return nil, service.err
	}
	if service.collection == nil {
		return nil, nil
	}
	collection := *service.collection
	return &collection, nil
}
