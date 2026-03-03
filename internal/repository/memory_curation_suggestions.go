package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const defaultMemoryCurationSuggestionLimit = 100

var (
	errMemoryCurationProjectIDRequired    = errors.New("project_id is required")
	errMemoryCurationReasonRequired       = errors.New("reason is required")
	errMemoryCurationConfidenceOutOfRange = errors.New("confidence_score must be between 0 and 1")
	errMemoryCurationActionStatusInvalid  = errors.New("status must be accepted, rejected, or applied")
	newMemoryCurationSuggestionUUID       = uuid.New
	nowMemoryCurationSuggestionUTC        = func() time.Time { return time.Now().UTC() }
)

// MemoryCurationSuggestionCreateInput captures suggestion persistence fields.
type MemoryCurationSuggestionCreateInput struct {
	ProjectID       string
	SessionID       *uuid.UUID
	SuggestionType  models.MemoryCurationSuggestionType
	Reason          string
	Recommendation  string
	PayloadJSON     map[string]any
	ConfidenceScore float64
	Status          models.MemoryCurationSuggestionStatus
	SuggestedAt     time.Time
}

// MemoryCurationSuggestionListInput captures list filtering controls.
type MemoryCurationSuggestionListInput struct {
	ProjectID      *string
	SessionID      *uuid.UUID
	SuggestionType *models.MemoryCurationSuggestionType
	Status         *models.MemoryCurationSuggestionStatus
	Limit          int
	Offset         int
}

// MemoryCurationSuggestionActionInput captures suggestion action state transitions.
type MemoryCurationSuggestionActionInput struct {
	SuggestionID uuid.UUID
	ProjectID    *string
	Status       models.MemoryCurationSuggestionStatus
	ActorUserID  uuid.UUID
	ActionedAt   time.Time
}

// MemoryCurationSuggestionResetInput captures reset filters for suggested curation records.
type MemoryCurationSuggestionResetInput struct {
	ProjectID      *string
	SuggestionType models.MemoryCurationSuggestionType
}

