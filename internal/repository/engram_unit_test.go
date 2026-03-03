package repository

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestBuildEngramQueryWhereIncludesAllFilters(t *testing.T) {
	fixture := buildQueryWhereAllFiltersFixture()
	whereSQL, params := buildEngramQueryWhere(
		fixture.request,
		&fixture.actorUserID,
		"[0.1,0.2,0.3]",
	)

	for _, fragment := range fixture.expectedFragments {
		if !strings.Contains(whereSQL, fragment) {
			t.Fatalf("expected where sql to contain %q, got %q", fragment, whereSQL)
		}
	}
	if !reflect.DeepEqual(params, fixture.expectedParams) {
		t.Fatalf("expected params %#v, got %#v", fixture.expectedParams, params)
	}
}

type queryWhereAllFiltersFixture struct {
	actorUserID       uuid.UUID
	request           models.EngramQueryRequest
	expectedFragments []string
	expectedParams    []any
}

func buildQueryWhereAllFiltersFixture() queryWhereAllFiltersFixture {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	createdAfter := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	createdBefore := time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC)
	accessCountMin := 3
	freshnessScoreMin := 0.6
	sourceSessionQualityMin := 0.7
	lastAccessedAfter := time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)
	lastAccessedBefore := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	freshnessComputedAfter := time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)
	freshnessComputedBefore := time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC)
	relationType := models.EngramLinkRelationSupports
	traceDepth := 1
	projectID := "engram-vault"
	return queryWhereAllFiltersFixture{
		actorUserID: actorUserID,
		request: models.EngramQueryRequest{
			Query:                   "durable memory",
			TopK:                    5,
			ProjectID:               &projectID,
			Tags:                    []string{"memory"},
			Keywords:                []string{"checkpoint"},
			CreatedAfter:            &createdAfter,
			CreatedBefore:           &createdBefore,
			AccessCountMin:          &accessCountMin,
			FreshnessScoreMin:       &freshnessScoreMin,
			SourceSessionQualityMin: &sourceSessionQualityMin,
			LastAccessedAfter:       &lastAccessedAfter,
			LastAccessedBefore:      &lastAccessedBefore,
			FreshnessComputedAfter:  &freshnessComputedAfter,
			FreshnessComputedBefore: &freshnessComputedBefore,
			RelationType:            &relationType,
			TraceDepth:              &traceDepth,
		},
		expectedFragments: []string{
			"WHERE deleted_at IS NULL",
			"project_id = $2",
			"tags && $4",
			"keywords && $5",
			"created_at >= $6",
			"created_at <= $7",
			"COALESCE(access_count, 0) >= $8",
			"COALESCE(freshness_score, 1.0) >= $9",
			"COALESCE(source_session_quality_score, 0.5) >= $10",
			"COALESCE(last_accessed_at, created_at) >= $11",
			"COALESCE(last_accessed_at, created_at) <= $12",
			"COALESCE(freshness_last_computed_at, created_at) >= $13",
			"COALESCE(freshness_last_computed_at, created_at) <= $14",
			"EXISTS (",
			"link.status = 'active'",
			"link.relation_type = $15",
			"owner_user_id = $3",
			"actor_user.user_id = $3",
			"pm.project_id = project_id",
			"pm.user_id = $3",
			"visibility_scope = 'project'",
		},
		expectedParams: []any{
			"[0.1,0.2,0.3]",
			"engram-vault",
			actorUserID,
			[]string{"memory"},
			[]string{"checkpoint"},
			createdAfter,
			createdBefore,
			accessCountMin,
			freshnessScoreMin,
			sourceSessionQualityMin,
			lastAccessedAfter,
			lastAccessedBefore,
			freshnessComputedAfter,
			freshnessComputedBefore,
			string(relationType),
		},
	}
}

