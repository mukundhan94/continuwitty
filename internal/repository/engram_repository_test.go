package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestListEngramsAppliesVisibilityAndProjectFilters(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000111")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000011")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000999")
	projectID := "engram-vault"
	threadID := "thread-1"
	createdAt := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	visibilityScope := "project"
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					engramID,
					projectID,
					threadID,
					"Checkpoint",
					"Summary",
					createdAt,
					[]string{"durable"},
					[]string{"checkpoint"},
					ownerUserID,
					visibilityScope,
				},
			},
		},
	}

	results, err := ListEngrams(
		context.Background(),
		db,
		ListEngramsInput{
			ProjectID:   &projectID,
			Limit:       25,
			Offset:      0,
			ActorUserID: &actorUserID,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, engramID, results[0].EngramID)
	requireEqual(t, "project", results[0].VisibilityScope)

	requireEqual(t, 1, len(db.querySQL))
	query := db.querySQL[0]
	requiredFragments := []string{
		"owner_user_id = $1",
		"actor_user.user_id = $1",
		"visibility_scope = 'project'",
		"pm.project_id = project_id",
		"pm.user_id = $1",
		"owner_user_id IS NULL",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(query, fragment) {
			t.Fatalf("expected visibility fragment %q in query, got %q", fragment, query)
		}
	}
	if !strings.Contains(query, "OR (visibility_scope = 'project'") {
		t.Fatalf("expected visibility clause in query, got %q", query)
	}
	if !strings.Contains(query, "project_id = $2") {
		t.Fatalf("expected project filter in query, got %q", query)
	}
	expectedArgs := []any{actorUserID, projectID, 25, 0}
	if !reflect.DeepEqual(db.queryArgs[0], expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestListEngramsWithoutActorOmitsVisibilityClause(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000112")
	threadID := "thread-2"
	createdAt := time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC)
	visibilityScope := "private"
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					engramID,
					"project-public",
					threadID,
					"Public checkpoint",
					"Summary",
					createdAt,
					[]string{},
					[]string{},
					nil,
					visibilityScope,
				},
			},
		},
	}

	results, err := ListEngrams(
		context.Background(),
		db,
		ListEngramsInput{
			Limit:  10,
			Offset: 5,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, "private", results[0].VisibilityScope)

	requireEqual(t, 1, len(db.querySQL))
	query := db.querySQL[0]
	if strings.Contains(query, "FROM project_members pm") {
		t.Fatalf("did not expect visibility clause when actor is nil: %q", query)
	}
	expectedArgs := []any{10, 5}
	if !reflect.DeepEqual(db.queryArgs[0], expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestQueryEngramsBuildsQueryAndReranks(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	db := buildQueryEngramsFixture(projectID)

	results, err := QueryEngrams(
		context.Background(),
		db,
		QueryEngramsInput{
			Request: models.EngramQueryRequest{
				Query:     "durable checkpoint",
				TopK:      1,
				ProjectID: &projectID,
			},
			QueryLiteral: "[0.1,0.2,0.3]",
			ActorUserID:  &actorUserID,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, "Lexical Match", results[0].Title)
	requireEqual(t, 7, results[0].AccessCount)
	requireEqual(t, 0.88, results[0].FreshnessScore)
	requireEqual(t, 1, results[0].FeedbackCount)
	requireEqual(t, 0, results[0].UsefulCount)
	requireEqual(t, 0.82, results[0].AvgRelevanceFeedback)
	requireEqual(t, 0.0, results[0].UsefulFeedbackRatio)
	requireEqual(t, 0, results[0].ContradictionCount)
	requireEqual(t, 0.0, results[0].ContradictionFeedbackRatio)
	requireEqual(t, 0.5, results[0].SourceSessionQualityScore)
	requireEqual(t, denseDistanceScore(0.25), results[0].DenseScore)
	requireEqual(t, 1.0, results[0].LexicalOverlapScore)
	requireEqual(t, normalizeFeedbackScore(0, 0), results[0].FeedbackSignalScore)
	requireEqual(t, normalizeEngagementScore(7), results[0].EngagementSignalScore)
	requireEqual(t, 0.88, results[0].FreshnessSignalScore)
	requireEqual(t, 0.5, results[0].AuthoritySignalScore)
	requireEqual(t, 1, results[0].RankPosition)
	requireEqual(
		t,
		combinedRankScore(
			rankScoreInput{
				distance:        0.25,
				lexicalOverlap:  1.0,
				feedbackScore:   normalizeFeedbackScore(0, 0),
				engagementScore: normalizeEngagementScore(7),
				freshnessScore:  0.88,
				authorityScore:  0.5,
			},
		),
		results[0].CompositeRankScore,
	)
	assertQueryEngramsRuntimeQuery(
		t,
		queryRuntimeAssertionInput{
			query:       db.querySQL[0],
			args:        db.queryArgs[0],
			projectID:   projectID,
			actorUserID: actorUserID,
		},
	)
}

func TestQueryEngramsAppliesCompositeRankScoreFilters(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	minScore := 0.7
	maxScore := 0.8
	db := buildQueryEngramsFixture(projectID)

	results, err := QueryEngrams(
		context.Background(),
		db,
		QueryEngramsInput{
			Request: models.EngramQueryRequest{
				Query:                 "durable checkpoint",
				TopK:                  2,
				ProjectID:             &projectID,
				CompositeRankScoreMin: &minScore,
				CompositeRankScoreMax: &maxScore,
			},
			QueryLiteral: "[0.1,0.2,0.3]",
			ActorUserID:  &actorUserID,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, "Lexical Match", results[0].Title)
	requireEqual(t, 1, results[0].RankPosition)
	if results[0].CompositeRankScore < minScore {
		t.Fatalf("expected composite score >= %v, got %v", minScore, results[0].CompositeRankScore)
	}
	if results[0].CompositeRankScore > maxScore {
		t.Fatalf("expected composite score <= %v, got %v", maxScore, results[0].CompositeRankScore)
	}
}

func buildQueryEngramsFixture(projectID string) *fakeQueryer {
	createdDense := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	createdLexical := time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC)
	return &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					uuid.MustParse("00000000-0000-0000-0000-000000000211"),
					projectID,
					"Dense Match",
					"",
					createdDense,
					[]string{},
					[]string{},
					nil,
					"private",
					"unrelated text",
					0,
					2,
					0.55,
					0,
					1,
					0.4,
					0.5,
					0.2,
				},
				{
					uuid.MustParse("00000000-0000-0000-0000-000000000212"),
					projectID,
					"Lexical Match",
					"",
					createdLexical,
					[]string{},
					[]string{"durable", "checkpoint"},
					nil,
					"private",
					"durable checkpoint lifecycle",
					0,
					1,
					0.82,
					0,
					7,
					0.88,
					0.5,
					0.25,
				},
			},
		},
	}
}