// CreateMemoryCurationSuggestion creates one persisted memory curation recommendation.
func CreateMemoryCurationSuggestion(
	ctx context.Context,
	db Queryer,
	input MemoryCurationSuggestionCreateInput,
) (*models.MemoryCurationSuggestion, error) {
	normalized, err := normalizeMemoryCurationSuggestionCreateInput(input)
	if err != nil {
		return nil, err
	}
	row := db.QueryRow(
		ctx,
		`
		INSERT INTO memory_curation_suggestions (
			suggestion_id,
			project_id,
			session_id,
			suggestion_type,
			reason,
			recommendation,
			payload_json,
			confidence_score,
			status,
			suggested_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $10)
		RETURNING
			suggestion_id,
			project_id,
			session_id,
			suggestion_type,
			reason,
			recommendation,
			payload_json,
			confidence_score,
			status,
			suggested_at,
			updated_at,
			actioned_at,
			action_taken_by
		`,
		newMemoryCurationSuggestionUUID(),
		normalized.ProjectID,
		optionalUUIDValue(normalized.SessionID),
		string(normalized.SuggestionType),
		normalized.Reason,
		normalized.Recommendation,
		marshalJSONMap(normalized.PayloadJSON),
		normalized.ConfidenceScore,
		string(normalized.Status),
		normalized.SuggestedAt,
	)
	record, err := scanMemoryCurationSuggestion(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListMemoryCurationSuggestions lists persisted memory curation suggestions with optional filters.
func ListMemoryCurationSuggestions(
	ctx context.Context,
	db Queryer,
	input MemoryCurationSuggestionListInput,
) ([]models.MemoryCurationSuggestion, error) {
	query, params := buildMemoryCurationSuggestionListQuery(input)
	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryCurationSuggestionRows(rows)
}

// GetMemoryCurationSuggestion gets one suggestion by id with optional project scoping.
func GetMemoryCurationSuggestion(
	ctx context.Context,
	db Queryer,
	suggestionID uuid.UUID,
	projectID *string,
) (*models.MemoryCurationSuggestion, error) {
	row := db.QueryRow(
		ctx,
		`
		SELECT
			suggestion_id,
			project_id,
			session_id,
			suggestion_type,
			reason,
			recommendation,
			payload_json,
			confidence_score,
			status,
			suggested_at,
			updated_at,
			actioned_at,
			action_taken_by
		FROM memory_curation_suggestions
		WHERE suggestion_id = $1
			AND ($2::text IS NULL OR project_id = $2)
		`,
		suggestionID,
		optionalStringValue(normalizeOptionalConsolidationProjectID(projectID)),
	)
	record, err := scanMemoryCurationSuggestion(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ApplyMemoryCurationSuggestionAction updates one suggestion to an actioned state.
func ApplyMemoryCurationSuggestionAction(
	ctx context.Context,
	db Queryer,
	input MemoryCurationSuggestionActionInput,
) (*models.MemoryCurationSuggestion, error) {
	normalized, err := normalizeMemoryCurationSuggestionActionInput(input)
	if err != nil {
		return nil, err
	}
	row := db.QueryRow(
		ctx,
		`
		UPDATE memory_curation_suggestions
		SET
			status = $2,
			actioned_at = $3,
			action_taken_by = $4,
			updated_at = $3
		WHERE suggestion_id = $1
			AND ($5::text IS NULL OR project_id = $5)
		RETURNING
			suggestion_id,
			project_id,
			session_id,
			suggestion_type,
			reason,
			recommendation,
			payload_json,
			confidence_score,
			status,
			suggested_at,
			updated_at,
			actioned_at,
			action_taken_by
		`,
		normalized.SuggestionID,
		string(normalized.Status),
		normalized.ActionedAt,
		normalized.ActorUserID,
		optionalStringValue(normalized.ProjectID),
	)
	record, err := scanMemoryCurationSuggestion(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ResetSuggestedMemoryCurationSuggestions deletes suggested records for one curation type and optional project scope.
func ResetSuggestedMemoryCurationSuggestions(
	ctx context.Context,
	db Queryer,
	input MemoryCurationSuggestionResetInput,
) (int, error) {
	suggestionType, err := models.ParseMemoryCurationSuggestionType(string(input.SuggestionType))
	if err != nil {
		return 0, err
	}
	projectID := normalizeOptionalConsolidationProjectID(input.ProjectID)
	rows, err := db.Query(
		ctx,
		`
		DELETE FROM memory_curation_suggestions
		WHERE status = 'suggested'
			AND suggestion_type = $1
			AND ($2::text IS NULL OR project_id = $2)
		RETURNING suggestion_id
		`,
		string(suggestionType),
		optionalStringValue(projectID),
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	removed := 0
	for rows.Next() {
		var suggestionID uuid.UUID
		if err := rows.Scan(&suggestionID); err != nil {
			return 0, err
		}
		removed++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return removed, nil
}

func buildMemoryCurationSuggestionListQuery(input MemoryCurationSuggestionListInput) (string, []any) {
	whereClauses := []string{"1 = 1"}
	params := make([]any, 0, 6)
	addParam := func(value any) string {
		params = append(params, value)
		return pgxPlaceholder(len(params))
	}

	if projectID := normalizeOptionalConsolidationProjectID(input.ProjectID); projectID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("project_id = %s", addParam(*projectID)))
	}
	if input.SessionID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("session_id = %s", addParam(*input.SessionID)))
	}
	if input.SuggestionType != nil {
		whereClauses = append(
			whereClauses,
			fmt.Sprintf("suggestion_type = %s", addParam(string(*input.SuggestionType))),
		)
	}
	if input.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = %s", addParam(string(*input.Status))))
	}
	limit := input.Limit
	if limit <= 0 {
		limit = defaultMemoryCurationSuggestionLimit
	}
	offset := max(input.Offset, 0)
	limitPlaceholder := addParam(limit)
	offsetPlaceholder := addParam(offset)

	query := `
		SELECT
			suggestion_id,
			project_id,
			session_id,
			suggestion_type,
			reason,
			recommendation,
			payload_json,
			confidence_score,
			status,
			suggested_at,
			updated_at,
			actioned_at,
			action_taken_by
		FROM memory_curation_suggestions
		WHERE ` + strings.Join(whereClauses, " AND ") + `
		ORDER BY suggested_at DESC, suggestion_id ASC
		LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder
	return query, params
}

type memoryCurationSuggestionRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanMemoryCurationSuggestionRows(rows memoryCurationSuggestionRows) ([]models.MemoryCurationSuggestion, error) {
	results := make([]models.MemoryCurationSuggestion, 0)
	for rows.Next() {
		record, err := scanMemoryCurationSuggestion(rows)
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

func scanMemoryCurationSuggestion(
	row interface {
		Scan(dest ...any) error
	},
) (models.MemoryCurationSuggestion, error) {
	var (
		record            models.MemoryCurationSuggestion
		rawSuggestionType string
		rawStatus         string
		rawPayload        []byte
	)
	if err := row.Scan(
		&record.SuggestionID,
		&record.ProjectID,
		&record.SessionID,
		&rawSuggestionType,
		&record.Reason,
		&record.Recommendation,
		&rawPayload,
		&record.ConfidenceScore,
		&rawStatus,
		&record.SuggestedAt,
		&record.UpdatedAt,
		&record.ActionedAt,
		&record.ActionTakenBy,
	); err != nil {
		return models.MemoryCurationSuggestion{}, err
	}
	suggestionType, err := models.ParseMemoryCurationSuggestionType(rawSuggestionType)
	if err != nil {
		return models.MemoryCurationSuggestion{}, err
	}
	status, err := models.ParseMemoryCurationSuggestionStatus(rawStatus)
	if err != nil {
		return models.MemoryCurationSuggestion{}, err
	}
	record.SuggestionType = suggestionType
	record.Status = status
	record.PayloadJSON = unmarshalJSONMap(rawPayload)
	return record, nil
}

func normalizeMemoryCurationSuggestionCreateInput(
	input MemoryCurationSuggestionCreateInput,
) (MemoryCurationSuggestionCreateInput, error) {
	normalized, err := normalizeMemoryCurationSuggestionProject(input)
	if err != nil {
		return MemoryCurationSuggestionCreateInput{}, err
	}
	normalized, err = normalizeMemoryCurationSuggestionType(normalized)
	if err != nil {
		return MemoryCurationSuggestionCreateInput{}, err
	}
	normalized, err = normalizeMemoryCurationSuggestionReason(normalized)
	if err != nil {
		return MemoryCurationSuggestionCreateInput{}, err
	}
	if memoryCurationConfidenceOutOfRange(normalized.ConfidenceScore) {
		return MemoryCurationSuggestionCreateInput{}, errMemoryCurationConfidenceOutOfRange
	}
	normalized, err = normalizeMemoryCurationSuggestionStatus(normalized)
	if err != nil {
		return MemoryCurationSuggestionCreateInput{}, err
	}
	normalized.SuggestedAt = normalizeMemoryCurationSuggestedAt(normalized.SuggestedAt)
	normalized.PayloadJSON = normalizeMemoryCurationPayload(normalized.PayloadJSON)
	return normalized, nil
}

func normalizeMemoryCurationSuggestionActionInput(
	input MemoryCurationSuggestionActionInput,
) (MemoryCurationSuggestionActionInput, error) {
	switch input.Status {
	case models.MemoryCurationSuggestionStatusAccepted,
		models.MemoryCurationSuggestionStatusRejected,
		models.MemoryCurationSuggestionStatusApplied:
	default:
		return MemoryCurationSuggestionActionInput{}, errMemoryCurationActionStatusInvalid
	}
	input.ProjectID = normalizeOptionalConsolidationProjectID(input.ProjectID)
	if input.ActionedAt.IsZero() {
		input.ActionedAt = nowMemoryCurationSuggestionUTC()
	}
	return input, nil
}

func normalizeMemoryCurationSuggestionProject(
	input MemoryCurationSuggestionCreateInput,
) (MemoryCurationSuggestionCreateInput, error) {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	if input.ProjectID == "" {
		return MemoryCurationSuggestionCreateInput{}, errMemoryCurationProjectIDRequired
	}
	return input, nil
}

func normalizeMemoryCurationSuggestionType(
	input MemoryCurationSuggestionCreateInput,
) (MemoryCurationSuggestionCreateInput, error) {
	normalizedType, err := models.ParseMemoryCurationSuggestionType(string(input.SuggestionType))
	if err != nil {
		return MemoryCurationSuggestionCreateInput{}, err
	}
	input.SuggestionType = normalizedType
	return input, nil
}

func normalizeMemoryCurationSuggestionReason(
	input MemoryCurationSuggestionCreateInput,
) (MemoryCurationSuggestionCreateInput, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" {
		return MemoryCurationSuggestionCreateInput{}, errMemoryCurationReasonRequired
	}
	input.Recommendation = strings.TrimSpace(input.Recommendation)
	return input, nil
}

func normalizeMemoryCurationSuggestionStatus(
	input MemoryCurationSuggestionCreateInput,
) (MemoryCurationSuggestionCreateInput, error) {
	if input.Status == "" {
		input.Status = models.MemoryCurationSuggestionStatusSuggested
	}
	status, err := models.ParseMemoryCurationSuggestionStatus(string(input.Status))
	if err != nil {
		return MemoryCurationSuggestionCreateInput{}, err
	}
	input.Status = status
	return input, nil
}

func memoryCurationConfidenceOutOfRange(value float64) bool {
	return value < 0 || value > 1
}

func normalizeMemoryCurationSuggestedAt(value time.Time) time.Time {
	if !value.IsZero() {
		return value
	}
	return nowMemoryCurationSuggestionUTC()
}

func normalizeMemoryCurationPayload(payload map[string]any) map[string]any {
	if payload != nil {
		return payload
	}
	return map[string]any{}
}

func optionalUUIDValue(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func marshalJSONMap(value map[string]any) string {
	encoded, err := json.Marshal(emptyMapIfNil(value))
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func unmarshalJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]any{}
	}
	if decoded == nil {
		return map[string]any{}
	}
	return decoded
}