func TestRerankByCombinedScorePrefersLexicalOverlap(t *testing.T) {
	rows := []map[string]any{
		{
			"engram_id":      uuid.MustParse("00000000-0000-0000-0000-000000000111"),
			"project_id":     "engram-vault",
			"title":          "Dense Match",
			"abstract":       "",
			"created_at":     time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC),
			"tags":           []string{},
			"keywords":       []string{},
			"retrieval_text": "unrelated text",
			"distance":       0.2,
		},
		{
			"engram_id":      uuid.MustParse("00000000-0000-0000-0000-000000000222"),
			"project_id":     "engram-vault",
			"title":          "Lexical Match",
			"abstract":       "",
			"created_at":     time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC),
			"tags":           []string{},
			"keywords":       []string{"durable", "checkpoint"},
			"retrieval_text": "durable checkpoint lifecycle",
			"distance":       0.25,
		},
	}

	reranked := rerankByCombinedScore(
		rerankRowsInput{
			rows:  rows,
			query: "durable checkpoint",
			topK:  1,
		},
	)
	if len(reranked) != 1 {
		t.Fatalf("expected one reranked result, got %d", len(reranked))
	}
	if title, _ := reranked[0]["title"].(string); title != "Lexical Match" {
		t.Fatalf("expected lexical match to rank first, got %q", title)
	}
}

func TestRerankByCombinedScoreIncorporatesFeedbackSignal(t *testing.T) {
	rows := []map[string]any{
		newRerankSignalRow(
			rerankSignalRowInput{
				engramID:           uuid.MustParse("00000000-0000-0000-0000-000000000311"),
				title:              "Useful Memory",
				usefulCount:        10,
				contradictionCount: 0,
				accessCount:        6,
				freshnessScore:     0.85,
				distance:           0.3,
			},
		),
		newRerankSignalRow(
			rerankSignalRowInput{
				engramID:           uuid.MustParse("00000000-0000-0000-0000-000000000312"),
				title:              "Contradicted Memory",
				usefulCount:        0,
				contradictionCount: 8,
				accessCount:        6,
				freshnessScore:     0.85,
				distance:           0.3,
			},
		),
	}

	reranked := rerankByCombinedScore(
		rerankRowsInput{
			rows:  rows,
			query: "checkpoint details",
			topK:  1,
		},
	)
	if len(reranked) != 1 {
		t.Fatalf("expected one reranked result, got %d", len(reranked))
	}
	if title, _ := reranked[0]["title"].(string); title != "Useful Memory" {
		t.Fatalf("expected positive feedback memory to rank first, got %q", title)
	}
}

func TestRerankByCombinedScoreIncorporatesEngagementAndFreshnessSignals(t *testing.T) {
	rows := []map[string]any{
		newRerankSignalRow(
			rerankSignalRowInput{
				engramID:           uuid.MustParse("00000000-0000-0000-0000-000000000321"),
				title:              "Active Fresh Memory",
				usefulCount:        2,
				contradictionCount: 0,
				accessCount:        28,
				freshnessScore:     0.95,
				distance:           0.35,
				keywords:           []string{"runbook"},
				retrievalText:      "runbook context",
				abstract:           "shared runbook context",
			},
		),
		newRerankSignalRow(
			rerankSignalRowInput{
				engramID:           uuid.MustParse("00000000-0000-0000-0000-000000000322"),
				title:              "Cold Stale Memory",
				usefulCount:        2,
				contradictionCount: 0,
				accessCount:        0,
				freshnessScore:     0.05,
				distance:           0.35,
				keywords:           []string{"runbook"},
				retrievalText:      "runbook context",
				abstract:           "shared runbook context",
			},
		),
	}

	reranked := rerankByCombinedScore(
		rerankRowsInput{
			rows:  rows,
			query: "runbook context",
			topK:  1,
		},
	)
	if len(reranked) != 1 {
		t.Fatalf("expected one reranked result, got %d", len(reranked))
	}
	if title, _ := reranked[0]["title"].(string); title != "Active Fresh Memory" {
		t.Fatalf("expected active/fresh memory to rank first, got %q", title)
	}
}

