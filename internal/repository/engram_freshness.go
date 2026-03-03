package repository

import (
	"context"
	"errors"
	"strings"
	"time"
)

const defaultEngramFreshnessHalfLifeDays = 70.0

var (
	errEngramFreshnessHalfLifeDaysInvalid = errors.New("half_life_days must be greater than zero")
	nowEngramFreshnessUTC                 = func() time.Time { return time.Now().UTC() }
)

// EngramFreshnessRefreshInput captures freshness maintenance refresh options.
type EngramFreshnessRefreshInput struct {
	ProjectID     *string
	HalfLifeDays  float64
	ReferenceTime time.Time
	UpdatedCount  int
}

// RefreshEngramFreshnessScores recomputes freshness scores using an exponential half-life decay.
func RefreshEngramFreshnessScores(
	ctx context.Context,
	db Queryer,
	input EngramFreshnessRefreshInput,
) (EngramFreshnessRefreshInput, error) {
	normalized, err := normalizeEngramFreshnessRefreshInput(input)
	if err != nil {
		return EngramFreshnessRefreshInput{}, err
	}
	updatedCount := 0
	if err := db.QueryRow(
		ctx,
		refreshEngramFreshnessScoresSQL(),
		normalized.ReferenceTime,
		normalized.HalfLifeDays,
		optionalStringValue(normalized.ProjectID),
	).Scan(&updatedCount); err != nil {
		return EngramFreshnessRefreshInput{}, err
	}
	normalized.UpdatedCount = updatedCount
	return normalized, nil
}

func normalizeEngramFreshnessRefreshInput(
	input EngramFreshnessRefreshInput,
) (EngramFreshnessRefreshInput, error) {
	projectID := normalizeOptionalEngramFreshnessProjectID(input.ProjectID)
	if input.HalfLifeDays == 0 {
		input.HalfLifeDays = defaultEngramFreshnessHalfLifeDays
	}
	if input.HalfLifeDays < 0 {
		return EngramFreshnessRefreshInput{}, errEngramFreshnessHalfLifeDaysInvalid
	}
	if input.ReferenceTime.IsZero() {
		input.ReferenceTime = nowEngramFreshnessUTC()
	}
	input.ProjectID = projectID
	return input, nil
}

func normalizeOptionalEngramFreshnessProjectID(projectID *string) *string {
	if projectID == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*projectID)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func refreshEngramFreshnessScoresSQL() string {
	return `
		WITH updated AS (
			UPDATE engrams e
			SET
				freshness_score = CASE
					WHEN $1 <= COALESCE(e.last_accessed_at, e.created_at) THEN 1.0
					ELSE EXP(
						(-LN(2.0)) *
						(EXTRACT(EPOCH FROM ($1 - COALESCE(e.last_accessed_at, e.created_at))) / 86400.0) /
						$2
					)
				END,
				freshness_last_computed_at = $1
			WHERE
				e.deleted_at IS NULL
				AND ($3::text IS NULL OR e.project_id = $3)
			RETURNING e.engram_id
		)
		SELECT COUNT(*) FROM updated;
	`
}
