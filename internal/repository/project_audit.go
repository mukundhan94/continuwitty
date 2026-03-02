package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProjectAuditEventCreateInput captures audit-event insert values.
type ProjectAuditEventCreateInput struct {
	EventID        uuid.UUID
	ProjectID      string
	ActorUserID    *uuid.UUID
	EventType      string
	TargetType     string
	TargetUserID   *uuid.UUID
	TargetEngramID *uuid.UUID
	Metadata       map[string]any
	CreatedAt      *time.Time
}

// ProjectAuditEventListInput captures audit list filters.
type ProjectAuditEventListInput struct {
	ProjectID string
	Limit     int
	Offset    int
}

// CreateProjectAuditEvent inserts an audit event.
func CreateProjectAuditEvent(
	ctx context.Context,
	db Queryer,
	input ProjectAuditEventCreateInput,
) (*models.ProjectAuditEventRecord, error) {
	eventID := input.EventID
	if eventID == uuid.Nil {
		eventID = uuid.New()
	}
	createdAt := time.Now().UTC()
	if input.CreatedAt != nil {
		createdAt = input.CreatedAt.UTC()
	}
	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(
		ctx,
		`
		INSERT INTO project_audit_events (
			event_id,
			project_id,
			actor_user_id,
			event_type,
			target_type,
			target_user_id,
			target_engram_id,
			metadata,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
		RETURNING
			event_id,
			project_id,
			actor_user_id,
			event_type,
			target_type,
			target_user_id,
			target_engram_id,
			metadata,
			created_at
		`,
		eventID,
		input.ProjectID,
		input.ActorUserID,
		input.EventType,
		input.TargetType,
		input.TargetUserID,
		input.TargetEngramID,
		string(metadataJSON),
		createdAt,
	)
	record, err := scanProjectAuditEventRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListProjectAuditEvents returns project audit events newest-first.
func ListProjectAuditEvents(
	ctx context.Context,
	db Queryer,
	input ProjectAuditEventListInput,
) ([]models.ProjectAuditEventRecord, error) {
	rows, err := db.Query(
		ctx,
		`
		SELECT
			event_id,
			project_id,
			actor_user_id,
			event_type,
			target_type,
			target_user_id,
			target_engram_id,
			metadata,
			created_at
		FROM project_audit_events
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
		`,
		input.ProjectID,
		input.Limit,
		input.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.ProjectAuditEventRecord, 0)
	for rows.Next() {
		record, scanErr := scanProjectAuditEventRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func scanProjectAuditEventRecord(row interface {
	Scan(dest ...any) error
}) (models.ProjectAuditEventRecord, error) {
	record := models.ProjectAuditEventRecord{}
	var metadataRaw []byte
	err := row.Scan(
		&record.EventID,
		&record.ProjectID,
		&record.ActorUserID,
		&record.EventType,
		&record.TargetType,
		&record.TargetUserID,
		&record.TargetEngramID,
		&metadataRaw,
		&record.CreatedAt,
	)
	if err != nil {
		return models.ProjectAuditEventRecord{}, err
	}
	record.Metadata = map[string]any{}
	if len(metadataRaw) > 0 {
		if unmarshalErr := json.Unmarshal(metadataRaw, &record.Metadata); unmarshalErr != nil {
			return models.ProjectAuditEventRecord{}, unmarshalErr
		}
	}
	return record, nil
}
