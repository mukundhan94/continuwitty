package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCreateFromConversationParity(t *testing.T) {
	actorUserID := uuid.MustParse("39800000-0000-0000-0000-000000000398")
	sourceSessionID := uuid.MustParse("39800000-0000-0000-0000-000000000399")
	params := createConversationParams(sourceSessionID)
	response := &EngramCreateFromConversationResponse{
		Engram: models.EngramCreateResponse{
			EngramID:           uuid.MustParse("39800000-0000-0000-0000-000000000400"),
			CreatedAt:          time.Unix(1700003980, 0).UTC(),
			ResolvedProjectID:  stringPtr("proj-alpha"),
			UsedDefaultProject: false,
		},
		EnrichmentReport: map[string]any{
			"enrichment_applied":   true,
			"auto_tags":            []string{"ops"},
			"auto_keywords":        []string{"risk"},
			"abstract_derived":     true,
			"resolved_project_id":  "proj-alpha",
			"used_default_project": false,
		},
	}
	service := &fakeEngramCreateConversationService{response: response}

	assertCreateConversationParityCase(
		t,
		service,
		response,
		"direct",
		directToolRequest(actorUserID.String(), "engram.create_from_conversation", params),
		false,
		expectedCreateConversationCall(actorUserID, models.UserRoleViewer, sourceSessionID),
	)
	assertCreateConversationParityCase(
		t,
		service,
		response,
		"tools call",
		toolsCallRequest(actorUserID.String(), "engram_create_from_conversation", params),
		true,
		expectedCreateConversationCall(actorUserID, models.UserRoleAnalyst, sourceSessionID),
	)
}

func TestCompatibilityServiceEngramCreateFromConversationUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39810000-0000-0000-0000-000000000398")
	service := &fakeEngramCreateConversationService{
		response: &EngramCreateFromConversationResponse{
			Engram: models.EngramCreateResponse{
				EngramID:           uuid.MustParse("39810000-0000-0000-0000-000000000399"),
				CreatedAt:          time.Unix(1700003981, 0).UTC(),
				ResolvedProjectID:  stringPtr("proj-default"),
				UsedDefaultProject: true,
			},
			EnrichmentReport: map[string]any{},
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramCreateConversationCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.create_from_conversation",
			map[string]any{
				"title":                 "Recap",
				"conversation_markdown": "Details",
			},
		),
	)
	_, _ = engramCreateConversationFromFrame(t, frame, false)
	if service.call.VisibilityScope != string(models.VisibilityScopePrivate) {
		t.Fatalf("expected default visibility scope to be private")
	}
	if len(service.call.Tags) != 0 || len(service.call.Keywords) != 0 {
		t.Fatalf("expected default tags/keywords to be empty")
	}
}

