package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrUsernameExists maps duplicate username violations from storage.
	ErrUsernameExists = errors.New("username already exists")
	newUUID           = uuid.New
)

// Queryer captures DB behavior used by user repository operations.
type Queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// UserUpdateInput captures optional mutable user fields.
type UserUpdateInput struct {
	Role         *models.UserRole
	IsActive     *bool
	PasswordHash *string
}

// CreateUserInput captures user attributes for user creation.
type CreateUserInput struct {
	Username     string
	PasswordHash string
	Role         models.UserRole
	IsActive     bool
}

func GetUserAuthRecord(ctx context.Context, db Queryer, username string) (*models.UserAuthRecord, error) {
	return getUserAuthRecord(
		ctx,
		db,
		"username = $1",
		username,
	)
}

func GetUserAuthRecordByID(ctx context.Context, db Queryer, userID uuid.UUID) (*models.UserAuthRecord, error) {
	return getUserAuthRecord(
		ctx,
		db,
		"user_id = $1",
		userID,
	)
}

func getUserAuthRecord(ctx context.Context, db Queryer, whereClause string, arg any) (*models.UserAuthRecord, error) {
	row := db.QueryRow(
		ctx,
		`SELECT user_id, username, password_hash, role, is_active, default_project_id, created_at
		FROM users
		WHERE `+whereClause,
		arg,
	)
	record, err := scanUserAuthRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func ListUsers(ctx context.Context, db Queryer, limit, offset int) ([]models.UserRecord, error) {
	rows, err := db.Query(
		ctx,
		`
		SELECT user_id, username, role, is_active, default_project_id, created_at
		FROM users
		ORDER BY created_at ASC
		LIMIT $1 OFFSET $2
		`,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.UserRecord, 0)
	for rows.Next() {
		record, scanErr := scanUserRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		results = append(results, *record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func CreateUser(ctx context.Context, db Queryer, input CreateUserInput) (*models.UserRecord, error) {
	if _, err := models.ParseUserRole(string(input.Role)); err != nil {
		return nil, err
	}
	row := db.QueryRow(
		ctx,
		`
		INSERT INTO users (user_id, username, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING user_id, username, role, is_active, default_project_id, created_at
		`,
		newUUID(),
		input.Username,
		input.PasswordHash,
		string(input.Role),
		input.IsActive,
	)
	record, err := scanUserRecord(row)
	if err != nil {
		if isDuplicateUsernameError(err) {
			return nil, ErrUsernameExists
		}
		return nil, err
	}
	return record, nil
}

func UpdateUser(ctx context.Context, db Queryer, userID uuid.UUID, input UserUpdateInput) (*models.UserRecord, error) {
	var roleValue any
	if input.Role != nil {
		if _, err := models.ParseUserRole(string(*input.Role)); err != nil {
			return nil, err
		}
		roleValue = string(*input.Role)
	}

	row := db.QueryRow(
		ctx,
		`
		UPDATE users
		SET
			role = COALESCE($1, role),
			is_active = COALESCE($2, is_active),
			password_hash = COALESCE($3, password_hash)
		WHERE user_id = $4
		RETURNING user_id, username, role, is_active, default_project_id, created_at
		`,
		roleValue,
		input.IsActive,
		input.PasswordHash,
		userID,
	)
	record, err := scanUserRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func scanUserAuthRecord(row pgx.Row) (*models.UserAuthRecord, error) {
	base, passwordHash, err := scanUserFields(row, true)
	if err != nil {
		return nil, err
	}
	role, err := parseUserRole(base.RoleValue)
	if err != nil {
		return nil, err
	}
	return userAuthRecordFromScan(base, role, passwordHash), nil
}

func scanUserRecord(row interface {
	Scan(dest ...any) error
}) (*models.UserRecord, error) {
	base, _, err := scanUserFields(row, false)
	if err != nil {
		return nil, err
	}
	role, err := parseUserRole(base.RoleValue)
	if err != nil {
		return nil, err
	}
	return userRecordFromScan(base, role), nil
}

func scanUserFields(
	row interface {
		Scan(dest ...any) error
	},
	withPassword bool,
) (userScanFields, string, error) {
	var (
		base         userScanFields
		passwordHash string
		err          error
	)
	if withPassword {
		err = row.Scan(
			&base.UserID,
			&base.Username,
			&passwordHash,
			&base.RoleValue,
			&base.IsActive,
			&base.DefaultProjectID,
			&base.CreatedAt,
		)
	} else {
		err = row.Scan(
			&base.UserID,
			&base.Username,
			&base.RoleValue,
			&base.IsActive,
			&base.DefaultProjectID,
			&base.CreatedAt,
		)
	}
	if err != nil {
		return userScanFields{}, "", err
	}
	return base, passwordHash, nil
}

func parseUserRole(roleValue string) (models.UserRole, error) {
	role, err := models.ParseUserRole(roleValue)
	if err != nil {
		return "", err
	}
	return role, nil
}

func userRecordFromScan(base userScanFields, role models.UserRole) *models.UserRecord {
	return &models.UserRecord{
		UserID:           base.UserID,
		Username:         base.Username,
		Role:             role,
		IsActive:         base.IsActive,
		DefaultProjectID: base.DefaultProjectID,
		CreatedAt:        base.CreatedAt,
	}
}

func userAuthRecordFromScan(base userScanFields, role models.UserRole, passwordHash string) *models.UserAuthRecord {
	return &models.UserAuthRecord{
		UserID:           base.UserID,
		Username:         base.Username,
		PasswordHash:     passwordHash,
		Role:             role,
		IsActive:         base.IsActive,
		DefaultProjectID: base.DefaultProjectID,
		CreatedAt:        base.CreatedAt,
	}
}

type userScanFields struct {
	UserID           uuid.UUID
	Username         string
	RoleValue        string
	IsActive         bool
	DefaultProjectID *string
	CreatedAt        time.Time
}

func isDuplicateUsernameError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