type queryRuntimeAssertionInput struct {
	query       string
	args        []any
	projectID   string
	actorUserID uuid.UUID
}

func assertQueryEngramsRuntimeQuery(t *testing.T, input queryRuntimeAssertionInput) {
	t.Helper()
	assertQueryHasRerankSignalClauses(t, input.query)
	if !strings.Contains(input.query, "WHERE deleted_at IS NULL AND project_id = $2") {
		t.Fatalf("expected project clause with pgx placeholders, got %q", input.query)
	}
	requiredVisibilityFragments := []string{
		"owner_user_id = $3",
		"actor_user.user_id = $3",
		"visibility_scope = 'project'",
		"pm.project_id = project_id",
		"pm.user_id = $3",
		"owner_user_id IS NULL",
	}
	for _, fragment := range requiredVisibilityFragments {
		if !strings.Contains(input.query, fragment) {
			t.Fatalf("expected actor visibility fragment %q in query, got %q", fragment, input.query)
		}
	}
	expectedArgs := []any{"[0.1,0.2,0.3]", input.projectID, input.actorUserID, 4}
	if !reflect.DeepEqual(input.args, expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, input.args)
	}
}

func assertQueryHasRerankSignalClauses(t *testing.T, query string) {
	t.Helper()
	if !strings.Contains(query, "embed <=> $1::vector AS distance") {
		t.Fatalf("expected vector distance clause in query, got %q", query)
	}
	if !strings.Contains(query, "COALESCE(access_count, 0) AS access_count") {
		t.Fatalf("expected access_count clause in query, got %q", query)
	}
	if !strings.Contains(query, "COALESCE(freshness_score, 1.0) AS freshness_score") {
		t.Fatalf("expected freshness_score clause in query, got %q", query)
	}
	if !strings.Contains(query, "COALESCE(avg_relevance_feedback, 0.5) AS avg_relevance_feedback") {
		t.Fatalf("expected avg_relevance_feedback clause in query, got %q", query)
	}
	if !strings.Contains(query, "COALESCE(source_session_quality_score, 0.5) AS source_session_quality_score") {
		t.Fatalf("expected source_session_quality_score clause in query, got %q", query)
	}
}
