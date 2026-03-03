package repository

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

const defaultConsolidationMinGroupSize = 2

var (
	errConsolidationMinGroupSizeInvalid = errors.New("min_group_size must be at least 2")
	newConsolidationSuggestionUUID      = uuid.New
	nowConsolidationSuggestionUTC       = func() time.Time { return time.Now().UTC() }
)

// ConsolidationSuggestionRefreshInput captures refresh options for consolidation suggestions.
type ConsolidationSuggestionRefreshInput struct {
	ProjectID    *string
	MinGroupSize int
	SuggestedAt  time.Time
	UpdatedCount int
}

// ConsolidationSuggestionListInput captures list filters for consolidation suggestions.
type ConsolidationSuggestionListInput struct {
	ProjectID *string
	Status    *models.ConsolidationSuggestionStatus
	Limit     int
	Offset    int
}

type consolidationDuplicateGroup struct {
	ProjectID       string
	NormalizedTitle string
	SourceEngramIDs []uuid.UUID
}

// RefreshExactDuplicateConsolidationSuggestions generates exact-duplicate title suggestions.
func RefreshExactDuplicateConsolidationSuggestions(
	ctx context.Context,
	db Queryer,
	input ConsolidationSuggestionRefreshInput,
) (ConsolidationSuggestionRefreshInput, error) {
	normalized, err := normalizeConsolidationSuggestionRefreshInput(input)
	if err != nil {
		return ConsolidationSuggestionRefreshInput{}, err
	}
	groups, err := listExactDuplicateTitleGroups(
		ctx,
		db,
		normalized.ProjectID,
		normalized.MinGroupSize,
	)
	if err != nil {
		return ConsolidationSuggestionRefreshInput{}, err
	}
	updatedCount := 0
	for _, group := range groups {
		applied, err := upsertExactDuplicateConsolidationSuggestion(
			ctx,
			db,
			group,
			normalized.SuggestedAt,
		)
		if err != nil {
			return ConsolidationSuggestionRefreshInput{}, err
		}
		if applied {
			updatedCount++
		}
	}
	normalized.UpdatedCount = updatedCount
	return normalized, nil
}

// ListEngramConsolidationSuggestions lists persisted consolidation suggestions with filters.
func ListEngramConsolidationSuggestions(
	ctx context.Context,
	db Queryer,
	input ConsolidationSuggestionListInput,
) ([]models.EngramConsolidationSuggestion, error) {
	query, params := buildConsolidationSuggestionListQuery(input)
	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConsolidationSuggestionRows(rows)
}