func TestCompatibilityServiceEngramCreateFromConversationValidationAndErrors(t *testing.T) {
	service := newEngramCreateConversationCompatibilityService(
		&fakeEngramCreateConversationService{},
	)
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "invalid payload", params: map[string]any{"title": "x", "conversation_markdown": "y", "tags": "bad"}},
		{name: "missing title", params: map[string]any{"conversation_markdown": "y"}},
		{name: "missing conversation markdown", params: map[string]any{"title": "x"}},
		{name: "invalid visibility", params: map[string]any{"title": "x", "conversation_markdown": "y", "visibility_scope": "bad"}},
		{name: "invalid source session", params: map[string]any{"title": "x", "conversation_markdown": "y", "source_session_id": "bad"}},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39820000-0000-0000-0000-000000000398",
					"engram_create_from_conversation",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newEngramCreateConversationCompatibilityService(
			&fakeEngramCreateConversationService{err: projects.ErrProjectNotFound},
		),
		toolsCallRequest(
			"39820000-0000-0000-0000-000000000399",
			"engram_create_from_conversation",
			map[string]any{"title": "x", "conversation_markdown": "y", "project_id": "missing"},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertErrorStatusCode(t, notFoundPayload, 404)
	assertErrorDetail(t, notFoundPayload, "Project not found")

	defaultMissingFrame := runCompatibilityRequestWithService(
		t,
		newEngramCreateConversationCompatibilityService(
			&fakeEngramCreateConversationService{err: projects.ErrProjectIDRequiredWhenNoDefaultProject},
		),
		toolsCallRequest(
			"39820000-0000-0000-0000-000000000400",
			"engram_create_from_conversation",
			map[string]any{"title": "x", "conversation_markdown": "y"},
		),
	)
	defaultMissingPayload := errorPayloadFromFrame(t, defaultMissingFrame)
	requireErrorCode(t, defaultMissingPayload, -32602)
	assertErrorStatusCode(t, defaultMissingPayload, 422)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramCreateConversationCompatibilityService(
			&fakeEngramCreateConversationService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39820000-0000-0000-0000-000000000401",
			"engram_create_from_conversation",
			map[string]any{"title": "x", "conversation_markdown": "y"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramCreateConversationCompatibilityService(
	service EngramCreateFromConversationService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCreateConversation: service},
	)
}

func createConversationParams(sourceSessionID uuid.UUID) map[string]any {
	return map[string]any{
		"project_id":            " proj-alpha ",
		"title":                 " Session recap ",
		"abstract":              " quick summary ",
		"conversation_markdown": " Details ",
		"thread_id":             "thread-1",
		"visibility_scope":      "project",
		"tags":                  []any{"alpha"},
		"keywords":              []any{"beta"},
		"retrieval_text":        "retrieval text",
		"source_session_id":     sourceSessionID.String(),
	}
}

func expectedCreateConversationCall(
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	sourceSessionID uuid.UUID,
) engramCreateConversationExpectation {
	return engramCreateConversationExpectation{
		actorUserID: actorUserID,
		actorRole:   actorRole,
		request: EngramCreateFromConversationRequest{
			ActorUserID:          actorUserID,
			ActorRole:            actorRole,
			ProjectID:            "proj-alpha",
			ThreadID:             stringPtr("thread-1"),
			Title:                "Session recap",
			Abstract:             "quick summary",
			ConversationMarkdown: "Details",
			Tags:                 []string{"alpha"},
			Keywords:             []string{"beta"},
			VisibilityScope:      string(models.VisibilityScopeProject),
			RetrievalText:        stringPtr("retrieval text"),
			SourceSessionID:      &sourceSessionID,
			EnrichmentOrigin:     "mcp.engram.create_from_conversation",
		},
	}
}

func assertCreateConversationParityCase(
	t *testing.T,
	service *fakeEngramCreateConversationService,
	response *EngramCreateFromConversationResponse,
	name string,
	request StreamCallRequest,
	asToolsCallPath bool,
	expectedCall engramCreateConversationExpectation,
) {
	t.Run(name, func(t *testing.T) {
		frame := runCompatibilityRequestWithService(
			t,
			newEngramCreateConversationCompatibilityService(service),
			request,
		)
		createdEngram, report := engramCreateConversationFromFrame(t, frame, asToolsCallPath)
		if !reflect.DeepEqual(response.Engram, createdEngram) {
			t.Fatalf("expected create response engram payload")
		}
		if !reflect.DeepEqual(response.EnrichmentReport, report) {
			t.Fatalf("expected create response enrichment report")
		}
		assertEngramCreateConversationCall(t, service.call, expectedCall)
	})
}

func engramCreateConversationFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) (models.EngramCreateResponse, map[string]any) {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	engram, ok := payload["engram"].(models.EngramCreateResponse)
	if !ok {
		t.Fatalf("expected engram payload")
	}
	report := mapFromMap(t, payload, "enrichment_report")
	return engram, report
}

type engramCreateConversationExpectation struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	request     EngramCreateFromConversationRequest
}

func assertEngramCreateConversationCall(
	t *testing.T,
	call EngramCreateFromConversationRequest,
	expected engramCreateConversationExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if !reflect.DeepEqual(call, expected.request) {
		t.Fatalf("expected create-from-conversation request forwarded")
	}
}

type fakeEngramCreateConversationService struct {
	response *EngramCreateFromConversationResponse
	err      error
	call     EngramCreateFromConversationRequest
}

func (service *fakeEngramCreateConversationService) CreateEngramFromConversation(
	_ context.Context,
	request EngramCreateFromConversationRequest,
) (*EngramCreateFromConversationResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	response := *service.response
	return &response, nil
}