func TestRerankByCombinedScoreIncorporatesAuthoritySignal(t *testing.T) {
	rows := []map[string]any{
		newRerankSignalRow(
			rerankSignalRowInput{
				engramID:                     uuid.MustParse("00000000-0000-0000-0000-000000000331"),
				title:                        "High Authority Memory",
				usefulCount:                  2,
				contradictionCount:           0,
				accessCount:                  8,
				freshnessScore:               0.7,
				sourceSessionQualityScore:    0.95,
				hasSourceSessionQualityScore: true,
				distance:                     0.35,
			},
		),
		newRerankSignalRow(
			rerankSignalRowInput{
				engramID:                     uuid.MustParse("00000000-0000-0000-0000-000000000332"),
				title:                        "Low Authority Memory",
				usefulCount:                  2,
				contradictionCount:           0,
				accessCount:                  8,
				freshnessScore:               0.7,
				sourceSessionQualityScore:    0.1,
				hasSourceSessionQualityScore: true,
				distance:                     0.35,
			},
		),
	}

	reranked := rerankByCombinedScore(
		rerankRowsInput{
			rows:  rows,
			query: "checkpoint details",
			topK:  1,
		},
	)
	if len(reranked) != 1 {
		t.Fatalf("expected one reranked result, got %d", len(reranked))
	}
	if title, _ := reranked[0]["title"].(string); title != "High Authority Memory" {
		t.Fatalf("expected high-authority memory to rank first, got %q", title)
	}
}

func TestFormatCitationsTruncatesAndStripsNewlines(t *testing.T) {
	title := "Primary source"
	citationText := formatCitations(
		[]models.RehydrationCitation{
			{
				URL:        "https://example.com/source",
				Title:      &title,
				Snippet:    "line1\n" + strings.Repeat("x", 200),
				CapturedAt: time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC),
			},
		},
	)

	if !strings.HasPrefix(citationText, "- Primary source (https://example.com/source):") {
		t.Fatalf("unexpected citation prefix: %q", citationText)
	}
	pieces := strings.SplitN(citationText, ":", 2)
	if len(pieces) != 2 {
		t.Fatalf("expected citation with snippet section, got %q", citationText)
	}
	if strings.Contains(pieces[1], "\n") {
		t.Fatalf("expected no newlines in snippet, got %q", pieces[1])
	}
	if !strings.HasSuffix(citationText, "...") {
		t.Fatalf("expected truncated snippet to end with ellipsis, got %q", citationText)
	}
}

func TestFormatCitationsReturnsDefaultForEmptyList(t *testing.T) {
	if formatCitations([]models.RehydrationCitation{}) != "- No citations available" {
		t.Fatalf("expected default citations text for empty list")
	}
}

func TestFormatDecisionsFormatsEntriesAndDefaults(t *testing.T) {
	decisionsText := formatDecisions(
		[]map[string]any{
			{
				"decision":  "Use durable checkpoints",
				"rationale": "Prevents context loss",
			},
		},
	)
	if decisionsText != "- Use durable checkpoints: Prevents context loss" {
		t.Fatalf("unexpected decisions text %q", decisionsText)
	}
	if formatDecisions([]map[string]any{}) != "- None" {
		t.Fatalf("expected decisions default text")
	}
}

func TestFormatOpenQuestionsFormatsEntriesAndDefaults(t *testing.T) {
	if got := formatOpenQuestions([]string{"What caused drift?"}); got != "- What caused drift?" {
		t.Fatalf("unexpected open questions text %q", got)
	}
	if got := formatOpenQuestions([]string{}); got != "- None" {
		t.Fatalf("expected open questions default text")
	}
}

func TestBuildRehydrationContextMarkdownIncludesExpectedSections(t *testing.T) {
	testCases := []struct {
		name                   string
		detailedExcerpt        string
		expectsDetailedSection bool
	}{
		{
			name:                   "with detailed excerpt",
			detailedExcerpt:        "Detailed analysis excerpt.",
			expectsDetailedSection: true,
		},
		{
			name:                   "without detailed excerpt",
			detailedExcerpt:        "",
			expectsDetailedSection: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			title := "Source"
			markdown := buildRehydrationContextMarkdown(
				rehydrationContextParts{
					Title:           "Checkpoint Summary",
					CompactSummary:  "Compact summary body.",
					DetailedExcerpt: testCase.detailedExcerpt,
					Decisions: []map[string]any{
						{
							"decision":  "Use snapshots",
							"rationale": "Improves continuity",
						},
					},
					OpenQuestions: []string{"Need retention policy?"},
					Citations: []models.RehydrationCitation{
						{
							URL:        "https://example.com/source",
							Title:      &title,
							Snippet:    "Key evidence.",
							CapturedAt: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
						},
					},
				},
			)

			if !strings.HasPrefix(markdown, "# Rehydration Context: Checkpoint Summary") {
				t.Fatalf("unexpected markdown prefix: %q", markdown)
			}
			if !strings.Contains(markdown, "## Compact Summary\nCompact summary body.") {
				t.Fatalf("missing compact summary section: %q", markdown)
			}
			hasDetailedSection := strings.Contains(markdown, "## Detailed Notes Excerpt")
			if hasDetailedSection != testCase.expectsDetailedSection {
				t.Fatalf(
					"expected detailed section present=%v, got %v",
					testCase.expectsDetailedSection,
					hasDetailedSection,
				)
			}
			if !strings.Contains(markdown, "## Key Decisions\n- Use snapshots: Improves continuity") {
				t.Fatalf("missing decisions section: %q", markdown)
			}
			if !strings.Contains(markdown, "## Open Questions\n- Need retention policy?") {
				t.Fatalf("missing open questions section: %q", markdown)
			}
			if !strings.Contains(markdown, "## Top Citations\n- Source (https://example.com/source): Key evidence.") {
				t.Fatalf("missing top citations section: %q", markdown)
			}
		})
	}
}

