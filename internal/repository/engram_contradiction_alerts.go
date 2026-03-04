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
	"github.com/jackc/pgx/v5"
)

var (
	errContradictionAlertResolveStatusInvalid = errors.New("status must be resolved or dismissed")
	newContradictionAlertUUID                 = uuid.New
	nowContradictionAlertUTC                  = func() time.Time { return time.Now().UTC() }
)

// ContradictionAlertRefreshInput captures contradiction-alert refresh inputs and outputs.
type ContradictionAlertRefreshInput struct {
	ProjectID    *string
	DetectedAt   time.Time
	UpdatedCount int
}

// ContradictionAlertListInput captures contradiction-alert list filters.
type ContradictionAlertListInput struct {
	ProjectID *string
	Status    *models.ContradictionAlertStatus
	Limit     int
	Offset    int
}

// ContradictionAlertResolveInput captures contradiction-alert resolve workflow inputs.
type ContradictionAlertResolveInput struct {
	AlertID    uuid.UUID
	ProjectID  *string
	Status     models.ContradictionAlertStatus
	ResolvedBy uuid.UUID
	ResolvedAt time.Time
}

type contradictionAlertGroup struct {
	ProjectID            string
	SourceEngramID       uuid.UUID
	TargetEngramID       uuid.UUID
	ContradictionLinkIDs []uuid.UUID
	ConfidenceScore      float64
}

// RefreshContradictionAlerts refreshes deterministic contradiction alerts from active contradiction links.
func RefreshContradictionAlerts(
	ctx context.Context,
	db Queryer,
	input ContradictionAlertRefreshInput,
) (ContradictionAlertRefreshInput, error) {
	normalized := normalizeContradictionAlertRefreshInput(input)
	groups, err := listContradictionAlertGroups(ctx, db, normalized.ProjectID)
	if err != nil {
		return ContradictionAlertRefreshInput{}, err
	}
	updatedCount, err := upsertContradictionAlertGroups(ctx, db, groups, normalized.DetectedAt)
	if err != nil {
		return ContradictionAlertRefreshInput{}, err
	}
	normalized.UpdatedCount = updatedCount
	return normalized, nil
}

// ListContradictionAlerts returns contradiction alerts with optional status/project filters.
func ListContradictionAlerts(
	ctx context.Context,
	db Queryer,
	input ContradictionAlertListInput,
) ([]models.EngramContradictionAlert, error) {
	query, params := buildContradictionAlertListQuery(input)
	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContradictionAlertRows(rows)
}

