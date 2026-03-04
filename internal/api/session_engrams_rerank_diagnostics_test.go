package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestMountSessionAuthRoutesQueryEngramsReturnsRerankDiagnostics(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor:        actor,
			queryEngrams: buildRerankDiagnosticsQueryHandler(t, actor),
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
	response := executeEngramRequest(
		t,
		handler,
		loginCookie,
		engramRequestSpec{
			method: http.MethodPost,
			path:   "/api/v1/engrams/query",
			body: map[string]any{
				"query": "authority scoped recall",
			},
		},
	)
	requireEqual(t, http.StatusOK, response.Code)

	var payload []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode query response: %v", err)
	}
	requireEqual(t, 1, len(payload))
	assertRerankDiagnosticsPayload(t, payload[0])
}

func buildRerankDiagnosticsQueryHandler(
	t *testing.T,
	actor *models.UserAuthRecord,
) func(context.Context, models.EngramQueryRequest, uuid.UUID) ([]models.EngramQueryResult, error) {
	t.Helper()
	return func(
		_ context.Context,
		_ models.EngramQueryRequest,
		actorUserID uuid.UUID,
	) ([]models.EngramQueryResult, error) {
		requireEqual(t, actor.UserID, actorUserID)
		return []models.EngramQueryResult{rerankDiagnosticResult()}, nil
	}
}

func rerankDiagnosticResult() models.EngramQueryResult {
	return models.EngramQueryResult{
		EngramID:                   uuid.MustParse("00000000-0000-0000-0000-000000000a81"),
		ProjectID:                  "proj-1",
		Title:                      "Authority scoped memory",
		Abstract:                   "Authority score should be visible",
		CreatedAt:                  time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC),
		VisibilityScope:            "private",
		AccessCount:                8,
		FreshnessScore:             0.74,
		FeedbackCount:              6,
		UsefulCount:                5,
		AvgRelevanceFeedback:       0.77,
		UsefulFeedbackRatio:        0.83,
		ContradictionCount:         2,
		ContradictionFeedbackRatio: 0.33,
		SourceSessionQualityScore:  0.81,
		CompositeRankScore:         0.79,
		DenseScore:                 0.89,
		LexicalOverlapScore:        0.66,
		FeedbackSignalScore:        0.72,
		EngagementSignalScore:      0.63,
		FreshnessSignalScore:       0.74,
		AuthoritySignalScore:       0.81,
		RankPosition:               1,
		Distance:                   0.12,
	}
}

func assertRerankDiagnosticsPayload(t *testing.T, payload map[string]any) {
	t.Helper()
	expected := map[string]float64{
		"access_count":                 8,
		"freshness_score":              0.74,
		"feedback_count":               6,
		"useful_count":                 5,
		"avg_relevance_feedback":       0.77,
		"useful_feedback_ratio":        0.83,
		"contradiction_count":          2,
		"contradiction_feedback_ratio": 0.33,
		"source_session_quality_score": 0.81,
		"composite_rank_score":         0.79,
		"dense_score":                  0.89,
		"lexical_overlap_score":        0.66,
		"feedback_signal_score":        0.72,
		"engagement_signal_score":      0.63,
		"freshness_signal_score":       0.74,
		"authority_signal_score":       0.81,
		"rank_position":                1,
	}
	for field, expectedValue := range expected {
		actual, ok := payload[field].(float64)
		if !ok {
			t.Fatalf("expected numeric payload field %s", field)
		}
		requireEqual(t, expectedValue, actual)
	}
}
