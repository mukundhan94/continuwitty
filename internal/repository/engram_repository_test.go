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
	db := buildNamedQueryEngramsFixture(
		queryEngramFixtureInput{projectID: projectID, kind: queryEngramFixtureDefault},
	)

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

type scoreBandFilterCase struct {
	name         string
	title        string
	min          float64
	max          float64
	fixtureKind  queryEngramFixtureKind
	applyBounds  func(*models.EngramQueryRequest, *float64, *float64)
	scoreFromRow func(models.EngramQueryResult) float64
	scoreLabel   string
}

var scoreBandFilterCaseFixtures = []scoreBandFilterCase{
	{
		name:        "composite rank",
		title:       "Lexical Match",
		min:         0.7,
		max:         0.8,
		fixtureKind: queryEngramFixtureDefault,
		applyBounds: func(request *models.EngramQueryRequest, min *float64, max *float64) {
			request.CompositeRankScoreMin = min
			request.CompositeRankScoreMax = max
		},
		scoreFromRow: func(result models.EngramQueryResult) float64 { return result.CompositeRankScore },
		scoreLabel:   "composite rank",
	},
	{
		name:        "dense",
		title:       "Lexical Match",
		min:         0.79,
		max:         0.81,
		fixtureKind: queryEngramFixtureDefault,
		applyBounds: func(request *models.EngramQueryRequest, min *float64, max *float64) {
			request.DenseScoreMin = min
			request.DenseScoreMax = max
		},
		scoreFromRow: func(result models.EngramQueryResult) float64 { return result.DenseScore },
		scoreLabel:   "dense",
	},
	{
		name:        "lexical overlap",
		title:       "Lexical Match",
		min:         0.9,
		max:         1.0,
		fixtureKind: queryEngramFixtureDefault,
		applyBounds: func(request *models.EngramQueryRequest, min *float64, max *float64) {
			request.LexicalOverlapScoreMin = min
			request.LexicalOverlapScoreMax = max
		},
		scoreFromRow: func(result models.EngramQueryResult) float64 { return result.LexicalOverlapScore },
		scoreLabel:   "lexical overlap",
	},
	{
		name:        "feedback signal",
		title:       "Useful Signal Match",
		min:         0.6,
		max:         0.8,
		fixtureKind: queryEngramFixtureFeedback,
		applyBounds: func(request *models.EngramQueryRequest, min *float64, max *float64) {
			request.FeedbackSignalScoreMin = min
			request.FeedbackSignalScoreMax = max
		},
		scoreFromRow: func(result models.EngramQueryResult) float64 { return result.FeedbackSignalScore },
		scoreLabel:   "feedback signal",
	},
	{
		name:        "engagement signal",
		title:       "Lexical Match",
		min:         0.6,
		max:         0.8,
		fixtureKind: queryEngramFixtureDefault,
		applyBounds: func(request *models.EngramQueryRequest, min *float64, max *float64) {
			request.EngagementSignalScoreMin = min
			request.EngagementSignalScoreMax = max
		},
		scoreFromRow: func(result models.EngramQueryResult) float64 { return result.EngagementSignalScore },
		scoreLabel:   "engagement signal",
	},
	{
		name:        "freshness signal",
		title:       "Lexical Match",
		min:         0.8,
		max:         0.95,
		fixtureKind: queryEngramFixtureDefault,
		applyBounds: func(request *models.EngramQueryRequest, min *float64, max *float64) {
			request.FreshnessSignalScoreMin = min
			request.FreshnessSignalScoreMax = max
		},
		scoreFromRow: func(result models.EngramQueryResult) float64 { return result.FreshnessSignalScore },
		scoreLabel:   "freshness signal",
	},
	{
		name:        "authority signal",
		title:       "Lexical Authority Match",
		min:         0.45,
		max:         0.55,
		fixtureKind: queryEngramFixtureAuthority,
		applyBounds: func(request *models.EngramQueryRequest, min *float64, max *float64) {
			request.AuthoritySignalScoreMin = min
			request.AuthoritySignalScoreMax = max
		},
		scoreFromRow: func(result models.EngramQueryResult) float64 { return result.AuthoritySignalScore },
		scoreLabel:   "authority signal",
	},
}

func TestQueryEngramsAppliesScoreBandFilters(t *testing.T) {
	for _, testCase := range scoreBandFilterCaseFixtures {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertScoreBandFilterCase(t, testCase)
		})
	}
}

