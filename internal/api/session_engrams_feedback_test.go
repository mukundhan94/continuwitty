package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestMountSessionAuthRoutesSubmitEngramFeedbackUsesRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAnalyst)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000f01")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000f09")
	feedbackID := uuid.MustParse("00000000-0000-0000-0000-000000000f02")
	note := "helpful answer"
	relevanceScore := 5
	avgRelevanceFeedback := 4.0
	var capturedInput SessionEngramFeedbackInput

	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			submitEngramFeedback: func(
				_ context.Context,
				input SessionEngramFeedbackInput,
			) (*models.EngramFeedbackRecord, error) {
				capturedInput = input
				return &models.EngramFeedbackRecord{
					FeedbackID:           feedbackID,
					EngramID:             input.EngramID,
					SessionID:            input.SessionID,
					ActorUserID:          input.ActorUserID,
					FeedbackType:         input.FeedbackType,
					Note:                 note,
					RelevanceScore:       &relevanceScore,
					CreatedAt:            time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
					UsefulCount:          3,
					FeedbackCount:        4,
					AvgRelevanceFeedback: &avgRelevanceFeedback,
					ContradictionCount:   1,
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	response := executeSubmitEngramFeedbackRequest(
		t,
		submitEngramFeedbackRequestInput{
			handler:     handler,
			loginCookie: loginCookie,
			engramID:    engramID,
			bodyPayload: map[string]any{
				"feedback_type":   "useful",
				"session_id":      sessionID.String(),
				"note":            "  helpful answer  ",
				"relevance_score": 5,
			},
		},
	)

	requireEqual(t, http.StatusOK, response.Code)
	assertCapturedFeedbackForwarding(
		t,
		capturedInput,
		feedbackForwardingExpectation{
			engramID:    engramID,
			sessionID:   sessionID,
			actorUserID: actor.UserID,
		},
	)
	assertFeedbackResponsePayload(
		t,
		response.Body.Bytes(),
		feedbackResponseExpectation{
			feedbackID:  feedbackID,
			engramID:    engramID,
			sessionID:   sessionID,
			actorUserID: actor.UserID,
		},
	)
}

func TestMountSessionAuthRoutesSubmitEngramFeedbackValidation(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAnalyst)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000f11")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			submitEngramFeedback: func(
				context.Context,
				SessionEngramFeedbackInput,
			) (*models.EngramFeedbackRecord, error) {
				t.Fatalf("submitEngramFeedback should not be called on invalid payload")
				return nil, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	testCases := []map[string]any{
		{"feedback_type": "invalid"},
		{"feedback_type": "useful", "relevance_score": 6},
		{"feedback_type": "useful", "session_id": "bad"},
	}
	for _, bodyPayload := range testCases {
		response := executeSubmitEngramFeedbackRequest(
			t,
			submitEngramFeedbackRequestInput{
				handler:     handler,
				loginCookie: loginCookie,
				engramID:    engramID,
				bodyPayload: bodyPayload,
			},
		)
		requireEqual(t, http.StatusBadRequest, response.Code)
	}
}

func TestMountSessionAuthRoutesSubmitEngramFeedbackReturnsNotFound(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAnalyst)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000f21")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			submitEngramFeedback: func(
				context.Context,
				SessionEngramFeedbackInput,
			) (*models.EngramFeedbackRecord, error) {
				return nil, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	response := executeSubmitEngramFeedbackRequest(
		t,
		submitEngramFeedbackRequestInput{
			handler:     handler,
			loginCookie: loginCookie,
			engramID:    engramID,
			bodyPayload: map[string]any{"feedback_type": "useful"},
		},
	)

	requireEqual(t, http.StatusNotFound, response.Code)
}

func assertCapturedFeedbackForwarding(
	t *testing.T,
	capturedInput SessionEngramFeedbackInput,
	expected feedbackForwardingExpectation,
) {
	t.Helper()
	requireEqual(t, expected.engramID, capturedInput.EngramID)
	requireEqual(t, expected.sessionID, derefUUID(capturedInput.SessionID))
	requireEqual(t, expected.actorUserID, capturedInput.ActorUserID)
	requireEqual(t, models.EngramFeedbackTypeUseful, capturedInput.FeedbackType)
	if capturedInput.Note == nil {
		t.Fatalf("expected note to be forwarded")
	}
	requireEqual(t, "helpful answer", *capturedInput.Note)
	if capturedInput.RelevanceScore == nil {
		t.Fatalf("expected relevance score to be forwarded")
	}
	requireEqual(t, 5, *capturedInput.RelevanceScore)
}

func assertFeedbackResponsePayload(
	t *testing.T,
	responseBody []byte,
	expected feedbackResponseExpectation,
) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	requireEqual(t, expected.feedbackID.String(), payload["feedback_id"].(string))
	requireEqual(t, expected.engramID.String(), payload["engram_id"].(string))
	requireEqual(t, expected.sessionID.String(), payload["session_id"].(string))
	requireEqual(t, expected.actorUserID.String(), payload["actor_user_id"].(string))
	requireEqual(t, "useful", payload["feedback_type"].(string))
	requireEqual(t, 5.0, payload["relevance_score"].(float64))
}

func executeSubmitEngramFeedbackRequest(
	t *testing.T,
	input submitEngramFeedbackRequestInput,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(input.bodyPayload)
	requireNoError(t, err)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/engrams/"+input.engramID.String()+"/feedback",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(input.loginCookie)
	response := httptest.NewRecorder()
	input.handler.ServeHTTP(response, request)
	return response
}

type submitEngramFeedbackRequestInput struct {
	handler     http.Handler
	loginCookie *http.Cookie
	engramID    uuid.UUID
	bodyPayload map[string]any
}

type feedbackForwardingExpectation struct {
	engramID    uuid.UUID
	sessionID   uuid.UUID
	actorUserID uuid.UUID
}

type feedbackResponseExpectation struct {
	feedbackID  uuid.UUID
	engramID    uuid.UUID
	sessionID   uuid.UUID
	actorUserID uuid.UUID
}