func TestBuildEngramJSONPayloadSerializesReportAndSourceSessionID(t *testing.T) {
	sourceSessionID := uuid.MustParse("00000000-0000-0000-0000-000000009999")
	payload := buildEngramJSONPayload(
		models.MemoryEngramCreate{
			ProjectID:               "project-1",
			ThreadID:                ptr("thread-1"),
			Title:                   "Checkpoint",
			Abstract:                "Summary",
			DetailedSummaryMarkdown: "Detailed body",
			SourceSessionID:         &sourceSessionID,
		},
		map[string]any{"enrichment_applied": true, "schema_version": "1.0"},
		time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC),
	)

	if got := payload["project_id"]; got != "project-1" {
		t.Fatalf("expected project_id project-1, got %#v", got)
	}
	if got := payload["thread_id"]; got != "thread-1" {
		t.Fatalf("expected thread_id thread-1, got %#v", got)
	}
	if got := payload["title"]; got != "Checkpoint" {
		t.Fatalf("expected title Checkpoint, got %#v", got)
	}
	if got := payload["source_session_id"]; got != sourceSessionID.String() {
		t.Fatalf("expected source_session_id %q, got %#v", sourceSessionID.String(), got)
	}
	autoMetadata, ok := payload["auto_metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected auto_metadata map, got %#v", payload["auto_metadata"])
	}
	if got := autoMetadata["enrichment_applied"]; got != true {
		t.Fatalf("expected enrichment_applied true, got %#v", got)
	}
	if got := payload["created_at"]; got != "2026-02-20T12:00:00+00:00" {
		t.Fatalf("expected created_at 2026-02-20T12:00:00+00:00, got %#v", got)
	}
}

func ptr(value string) *string {
	return &value
}

type rerankSignalRowInput struct {
	engramID                     uuid.UUID
	title                        string
	abstract                     string
	keywords                     []string
	retrievalText                string
	usefulCount                  int
	contradictionCount           int
	accessCount                  int
	freshnessScore               float64
	sourceSessionQualityScore    float64
	hasSourceSessionQualityScore bool
	distance                     float64
}

func newRerankSignalRow(input rerankSignalRowInput) map[string]any {
	if input.abstract == "" {
		input.abstract = "shared checkpoint details"
	}
	if input.retrievalText == "" {
		input.retrievalText = "checkpoint details"
	}
	if input.keywords == nil {
		input.keywords = []string{"checkpoint"}
	}
	if !input.hasSourceSessionQualityScore {
		input.sourceSessionQualityScore = 0.5
	}
	return map[string]any{
		"engram_id":                    input.engramID,
		"project_id":                   "engram-vault",
		"title":                        input.title,
		"abstract":                     input.abstract,
		"created_at":                   time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC),
		"tags":                         []string{},
		"keywords":                     input.keywords,
		"retrieval_text":               input.retrievalText,
		"useful_count":                 input.usefulCount,
		"contradiction_count":          input.contradictionCount,
		"access_count":                 input.accessCount,
		"freshness_score":              input.freshnessScore,
		"source_session_quality_score": input.sourceSessionQualityScore,
		"distance":                     input.distance,
	}
}
