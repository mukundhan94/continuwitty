package mcp

import (
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatListProjectDocumentsParity(t *testing.T) {
	actorUserID := uuid.MustParse("37000000-0000-0000-0000-000000000370")
	service := newFakeProjectDocumentListService(actorUserID)

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: directToolRequest(
				actorUserID.String(),
				"chat.list_project_documents",
				map[string]any{"project_id": "proj-alpha", "limit": 25.0, "offset": 2.0},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_list_project_documents",
				map[string]any{"project_id": "proj-alpha", "limit": 25.0, "offset": 2.0},
			),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newChatProjectDocumentsCompatibilityService(service),
				testCase.request,
			)
			documents := projectDocumentsFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(service.documents, documents) {
				t.Fatalf("expected documents payload to match service output")
			}
			assertProjectDocumentListCall(
				t,
				service.call,
				projectDocumentListExpectation{
					actorUserID: actorUserID,
					projectID:   stringPtr("proj-alpha"),
					limit:       25,
					offset:      2,
				},
			)
		})
	}
}

func TestCompatibilityServiceChatListProjectDocumentsUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("37100000-0000-0000-0000-000000000371")
	service := &fakeProjectDocumentListService{documents: []models.DocumentRecord{}}

	frame := runCompatibilityRequestWithService(
		t,
		newChatProjectDocumentsCompatibilityService(service),
		directToolRequest(actorUserID.String(), "chat.list_project_documents", map[string]any{}),
	)
	_ = projectDocumentsFromFrame(t, frame, false)
	assertProjectDocumentListCall(
		t,
		service.call,
		projectDocumentListExpectation{
			actorUserID: actorUserID,
			projectID:   nil,
			limit:       defaultChatProjectDocumentsLimit,
			offset:      defaultChatProjectDocumentsOffset,
		},
	)
}

func TestCompatibilityServiceChatListProjectDocumentsRejectsInvalidPaging(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newChatProjectDocumentsCompatibilityService(&fakeProjectDocumentListService{}),
		toolsCallRequest(
			"37200000-0000-0000-0000-000000000372",
			"chat_list_project_documents",
			map[string]any{"limit": "bad"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
}

func TestCompatibilityServiceChatListProjectDocumentsReturnsInternalErrorOnServiceFailure(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newChatProjectDocumentsCompatibilityService(&fakeProjectDocumentListService{err: errors.New("boom")}),
		toolsCallRequest(
			"37300000-0000-0000-0000-000000000373",
			"chat_list_project_documents",
			map[string]any{"limit": 10.0},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, frame), -32603)
}
