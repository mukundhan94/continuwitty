package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramFeedbackParity(t *testing.T) {
	actorUserID := uuid.MustParse("39910000-0000-0000-0000-000000000399")
	engramID := uuid.MustParse("39910000-0000-0000-0000-000000000400")
	sessionID := uuid.MustParse("39910000-0000-0000-0000-000000000498")
	feedbackID := uuid.MustParse("39910000-0000-0000-0000-000000000401")
	record := models.EngramFeedbackRecord{
		FeedbackID:           feedbackID,
		EngramID:             engramID,
		SessionID:            &sessionID,
		ActorUserID:          actorUserID,
		FeedbackType:         models.EngramFeedbackTypeUseful,
		Note:                 "helpful",
		RelevanceScore:       intPtr(5),
		CreatedAt:            time.Date(2026, 3, 2, 13, 0, 0, 0, time.UTC),
		UsefulCount:          8,
		FeedbackCount:        9,
		AvgRelevanceFeedback: float64Ptr(4.4),
		ContradictionCount:   1,
	}
	service := &fakeEngramFeedbackService{record: &record}
	params := map[string]any{
		"engram_id":       engramID.String(),
		"session_id":      sessionID.String(),
		"feedback_type":   "useful",
		"note":            "helpful",
		"relevance_score": 5,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.feedback", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_feedback", params),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramFeedbackCompatibilityService(service),
				testCase.request,
			)
			got := engramFeedbackFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(record, got) {
				t.Fatalf("expected feedback payload to match service output")
			}
			assertEngramFeedbackCall(
				t,
				service.call,
				EngramFeedbackRequest{
					ActorUserID:    actorUserID,
					EngramID:       engramID,
					SessionID:      &sessionID,
					FeedbackType:   models.EngramFeedbackTypeUseful,
					Note:           stringPtr("helpful"),
					RelevanceScore: intPtr(5),
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramFeedbackNormalizesOptionalNote(t *testing.T) {
	actorUserID := uuid.MustParse("39920000-0000-0000-0000-000000000399")
	engramID := uuid.MustParse("39920000-0000-0000-0000-000000000400")
	testCases := []struct {
		name         string
		feedbackType models.EngramFeedbackType
		params       map[string]any
	}{
		{
			name:         "omitted note",
			feedbackType: models.EngramFeedbackTypeContradiction,
			params: map[string]any{
				"engram_id":     engramID.String(),
				"feedback_type": "contradiction",
			},
		},
		{
			name:         "blank note",
			feedbackType: models.EngramFeedbackTypeUseful,
			params: map[string]any{
				"engram_id":     engramID.String(),
				"feedback_type": "useful",
				"note":          "   ",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			service := &fakeEngramFeedbackService{
				record: &models.EngramFeedbackRecord{
					FeedbackID:         uuid.MustParse("39920000-0000-0000-0000-000000000401"),
					EngramID:           engramID,
					ActorUserID:        actorUserID,
					FeedbackType:       testCase.feedbackType,
					CreatedAt:          time.Date(2026, 3, 2, 13, 10, 0, 0, time.UTC),
					UsefulCount:        0,
					FeedbackCount:      2,
					ContradictionCount: 2,
				},
			}
			frame := runCompatibilityRequestWithService(
				t,
				newEngramFeedbackCompatibilityService(service),
				directToolRequest(
					actorUserID.String(),
					"engram.feedback",
					testCase.params,
				),
			)
			_ = engramFeedbackFromFrame(t, frame, false)
			if service.call.Note != nil {
				t.Fatalf("expected normalized note to be nil")
			}
		})
	}
}

func TestCompatibilityServiceEngramFeedbackValidationAndErrors(t *testing.T) {
	service := newEngramFeedbackCompatibilityService(&fakeEngramFeedbackService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing engram id", params: map[string]any{}},
		{name: "invalid engram id", params: map[string]any{"engram_id": "bad"}},
		{name: "missing feedback type", params: map[string]any{"engram_id": uuid.NewString()}},
		{name: "invalid feedback type", params: map[string]any{"engram_id": uuid.NewString(), "feedback_type": "bad"}},
		{name: "invalid session id", params: map[string]any{"engram_id": uuid.NewString(), "session_id": "bad", "feedback_type": "useful"}},
		{name: "invalid note", params: map[string]any{"engram_id": uuid.NewString(), "feedback_type": "useful", "note": 123}},
		{name: "invalid relevance score", params: map[string]any{"engram_id": uuid.NewString(), "feedback_type": "useful", "relevance_score": 7}},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39930000-0000-0000-0000-000000000399",
					"engram_feedback",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundID := uuid.MustParse("39930000-0000-0000-0000-000000000400")
	notFoundFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			"39930000-0000-0000-0000-000000000401",
			"engram_feedback",
			map[string]any{
				"engram_id":     notFoundID.String(),
				"feedback_type": "useful",
			},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32004)
	assertEngramMutationNotFoundData(t, notFoundPayload, notFoundID)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramFeedbackCompatibilityService(
			&fakeEngramFeedbackService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39930000-0000-0000-0000-000000000402",
			"engram_feedback",
			map[string]any{
				"engram_id":     notFoundID.String(),
				"feedback_type": "useful",
			},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramFeedbackCompatibilityService(service EngramFeedbackService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramFeedback: service},
	)
}

func engramFeedbackFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.EngramFeedbackRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	feedback, ok := payload["feedback"].(models.EngramFeedbackRecord)
	if !ok {
		t.Fatalf("expected engram feedback payload")
	}
	return feedback
}

func assertEngramFeedbackCall(
	t *testing.T,
	actual EngramFeedbackRequest,
	expected EngramFeedbackRequest,
) {
	t.Helper()
	if actual.ActorUserID != expected.ActorUserID {
		t.Fatalf("expected actor user id to be forwarded")
	}
	if actual.EngramID != expected.EngramID {
		t.Fatalf("expected engram id to be forwarded")
	}
	if actual.FeedbackType != expected.FeedbackType {
		t.Fatalf("expected feedback type to be forwarded")
	}
	assertOptionalDeleteReason(t, actual.Note, expected.Note)
	assertOptionalPointerValue(t, actual.RelevanceScore, expected.RelevanceScore)
	assertOptionalPointerValue(t, actual.SessionID, expected.SessionID)
}

func float64Ptr(value float64) *float64 {
	return &value
}

func assertOptionalPointerValue[T comparable](
	t *testing.T,
	actual *T,
	expected *T,
) {
	t.Helper()
	if actual == nil && expected == nil {
		return
	}
	if actual == nil || expected == nil {
		t.Fatalf("expected optional pointers to match")
	}
	if *actual != *expected {
		t.Fatalf("expected optional pointers to match")
	}
}

type fakeEngramFeedbackService struct {
	record *models.EngramFeedbackRecord
	err    error
	call   EngramFeedbackRequest
}

func (service *fakeEngramFeedbackService) SubmitEngramFeedback(
	_ context.Context,
	request EngramFeedbackRequest,
) (*models.EngramFeedbackRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.record == nil {
		return nil, nil
	}
	record := *service.record
	return &record, nil
}
