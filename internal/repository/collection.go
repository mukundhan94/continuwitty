package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const collectionColumns = `
	collection_id,
	project_id,
	owner_user_id,
	name,
	description,
	created_at,
	updated_at,
	deleted_at,
	deleted_by_user_id,
	delete_reason
`

var (
	// ErrCollectionNameExists maps duplicate active-collection name violations.
	ErrCollectionNameExists = errors.New("collection name already exists for this project")
	newCollectionUUID       = uuid.New
)

// CollectionListInput captures collection list filters.
type CollectionListInput struct {
	ProjectID      *string
	OwnerUserID    *uuid.UUID
	IncludeDeleted bool
	Limit          int
	Offset         int
}

// CollectionCreateInput captures collection create dependencies.
type CollectionCreateInput struct {
	ProjectID   string
	OwnerUserID uuid.UUID
	Name        string
	Description string
}

// CollectionUpdateInput captures collection update dependencies.
type CollectionUpdateInput struct {
	CollectionID uuid.UUID
	Name         *string
	Description  *string
}

// CollectionSoftDeleteInput captures collection soft-delete dependencies.
type CollectionSoftDeleteInput struct {
	CollectionID    uuid.UUID
	DeletedByUserID uuid.UUID
	Reason          *string
}

// CollectionAddItemsInput captures collection item add dependencies.
type CollectionAddItemsInput struct {
	CollectionID uuid.UUID
	ActorUserID  uuid.UUID
	EngramIDs    []uuid.UUID
}

// CollectionRemoveItemInput captures collection item delete dependencies.
type CollectionRemoveItemInput struct {
	CollectionID uuid.UUID
	EngramID     uuid.UUID
}

// ListCollections returns collections filtered by project/owner/deleted state.
func ListCollections(
	ctx context.Context,
	db Queryer,
	input CollectionListInput,
) ([]models.EngramCollectionRecord, error) {
	sql, params := buildCollectionListQuery(input)
	rows, err := db.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.EngramCollectionRecord, 0)
	for rows.Next() {
		record, scanErr := scanCollectionRecord(rows)
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

type collectionListQueryBuilder struct {
	whereClauses []string
	params       []any
}

func newCollectionListQueryBuilder() collectionListQueryBuilder {
	return collectionListQueryBuilder{
		whereClauses: []string{"1=1"},
		params:       make([]any, 0),
	}
}

func (builder *collectionListQueryBuilder) addClause(clause string) {
	builder.whereClauses = append(builder.whereClauses, clause)
}

func (builder *collectionListQueryBuilder) addParam(value any) string {
	builder.params = append(builder.params, value)
	return pgxPlaceholder(len(builder.params))
}

func (builder *collectionListQueryBuilder) addOptionalProjectFilter(projectID *string) {
	if projectID == nil {
		return
	}
	trimmed := strings.TrimSpace(*projectID)
	if trimmed == "" {
		return
	}
	placeholder := builder.addParam(trimmed)
	builder.addClause(fmt.Sprintf("project_id = %s", placeholder))
}

func (builder *collectionListQueryBuilder) addOptionalOwnerFilter(ownerUserID *uuid.UUID) {
	if ownerUserID == nil {
		return
	}
	placeholder := builder.addParam(*ownerUserID)
	builder.addClause(fmt.Sprintf("owner_user_id = %s", placeholder))
}

func buildCollectionListQuery(input CollectionListInput) (string, []any) {
	builder := newCollectionListQueryBuilder()
	builder.addOptionalProjectFilter(input.ProjectID)
	builder.addOptionalOwnerFilter(input.OwnerUserID)
	if !input.IncludeDeleted {
		builder.addClause("deleted_at IS NULL")
	}

	limitPlaceholder := builder.addParam(input.Limit)
	offsetPlaceholder := builder.addParam(input.Offset)

	return fmt.Sprintf(
		`
		SELECT %s
		FROM engram_collections
		WHERE %s
		ORDER BY created_at DESC
		LIMIT %s OFFSET %s
		`,
		collectionColumns,
		strings.Join(builder.whereClauses, " AND "),
		limitPlaceholder,
		offsetPlaceholder,
	), builder.params
}

// GetCollection fetches a collection by ID with optional deleted filtering.
func GetCollection(
	ctx context.Context,
	db Queryer,
	collectionID uuid.UUID,
	includeDeleted bool,
) (*models.EngramCollectionRecord, error) {
	sql := fmt.Sprintf(
		`
		SELECT %s
		FROM engram_collections
		WHERE collection_id = $1
		`,
		collectionColumns,
	)
	if !includeDeleted {
		sql += " AND deleted_at IS NULL"
	}
	sql += " LIMIT 1"

	row := db.QueryRow(ctx, sql, collectionID)
	record, err := scanCollectionRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateCollection creates a collection for a project owner.
func CreateCollection(
	ctx context.Context,
	db Queryer,
	input CollectionCreateInput,
) (*models.EngramCollectionRecord, error) {
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			INSERT INTO engram_collections (
				collection_id,
				project_id,
				owner_user_id,
				name,
				description,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, now(), now())
			RETURNING %s
			`,
			collectionColumns,
		),
		newCollectionUUID(),
		input.ProjectID,
		input.OwnerUserID,
		input.Name,
		input.Description,
	)
	record, err := scanCollectionRecord(row)
	if err != nil {
		if isCollectionNameUniqueViolation(err) {
			return nil, ErrCollectionNameExists
		}
		return nil, err
	}
	return &record, nil
}

// UpdateCollection updates mutable collection fields.
func UpdateCollection(
	ctx context.Context,
	db Queryer,
	input CollectionUpdateInput,
) (*models.EngramCollectionRecord, error) {
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			UPDATE engram_collections
			SET
				name = COALESCE($1, name),
				description = COALESCE($2, description),
				updated_at = now()
			WHERE
				collection_id = $3
				AND deleted_at IS NULL
			RETURNING %s
			`,
			collectionColumns,
		),
		input.Name,
		input.Description,
		input.CollectionID,
	)
	record, err := scanCollectionRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		if isCollectionNameUniqueViolation(err) {
			return nil, ErrCollectionNameExists
		}
		return nil, err
	}
	return &record, nil
}