func buildConsolidationSuggestionListQuery(input ConsolidationSuggestionListInput) (string, []any) {
	whereClauses := []string{"1 = 1"}
	params := make([]any, 0, 4)
	addParam := func(value any) string {
		params = append(params, value)
		return pgxPlaceholder(len(params))
	}

	if projectID := normalizeOptionalConsolidationProjectID(input.ProjectID); projectID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("project_id = %s", addParam(*projectID)))
	}
	if input.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = %s", addParam(string(*input.Status))))
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := max(input.Offset, 0)
	limitPlaceholder := addParam(limit)
	offsetPlaceholder := addParam(offset)

	query := `
		SELECT
			suggestion_id,
			project_id,
			source_engram_ids,
			consolidation_type,
			reason,
			consolidation_hash,
			confidence_score,
			status,
			suggested_at,
			updated_at,
			actioned_at,
			action_taken_by
		FROM engram_consolidation_suggestions
		WHERE ` + strings.Join(whereClauses, " AND ") + `
		ORDER BY suggested_at DESC, suggestion_id ASC
		LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder
	return query, params
}

type consolidationSuggestionRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanConsolidationSuggestionRows(
	rows consolidationSuggestionRows,
) ([]models.EngramConsolidationSuggestion, error) {
	results := make([]models.EngramConsolidationSuggestion, 0)
	for rows.Next() {
		record, err := scanConsolidationSuggestion(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func scanConsolidationSuggestion(
	row interface {
		Scan(dest ...any) error
	},
) (models.EngramConsolidationSuggestion, error) {
	var (
		record    models.EngramConsolidationSuggestion
		rawStatus string
	)
	if err := row.Scan(
		&record.SuggestionID,
		&record.ProjectID,
		&record.SourceEngramIDs,
		&record.ConsolidationType,
		&record.Reason,
		&record.ConsolidationHash,
		&record.ConfidenceScore,
		&rawStatus,
		&record.SuggestedAt,
		&record.UpdatedAt,
		&record.ActionedAt,
		&record.ActionTakenBy,
	); err != nil {
		return models.EngramConsolidationSuggestion{}, err
	}
	status, err := models.ParseConsolidationSuggestionStatus(rawStatus)
	if err != nil {
		return models.EngramConsolidationSuggestion{}, err
	}
	record.Status = status
	return record, nil
}

func listExactDuplicateTitleGroups(
	ctx context.Context,
	db Queryer,
	projectID *string,
	minGroupSize int,
) ([]consolidationDuplicateGroup, error) {
	rows, err := db.Query(
		ctx,
		`
		SELECT
			project_id,
			LOWER(TRIM(title)) AS normalized_title,
			ARRAY_AGG(engram_id ORDER BY created_at DESC) AS source_engram_ids
		FROM engrams
		WHERE
			deleted_at IS NULL
			AND ($1::text IS NULL OR project_id = $1)
			AND TRIM(title) <> ''
		GROUP BY project_id, LOWER(TRIM(title))
		HAVING COUNT(*) >= $2
		`,
		optionalStringValue(projectID),
		minGroupSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]consolidationDuplicateGroup, 0)
	for rows.Next() {
		group := consolidationDuplicateGroup{}
		if err := rows.Scan(
			&group.ProjectID,
			&group.NormalizedTitle,
			&group.SourceEngramIDs,
		); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}

func normalizeConsolidationSuggestionRefreshInput(
	input ConsolidationSuggestionRefreshInput,
) (ConsolidationSuggestionRefreshInput, error) {
	input.ProjectID = normalizeOptionalConsolidationProjectID(input.ProjectID)
	if input.MinGroupSize == 0 {
		input.MinGroupSize = defaultConsolidationMinGroupSize
	}
	if input.MinGroupSize < defaultConsolidationMinGroupSize {
		return ConsolidationSuggestionRefreshInput{}, errConsolidationMinGroupSizeInvalid
	}
	if input.SuggestedAt.IsZero() {
		input.SuggestedAt = nowConsolidationSuggestionUTC()
	}
	return input, nil
}

func normalizeOptionalConsolidationProjectID(projectID *string) *string {
	if projectID == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*projectID)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func upsertExactDuplicateConsolidationSuggestion(
	ctx context.Context,
	db Queryer,
	group consolidationDuplicateGroup,
	suggestedAt time.Time,
) (bool, error) {
	consolidationHash := suggestionHash(
		group.ProjectID,
		string(models.ConsolidationSuggestionTypeExactDuplicate),
		group.NormalizedTitle,
	)
	confidenceScore := duplicateConfidenceScore(len(group.SourceEngramIDs))
	reason := "Possible exact duplicate title cluster: " + group.NormalizedTitle
	applied := false
	err := db.QueryRow(
		ctx,
		`
		WITH upserted AS (
			INSERT INTO engram_consolidation_suggestions (
				suggestion_id,
				project_id,
				source_engram_ids,
				consolidation_type,
				reason,
				consolidation_hash,
				confidence_score,
				status,
				suggested_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'suggested', $8, $8)
			ON CONFLICT (consolidation_hash) DO UPDATE
			SET
				source_engram_ids = EXCLUDED.source_engram_ids,
				reason = EXCLUDED.reason,
				confidence_score = EXCLUDED.confidence_score,
				updated_at = EXCLUDED.updated_at,
				suggested_at = EXCLUDED.suggested_at
			WHERE engram_consolidation_suggestions.status = 'suggested'
			RETURNING suggestion_id
		)
		SELECT EXISTS(SELECT 1 FROM upserted);
		`,
		newConsolidationSuggestionUUID(),
		group.ProjectID,
		group.SourceEngramIDs,
		string(models.ConsolidationSuggestionTypeExactDuplicate),
		reason,
		consolidationHash,
		confidenceScore,
		suggestedAt,
	).Scan(&applied)
	if err != nil {
		return false, err
	}
	return applied, nil
}

func suggestionHash(projectID string, suggestionType string, normalizedTitle string) string {
	checksum := sha1.Sum([]byte(strings.Join([]string{projectID, suggestionType, normalizedTitle}, "|")))
	return hex.EncodeToString(checksum[:])
}

func duplicateConfidenceScore(groupSize int) float64 {
	base := 0.70 + (float64(max(groupSize, 2)-2) * 0.05)
	return clamp01(min(base, 0.95))
}