// ResolveContradictionAlert marks one contradiction alert as resolved or dismissed.
func ResolveContradictionAlert(
	ctx context.Context,
	db Queryer,
	input ContradictionAlertResolveInput,
) (*models.EngramContradictionAlert, error) {
	normalized, err := normalizeContradictionAlertResolveInput(input)
	if err != nil {
		return nil, err
	}
	row := db.QueryRow(
		ctx,
		`
		UPDATE engram_contradiction_alerts
		SET
			status = $2,
			resolved_at = $3,
			resolved_by = $4,
			updated_at = $3
		WHERE
			alert_id = $1
			AND status = 'open'
			AND ($5::text IS NULL OR project_id = $5)
		RETURNING
			alert_id,
			project_id,
			source_engram_id,
			target_engram_id,
			contradiction_link_ids,
			reason,
			alert_hash,
			confidence_score,
			status,
			detected_at,
			updated_at,
			resolved_at,
			resolved_by
		`,
		normalized.AlertID,
		string(normalized.Status),
		normalized.ResolvedAt,
		normalized.ResolvedBy,
		optionalStringValue(normalized.ProjectID),
	)
	record, err := scanContradictionAlert(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func normalizeContradictionAlertRefreshInput(
	input ContradictionAlertRefreshInput,
) ContradictionAlertRefreshInput {
	input.ProjectID = normalizeOptionalConsolidationProjectID(input.ProjectID)
	if input.DetectedAt.IsZero() {
		input.DetectedAt = nowContradictionAlertUTC()
	}
	return input
}

func normalizeContradictionAlertResolveInput(
	input ContradictionAlertResolveInput,
) (ContradictionAlertResolveInput, error) {
	switch input.Status {
	case models.ContradictionAlertStatusResolved, models.ContradictionAlertStatusDismissed:
	default:
		return ContradictionAlertResolveInput{}, errContradictionAlertResolveStatusInvalid
	}
	input.ProjectID = normalizeOptionalConsolidationProjectID(input.ProjectID)
	if input.ResolvedAt.IsZero() {
		input.ResolvedAt = nowContradictionAlertUTC()
	}
	return input, nil
}

func listContradictionAlertGroups(
	ctx context.Context,
	db Queryer,
	projectID *string,
) ([]contradictionAlertGroup, error) {
	rows, err := db.Query(
		ctx,
		`
		SELECT
			project_id,
			CASE
				WHEN source_engram_id::text <= target_engram_id::text THEN source_engram_id
				ELSE target_engram_id
			END AS canonical_source_engram_id,
			CASE
				WHEN source_engram_id::text <= target_engram_id::text THEN target_engram_id
				ELSE source_engram_id
			END AS canonical_target_engram_id,
			ARRAY_AGG(link_id ORDER BY updated_at DESC) AS contradiction_link_ids,
			MAX(confidence) AS confidence_score
		FROM engram_links
		WHERE
			status = 'active'
			AND relation_type = 'contradicts'
			AND ($1::text IS NULL OR project_id = $1)
		GROUP BY
			project_id,
			CASE
				WHEN source_engram_id::text <= target_engram_id::text THEN source_engram_id
				ELSE target_engram_id
			END,
			CASE
				WHEN source_engram_id::text <= target_engram_id::text THEN target_engram_id
				ELSE source_engram_id
			END
		`,
		optionalStringValue(projectID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]contradictionAlertGroup, 0)
	for rows.Next() {
		group := contradictionAlertGroup{}
		if err := rows.Scan(
			&group.ProjectID,
			&group.SourceEngramID,
			&group.TargetEngramID,
			&group.ContradictionLinkIDs,
			&group.ConfidenceScore,
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

func upsertContradictionAlertGroups(
	ctx context.Context,
	db Queryer,
	groups []contradictionAlertGroup,
	detectedAt time.Time,
) (int, error) {
	updatedCount := 0
	for _, group := range groups {
		applied, err := upsertContradictionAlertGroup(ctx, db, group, detectedAt)
		if err != nil {
			return 0, err
		}
		if applied {
			updatedCount++
		}
	}
	return updatedCount, nil
}

func upsertContradictionAlertGroup(
	ctx context.Context,
	db Queryer,
	group contradictionAlertGroup,
	detectedAt time.Time,
) (bool, error) {
	alertHash := contradictionAlertHash(group.ProjectID, group.SourceEngramID, group.TargetEngramID)
	applied := false
	err := db.QueryRow(
		ctx,
		`
		WITH upserted AS (
			INSERT INTO engram_contradiction_alerts (
				alert_id,
				project_id,
				source_engram_id,
				target_engram_id,
				contradiction_link_ids,
				reason,
				alert_hash,
				confidence_score,
				status,
				detected_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'open', $9, $9)
			ON CONFLICT (alert_hash) DO UPDATE
			SET
				contradiction_link_ids = EXCLUDED.contradiction_link_ids,
				reason = EXCLUDED.reason,
				confidence_score = EXCLUDED.confidence_score,
				status = 'open',
				detected_at = EXCLUDED.detected_at,
				updated_at = EXCLUDED.updated_at,
				resolved_at = NULL,
				resolved_by = NULL
			RETURNING alert_id
		)
		SELECT EXISTS(SELECT 1 FROM upserted);
		`,
		newContradictionAlertUUID(),
		group.ProjectID,
		group.SourceEngramID,
		group.TargetEngramID,
		group.ContradictionLinkIDs,
		"Active contradiction link relationship detected.",
		alertHash,
		clamp01(group.ConfidenceScore),
		detectedAt,
	).Scan(&applied)
	if err != nil {
		return false, err
	}
	return applied, nil
}

func buildContradictionAlertListQuery(input ContradictionAlertListInput) (string, []any) {
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
			alert_id,
			project_id,
			source_engram_id,
			target_engram_id,
			contradiction_link_ids,
			reason,
			alert_hash,
			confidence_score,
			status,
			detected_at,
			updated_at,
			resolved_at,
			resolved_by
		FROM engram_contradiction_alerts
		WHERE ` + strings.Join(whereClauses, " AND ") + `
		ORDER BY detected_at DESC, alert_id ASC
		LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder
	return query, params
}

func scanContradictionAlertRows(
	rows interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	},
) ([]models.EngramContradictionAlert, error) {
	results := make([]models.EngramContradictionAlert, 0)
	for rows.Next() {
		record, err := scanContradictionAlert(rows)
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

func scanContradictionAlert(
	row interface {
		Scan(dest ...any) error
	},
) (models.EngramContradictionAlert, error) {
	var (
		record    models.EngramContradictionAlert
		rawStatus string
	)
	if err := row.Scan(
		&record.AlertID,
		&record.ProjectID,
		&record.SourceEngramID,
		&record.TargetEngramID,
		&record.ContradictionLinkIDs,
		&record.Reason,
		&record.AlertHash,
		&record.ConfidenceScore,
		&rawStatus,
		&record.DetectedAt,
		&record.UpdatedAt,
		&record.ResolvedAt,
		&record.ResolvedBy,
	); err != nil {
		return models.EngramContradictionAlert{}, err
	}
	parsedStatus, err := models.ParseContradictionAlertStatus(rawStatus)
	if err != nil {
		return models.EngramContradictionAlert{}, err
	}
	record.Status = parsedStatus
	return record, nil
}

func contradictionAlertHash(projectID string, sourceEngramID uuid.UUID, targetEngramID uuid.UUID) string {
	key := strings.Join(
		[]string{projectID, sourceEngramID.String(), targetEngramID.String()},
		"|",
	)
	checksum := sha1.Sum([]byte(key))
	return hex.EncodeToString(checksum[:])
}