// SoftDeleteCollection soft-deletes a collection when present.
func SoftDeleteCollection(
	ctx context.Context,
	db Queryer,
	input CollectionSoftDeleteInput,
) (bool, error) {
	row := db.QueryRow(
		ctx,
		`
		UPDATE engram_collections
		SET
			deleted_at = now(),
			deleted_by_user_id = $1,
			delete_reason = $2
		WHERE
			collection_id = $3
			AND deleted_at IS NULL
		RETURNING collection_id
		`,
		input.DeletedByUserID,
		input.Reason,
		input.CollectionID,
	)
	var removedID uuid.UUID
	err := row.Scan(&removedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// AddCollectionItems adds visible project-matching engrams to a collection.
func AddCollectionItems(
	ctx context.Context,
	db Queryer,
	input CollectionAddItemsInput,
) (int, error) {
	if len(input.EngramIDs) == 0 {
		return 0, nil
	}
	rows, err := db.Query(
		ctx,
		`
		INSERT INTO engram_collection_items (
			collection_id,
			engram_id,
			added_by_user_id,
			created_at
		)
		SELECT
			c.collection_id,
			e.engram_id,
			$1,
			now()
		FROM engram_collections c
		JOIN engrams e
		  ON e.engram_id = ANY($2::UUID[])
		 AND e.project_id = c.project_id
		 AND e.deleted_at IS NULL
		WHERE
			c.collection_id = $3
			AND c.deleted_at IS NULL
		ON CONFLICT (collection_id, engram_id) DO NOTHING
		RETURNING engram_id
		`,
		input.ActorUserID,
		input.EngramIDs,
		input.CollectionID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var engramID uuid.UUID
		if scanErr := rows.Scan(&engramID); scanErr != nil {
			return 0, scanErr
		}
		count += 1
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

// RemoveCollectionItem removes one engram from a collection.
func RemoveCollectionItem(
	ctx context.Context,
	db Queryer,
	input CollectionRemoveItemInput,
) (bool, error) {
	row := db.QueryRow(
		ctx,
		`
		DELETE FROM engram_collection_items
		WHERE collection_id = $1 AND engram_id = $2
		RETURNING engram_id
		`,
		input.CollectionID,
		input.EngramID,
	)
	var removedEngramID uuid.UUID
	err := row.Scan(&removedEngramID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func scanCollectionRecord(row interface {
	Scan(dest ...any) error
}) (models.EngramCollectionRecord, error) {
	var record models.EngramCollectionRecord
	err := row.Scan(
		&record.CollectionID,
		&record.ProjectID,
		&record.OwnerUserID,
		&record.Name,
		&record.Description,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.DeletedAt,
		&record.DeletedByUserID,
		&record.DeleteReason,
	)
	if err != nil {
		return models.EngramCollectionRecord{}, err
	}
	return record, nil
}

func isCollectionNameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if pgErr.Code != "23505" {
		return false
	}
	if strings.Contains(pgErr.ConstraintName, "engram_collections_project_name_active_uidx") {
		return true
	}
	return strings.Contains(strings.ToLower(pgErr.Message), "duplicate")
}
