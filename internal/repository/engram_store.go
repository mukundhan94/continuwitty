package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

// ListEngramsInput captures list query filters and pagination.
type ListEngramsInput struct {
	ProjectID   *string
	Limit       int
	Offset      int
	ActorUserID *uuid.UUID
}

// QueryEngramsInput captures query operation dependencies.
type QueryEngramsInput struct {
	Request      models.EngramQueryRequest
	QueryLiteral string
	ActorUserID  *uuid.UUID
}

// ListEngrams returns engram summaries with optional project and actor scoping.
func ListEngrams(ctx context.Context, db Queryer, input ListEngramsInput) ([]models.EngramSummary, error) {
	baseQuery := `
		SELECT
			engram_id, project_id, thread_id, title, abstract, created_at, tags, keywords,
			owner_user_id, visibility_scope
		FROM engrams
	`

	whereClauses := []string{"deleted_at IS NULL"}
	params := make([]any, 0, 4)

	if input.ActorUserID != nil {
		actorPlaceholder := pgxPlaceholder(len(params) + 1)
		whereClauses = append(
			whereClauses,
			buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "owner_user_id", visibilityColumn: "visibility_scope", projectColumn: "project_id", actorPlaceholder: actorPlaceholder, includeOwnerless: true}),
		)
		params = append(params, *input.ActorUserID)
	}
	if input.ProjectID != nil && *input.ProjectID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("project_id = %s", pgxPlaceholder(len(params)+1)))
		params = append(params, *input.ProjectID)
	}

	sql := baseQuery + " WHERE " + strings.Join(whereClauses, " AND ")
	sql += fmt.Sprintf(
		" ORDER BY created_at DESC LIMIT %s OFFSET %s",
		pgxPlaceholder(len(params)+1),
		pgxPlaceholder(len(params)+2),
	)
	params = append(params, input.Limit, input.Offset)

	rows, err := db.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.EngramSummary, 0)
	for rows.Next() {
		record, scanErr := scanEngramSummary(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		results = append(results, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// QueryEngrams queries candidate rows by vector distance and reranks with lexical overlap.
func QueryEngrams(ctx context.Context, db Queryer, input QueryEngramsInput) ([]models.EngramQueryResult, error) {
	topK := input.Request.TopK
	if topK <= 0 {
		topK = 5
	}

	whereSQL, params := buildEngramQueryWhere(
		input.Request,
		input.ActorUserID,
		input.QueryLiteral,
	)
	candidateLimit := min(max(topK*4, topK), 200)
	limitPlaceholder := pgxPlaceholder(len(params) + 1)

	sql := fmt.Sprintf(
		`
		SELECT
			engram_id,
			project_id,
			title,
			abstract,
			created_at,
			tags,
			keywords,
			owner_user_id,
			visibility_scope,
			retrieval_text,
			COALESCE(useful_count, 0) AS useful_count,
			COALESCE(feedback_count, 0) AS feedback_count,
			COALESCE(contradiction_count, 0) AS contradiction_count,
			COALESCE(access_count, 0) AS access_count,
			COALESCE(freshness_score, 1.0) AS freshness_score,
			COALESCE(source_session_quality_score, 0.5) AS source_session_quality_score,
			embed <=> $1::vector AS distance
		FROM engrams
		%s
		ORDER BY distance ASC, created_at DESC
		LIMIT %s
		`,
		whereSQL,
		limitPlaceholder,
	)
	params = append(params, candidateLimit)

	rows, err := db.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidateRows := make([]map[string]any, 0)
	for rows.Next() {
		candidate, scanErr := scanEngramCandidateRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		candidateRows = append(candidateRows, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rerankedRows := rerankByCombinedScore(
		rerankRowsInput{
			rows:  candidateRows,
			query: input.Request.Query,
			topK:  topK,
		},
	)
	results := make([]models.EngramQueryResult, 0, len(rerankedRows))
	for _, row := range rerankedRows {
		results = append(results, mapEngramQueryResult(row))
	}
	return results, nil
}

func scanEngramSummary(row interface {
	Scan(dest ...any) error
}) (models.EngramSummary, error) {
	var (
		record          models.EngramSummary
		tags            []string
		keywords        []string
		visibilityScope *string
	)
	err := row.Scan(
		&record.EngramID,
		&record.ProjectID,
		&record.ThreadID,
		&record.Title,
		&record.Abstract,
		&record.CreatedAt,
		&tags,
		&keywords,
		&record.OwnerUserID,
		&visibilityScope,
	)
	if err != nil {
		return models.EngramSummary{}, err
	}
	record.Tags = tags
	if record.Tags == nil {
		record.Tags = []string{}
	}
	record.Keywords = keywords
	if record.Keywords == nil {
		record.Keywords = []string{}
	}
	record.VisibilityScope = visibilityOrDefault(visibilityScope)
	return record, nil
}

func scanEngramCandidateRow(row interface {
	Scan(dest ...any) error
}) (map[string]any, error) {
	var (
		engramID                  uuid.UUID
		projectID                 string
		title                     string
		abstract                  string
		createdAt                 time.Time
		tags                      []string
		keywords                  []string
		ownerUserID               *uuid.UUID
		visibilityScope           *string
		retrievalText             string
		usefulCount               int
		feedbackCount             int
		contradiction             int
		accessCount               int
		freshnessScore            float64
		sourceSessionQualityScore float64
		distance                  float64
	)

	err := row.Scan(
		&engramID,
		&projectID,
		&title,
		&abstract,
		&createdAt,
		&tags,
		&keywords,
		&ownerUserID,
		&visibilityScope,
		&retrievalText,
		&usefulCount,
		&feedbackCount,
		&contradiction,
		&accessCount,
		&freshnessScore,
		&sourceSessionQualityScore,
		&distance,
	)
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []string{}
	}
	if keywords == nil {
		keywords = []string{}
	}
	return map[string]any{
		"engram_id":                    engramID,
		"project_id":                   projectID,
		"title":                        title,
		"abstract":                     abstract,
		"created_at":                   createdAt,
		"tags":                         tags,
		"keywords":                     keywords,
		"owner_user_id":                ownerUserID,
		"visibility_scope":             visibilityOrDefault(visibilityScope),
		"retrieval_text":               retrievalText,
		"useful_count":                 usefulCount,
		"feedback_count":               feedbackCount,
		"contradiction_count":          contradiction,
		"access_count":                 accessCount,
		"freshness_score":              freshnessScore,
		"source_session_quality_score": sourceSessionQualityScore,
		"distance":                     distance,
	}, nil
}

func mapEngramQueryResult(row map[string]any) models.EngramQueryResult {
	result := models.EngramQueryResult{
		EngramID:                  uuidFromAny(row["engram_id"]),
		ProjectID:                 stringFromAny(row["project_id"]),
		Title:                     stringFromAny(row["title"]),
		Abstract:                  stringFromAny(row["abstract"]),
		CreatedAt:                 timeFromAny(row["created_at"]),
		Tags:                      stringSliceFromAny(row["tags"]),
		Keywords:                  stringSliceFromAny(row["keywords"]),
		VisibilityScope:           visibilityFromAny(row["visibility_scope"]),
		AccessCount:               intFromAny(row["access_count"]),
		FreshnessScore:            float64FromAny(row["freshness_score"]),
		FeedbackCount:             intFromAny(row["feedback_count"]),
		ContradictionCount:        intFromAny(row["contradiction_count"]),
		SourceSessionQualityScore: float64FromAny(row["source_session_quality_score"]),
		Distance:                  float64FromAny(row["distance"]),
	}
	if ownerUserID, ok := row["owner_user_id"].(*uuid.UUID); ok {
		result.OwnerUserID = ownerUserID
	}
	return result
}

func visibilityOrDefault(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "private"
	}
	return *value
}

func visibilityFromAny(value any) string {
	if typed, ok := value.(string); ok && strings.TrimSpace(typed) != "" {
		return typed
	}
	return "private"
}

func pgxPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func uuidFromAny(value any) uuid.UUID {
	if typed, ok := value.(uuid.UUID); ok {
		return typed
	}
	return uuid.Nil
}
