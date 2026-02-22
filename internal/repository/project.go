package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProjectListInput captures list-project filters.
type ProjectListInput struct {
	ActorUserID     uuid.UUID
	ActorRole       string
	IncludeArchived bool
	Limit           int
	Offset          int
}

// ProjectGetInput captures get-project filters.
type ProjectGetInput struct {
	ProjectID       string
	ActorUserID     uuid.UUID
	ActorRole       string
	IncludeArchived bool
}

// ProjectCreateInput captures create-project values.
type ProjectCreateInput struct {
	ProjectID   string
	Name        string
	Description string
	OwnerUserID uuid.UUID
}

// ProjectEnsureInput captures ensure-project dependencies.
type ProjectEnsureInput struct {
	ProjectID   string
	OwnerUserID uuid.UUID
}

// ListProjectsForActor lists projects visible to the actor.
func ListProjectsForActor(
	ctx context.Context,
	db Queryer,
	input ProjectListInput,
) ([]models.ProjectRecord, error) {
	where := []string{"1=1"}
	params := make([]any, 0, 4)
	if input.ActorRole != "admin" {
		where = append(where, fmt.Sprintf("owner_user_id = %s", pgxPlaceholder(len(params)+1)))
		params = append(params, input.ActorUserID)
	}
	if !input.IncludeArchived {
		where = append(where, "is_archived = FALSE")
	}

	sql := fmt.Sprintf(
		`
		SELECT
			project_id,
			name,
			description,
			owner_user_id,
			is_archived,
			created_at,
			updated_at
		FROM projects
		WHERE %s
		ORDER BY created_at ASC
		LIMIT %s OFFSET %s
		`,
		strings.Join(where, " AND "),
		pgxPlaceholder(len(params)+1),
		pgxPlaceholder(len(params)+2),
	)
	params = append(params, input.Limit, input.Offset)

	rows, err := db.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.ProjectRecord, 0)
	for rows.Next() {
		record, scanErr := scanProjectRecord(rows)
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

// GetProjectForActor returns a project if visible to the actor.
func GetProjectForActor(
	ctx context.Context,
	db Queryer,
	input ProjectGetInput,
) (*models.ProjectRecord, error) {
	where := []string{fmt.Sprintf("project_id = %s", pgxPlaceholder(1))}
	params := []any{input.ProjectID}
	if input.ActorRole != "admin" {
		where = append(where, fmt.Sprintf("owner_user_id = %s", pgxPlaceholder(len(params)+1)))
		params = append(params, input.ActorUserID)
	}
	if !input.IncludeArchived {
		where = append(where, "is_archived = FALSE")
	}

	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			SELECT
				project_id,
				name,
				description,
				owner_user_id,
				is_archived,
				created_at,
				updated_at
			FROM projects
			WHERE %s
			LIMIT 1
			`,
			strings.Join(where, " AND "),
		),
		params...,
	)
	record, err := scanProjectRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateProject creates or updates a project row by project ID.
func CreateProject(
	ctx context.Context,
	db Queryer,
	input ProjectCreateInput,
) (*models.ProjectRecord, error) {
	now := time.Now().UTC()
	row := db.QueryRow(
		ctx,
		`
		INSERT INTO projects (
			project_id,
			name,
			description,
			owner_user_id,
			is_archived,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, FALSE, $5, $5)
		ON CONFLICT (project_id) DO UPDATE
		SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at
		RETURNING
			project_id,
			name,
			description,
			owner_user_id,
			is_archived,
			created_at,
			updated_at
		`,
		input.ProjectID,
		input.Name,
		input.Description,
		input.OwnerUserID,
		now,
	)
	record, err := scanProjectRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// EnsureProjectExists resolves the existing project or creates an autocreated project.
func EnsureProjectExists(
	ctx context.Context,
	db Queryer,
	input ProjectEnsureInput,
) (*models.ProjectRecord, error) {
	existing, err := GetProjectForActor(
		ctx,
		db,
		ProjectGetInput{
			ProjectID:       input.ProjectID,
			ActorUserID:     input.OwnerUserID,
			ActorRole:       "admin",
			IncludeArchived: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return CreateProject(
		ctx,
		db,
		ProjectCreateInput{
			ProjectID:   input.ProjectID,
			Name:        input.ProjectID,
			Description: "Autocreated from workflow input.",
			OwnerUserID: input.OwnerUserID,
		},
	)
}

// GetUserDefaultProjectID returns the user's default project ID when set.
func GetUserDefaultProjectID(
	ctx context.Context,
	db Queryer,
	userID uuid.UUID,
) (*string, error) {
	row := db.QueryRow(
		ctx,
		`
		SELECT default_project_id
		FROM users
		WHERE user_id = $1
		LIMIT 1
		`,
		userID,
	)
	var defaultProjectID *string
	err := row.Scan(&defaultProjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return defaultProjectID, nil
}

// SetUserDefaultProjectID sets the user's default project and reports whether a row changed.
func SetUserDefaultProjectID(
	ctx context.Context,
	db Queryer,
	userID uuid.UUID,
	projectID string,
) (bool, error) {
	row := db.QueryRow(
		ctx,
		`
		UPDATE users
		SET default_project_id = $1
		WHERE user_id = $2
		RETURNING user_id
		`,
		projectID,
		userID,
	)
	var updatedUserID uuid.UUID
	err := row.Scan(&updatedUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func scanProjectRecord(row interface {
	Scan(dest ...any) error
}) (models.ProjectRecord, error) {
	record := models.ProjectRecord{}
	err := row.Scan(
		&record.ProjectID,
		&record.Name,
		&record.Description,
		&record.OwnerUserID,
		&record.IsArchived,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return models.ProjectRecord{}, err
	}
	return record, nil
}