func assertScoreBandFilterCase(t *testing.T, testCase scoreBandFilterCase) {
	t.Helper()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	db := buildNamedQueryEngramsFixture(
		queryEngramFixtureInput{projectID: projectID, kind: testCase.fixtureKind},
	)
	request := models.EngramQueryRequest{
		Query:     "durable checkpoint",
		TopK:      2,
		ProjectID: &projectID,
	}
	testCase.applyBounds(&request, &testCase.min, &testCase.max)
	results, err := QueryEngrams(
		context.Background(),
		db,
		QueryEngramsInput{
			Request:      request,
			QueryLiteral: "[0.1,0.2,0.3]",
			ActorUserID:  &actorUserID,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, testCase.title, results[0].Title)
	requireEqual(t, 1, results[0].RankPosition)
	score := testCase.scoreFromRow(results[0])
	if score < testCase.min {
		t.Fatalf("expected %s >= %v, got %v", testCase.scoreLabel, testCase.min, score)
	}
	if score > testCase.max {
		t.Fatalf("expected %s <= %v, got %v", testCase.scoreLabel, testCase.max, score)
	}
}

type queryEngramFixtureKind int

const (
	queryEngramFixtureDefault queryEngramFixtureKind = iota + 1
	queryEngramFixtureAuthority
	queryEngramFixtureFeedback
)

type queryEngramFixtureInput struct {
	projectID string
	kind      queryEngramFixtureKind
}

func buildNamedQueryEngramsFixture(input queryEngramFixtureInput) *fakeQueryer {
	createdDense := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	createdLexical := time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC)
	denseCandidate := queryEngramsFixtureCandidate{
		id:           "00000000-0000-0000-0000-000000000211",
		title:        "Dense Match",
		createdAt:    createdDense,
		keywords:     []string{},
		retrieval:    "unrelated text",
		usefulCount:  0,
		feedback:     2,
		avgRelevance: 0.55,
		accessCount:  1,
		freshness:    0.4,
		sourceScore:  0.5,
		distance:     0.2,
	}
	lexicalCandidate := queryEngramsFixtureCandidate{
		id:           "00000000-0000-0000-0000-000000000212",
		title:        "Lexical Match",
		createdAt:    createdLexical,
		keywords:     []string{"durable", "checkpoint"},
		retrieval:    "durable checkpoint lifecycle",
		usefulCount:  0,
		feedback:     1,
		avgRelevance: 0.82,
		accessCount:  7,
		freshness:    0.88,
		sourceScore:  0.5,
		distance:     0.25,
	}
	switch input.kind {
	case queryEngramFixtureAuthority:
		denseCandidate.id = "00000000-0000-0000-0000-000000000231"
		denseCandidate.title = "Dense Authority Match"
		denseCandidate.sourceScore = 0.2
		lexicalCandidate.id = "00000000-0000-0000-0000-000000000232"
		lexicalCandidate.title = "Lexical Authority Match"
	case queryEngramFixtureFeedback:
		denseCandidate.id = "00000000-0000-0000-0000-000000000221"
		denseCandidate.title = "Contradiction Signal Match"
		denseCandidate.feedback = 1
		denseCandidate.contradictionCount = 1
		lexicalCandidate.id = "00000000-0000-0000-0000-000000000222"
		lexicalCandidate.title = "Useful Signal Match"
		lexicalCandidate.usefulCount = 1
	}
	return buildQueryEngramsFixtureFromCandidates(input.projectID, denseCandidate, lexicalCandidate)
}

type queryEngramsFixtureCandidate struct {
	id                 string
	title              string
	createdAt          time.Time
	keywords           []string
	retrieval          string
	usefulCount        int
	feedback           int
	avgRelevance       float64
	contradictionCount int
	accessCount        int
	freshness          float64
	sourceScore        float64
	distance           float64
}

func buildQueryEngramsFixtureFromCandidates(
	projectID string,
	candidates ...queryEngramsFixtureCandidate,
) *fakeQueryer {
	values := make([][]any, 0, len(candidates))
	for _, candidate := range candidates {
		values = append(values, buildQueryEngramsFixtureRow(projectID, candidate))
	}
	return &fakeQueryer{queryRowsResult: &fakeRows{values: values}}
}

func buildQueryEngramsFixtureRow(projectID string, candidate queryEngramsFixtureCandidate) []any {
	return []any{
		uuid.MustParse(candidate.id),
		projectID,
		candidate.title,
		"",
		candidate.createdAt,
		[]string{},
		candidate.keywords,
		nil,
		"private",
		candidate.retrieval,
		candidate.usefulCount,
		candidate.feedback,
		candidate.avgRelevance,
		candidate.contradictionCount,
		candidate.accessCount,
		candidate.freshness,
		candidate.sourceScore,
		candidate.distance,
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
