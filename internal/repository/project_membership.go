package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProjectMemberListInput captures list-member filters.
type ProjectMemberListInput struct {
	ProjectID      string
	IncludeRevoked bool
	Limit          int
	Offset         int
}

// ProjectMemberAddInput captures member add/upsert fields.
type ProjectMemberAddInput struct {
	ProjectID     string
	UserID        uuid.UUID
	Role          models.ProjectMemberRole
	AddedByUserID uuid.UUID
}

// ProjectMemberUpdateInput captures member update fields.
type ProjectMemberUpdateInput struct {
	ProjectID       string
	UserID          uuid.UUID
	Role            models.ProjectMemberRole
	UpdatedByUserID uuid.UUID
}

// ProjectMemberRemoveInput captures member revoke fields.
type ProjectMemberRemoveInput struct {
	ProjectID        string
	UserID           uuid.UUID
	RevokedByUserID  uuid.UUID
	RevocationReason string
}

// ProjectMemberGetInput captures project/user member lookup controls.
type ProjectMemberGetInput struct {
	ProjectID      string
	UserID         uuid.UUID
	IncludeRevoked bool
}

// ActorProjectRoleInput captures actor-scoped membership role lookup inputs.
type ActorProjectRoleInput struct {
	ProjectID   string
	ActorUserID uuid.UUID
}

// ListProjectMembers returns project membership rows filtered by active/revoked status.
func ListProjectMembers(
	ctx context.Context,
	db Queryer,
	input ProjectMemberListInput,
) ([]models.ProjectMemberRecord, error) {
	query := `
		SELECT
			project_id,
			user_id,
			role,
			added_by_user_id,
			created_at,
			updated_at,
			revoked_at,
			revoked_by_user_id
		FROM project_members
		WHERE project_id = $1
	`
	params := []any{input.ProjectID}
	if !input.IncludeRevoked {
		query += " AND revoked_at IS NULL"
	}
	query += " ORDER BY created_at ASC LIMIT $2 OFFSET $3"
	params = append(params, input.Limit, input.Offset)

	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.ProjectMemberRecord, 0)
	for rows.Next() {
		record, scanErr := scanProjectMemberRecord(rows)
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

// AddOrRestoreProjectMember inserts or reactivates a project member row.
func AddOrRestoreProjectMember(
	ctx context.Context,
	db Queryer,
	input ProjectMemberAddInput,
) (*models.ProjectMemberRecord, error) {
	now := time.Now().UTC()
	row := db.QueryRow(
		ctx,
		`
		INSERT INTO project_members (
			project_id,
			user_id,
			role,
			added_by_user_id,
			created_at,
			updated_at,
			revoked_at,
			revoked_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $5, NULL, NULL)
		ON CONFLICT (project_id, user_id) DO UPDATE
		SET
			role = EXCLUDED.role,
			added_by_user_id = EXCLUDED.added_by_user_id,
			updated_at = EXCLUDED.updated_at,
			revoked_at = NULL,
			revoked_by_user_id = NULL
		RETURNING
			project_id,
			user_id,
			role,
			added_by_user_id,
			created_at,
			updated_at,
			revoked_at,
			revoked_by_user_id
		`,
		input.ProjectID,
		input.UserID,
		string(input.Role),
		input.AddedByUserID,
		now,
	)
	record, err := scanProjectMemberRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateProjectMemberRole updates an active membership role.
func UpdateProjectMemberRole(
	ctx context.Context,
	db Queryer,
	input ProjectMemberUpdateInput,
) (*models.ProjectMemberRecord, error) {
	now := time.Now().UTC()
	row := db.QueryRow(
		ctx,
		`
		UPDATE project_members
		SET
			role = $1,
			updated_at = $2
		WHERE
			project_id = $3
			AND user_id = $4
			AND revoked_at IS NULL
		RETURNING
			project_id,
			user_id,
			role,
			added_by_user_id,
			created_at,
			updated_at,
			revoked_at,
			revoked_by_user_id
		`,
		string(input.Role),
		now,
		input.ProjectID,
		input.UserID,
	)
	record, err := scanProjectMemberRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// RemoveProjectMember revokes an active membership row.
func RemoveProjectMember(
	ctx context.Context,
	db Queryer,
	input ProjectMemberRemoveInput,
) (bool, error) {
	now := time.Now().UTC()
	row := db.QueryRow(
		ctx,
		`
		UPDATE project_members
		SET
			updated_at = $1,
			revoked_at = $1,
			revoked_by_user_id = $2
		WHERE
			project_id = $3
			AND user_id = $4
			AND revoked_at IS NULL
		RETURNING project_id
		`,
		now,
		input.RevokedByUserID,
		input.ProjectID,
		input.UserID,
	)
	var projectID string
	err := row.Scan(&projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetProjectMember returns a member row by project/user when visible under includeRevoked.
func GetProjectMember(
	ctx context.Context,
	db Queryer,
	input ProjectMemberGetInput,
) (*models.ProjectMemberRecord, error) {
	query := `
		SELECT
			project_id,
			user_id,
			role,
			added_by_user_id,
			created_at,
			updated_at,
			revoked_at,
			revoked_by_user_id
		FROM project_members
		WHERE
			project_id = $1
			AND user_id = $2
	`
	if !input.IncludeRevoked {
		query += " AND revoked_at IS NULL"
	}
	query += " LIMIT 1"
	row := db.QueryRow(ctx, query, input.ProjectID, input.UserID)
	record, err := scanProjectMemberRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ResolveActorProjectRole returns actor membership role for a project including canonical owner override.
func ResolveActorProjectRole(
	ctx context.Context,
	db Queryer,
	input ActorProjectRoleInput,
) (*models.ProjectMemberRole, error) {
	row := db.QueryRow(
		ctx,
		`
		SELECT
			CASE
				WHEN p.owner_user_id = $2 THEN 'owner'
				ELSE pm.role
			END AS role
		FROM projects p
		LEFT JOIN project_members pm
		  ON pm.project_id = p.project_id
		 AND pm.user_id = $2
		 AND pm.revoked_at IS NULL
		WHERE
			p.project_id = $1
			AND (p.owner_user_id = $2 OR pm.user_id IS NOT NULL)
		LIMIT 1
		`,
		input.ProjectID,
		input.ActorUserID,
	)
	var roleRaw string
	if err := row.Scan(&roleRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	role, err := models.ParseProjectMemberRole(strings.TrimSpace(roleRaw))
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func scanProjectMemberRecord(row interface {
	Scan(dest ...any) error
}) (models.ProjectMemberRecord, error) {
	record := models.ProjectMemberRecord{}
	var roleRaw string
	err := row.Scan(
		&record.ProjectID,
		&record.UserID,
		&roleRaw,
		&record.AddedByUserID,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.RevokedAt,
		&record.RevokedByUserID,
	)
	if err != nil {
		return models.ProjectMemberRecord{}, err
	}
	role, err := models.ParseProjectMemberRole(strings.TrimSpace(roleRaw))
	if err != nil {
		return models.ProjectMemberRecord{}, err
	}
	record.Role = role
	return record, nil
}
