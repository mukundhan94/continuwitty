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
	feedbackID := uuid.MustParse("00000000-0000-0000-0000-000000000f02")
	note := "helpful answer"
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
					FeedbackID:         feedbackID,
					EngramID:           input.EngramID,
					ActorUserID:        input.ActorUserID,
					FeedbackType:       input.FeedbackType,
					Note:               note,
					CreatedAt:          time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
					UsefulCount:        3,
					ContradictionCount: 1,
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

	body, _ := json.Marshal(map[string]any{
		"feedback_type": "useful",
		"note":          "  helpful answer  ",
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/engrams/"+engramID.String()+"/feedback",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, engramID, capturedInput.EngramID)
	requireEqual(t, actor.UserID, capturedInput.ActorUserID)
	requireEqual(t, models.EngramFeedbackTypeUseful, capturedInput.FeedbackType)
	if capturedInput.Note == nil {
		t.Fatalf("expected note to be forwarded")
	}
	requireEqual(t, "  helpful answer  ", *capturedInput.Note)

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	requireEqual(t, feedbackID.String(), payload["feedback_id"].(string))
	requireEqual(t, engramID.String(), payload["engram_id"].(string))
	requireEqual(t, actor.UserID.String(), payload["actor_user_id"].(string))
	requireEqual(t, "useful", payload["feedback_type"].(string))
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

	body, _ := json.Marshal(map[string]any{"feedback_type": "invalid"})
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/engrams/"+engramID.String()+"/feedback",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusBadRequest, response.Code)
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

	body, _ := json.Marshal(map[string]any{"feedback_type": "useful"})
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/engrams/"+engramID.String()+"/feedback",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusNotFound, response.Code)
}
