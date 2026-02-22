package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeQueryer struct {
	queryRowResult  *fakeRow
	queryRowsResult *fakeRows
	queryRowSQL     []string
	queryRowArgs    [][]any
	querySQL        []string
	queryArgs       [][]any
	queryErr        error
}

func (f *fakeQueryer) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.queryRowSQL = append(f.queryRowSQL, sql)
	f.queryRowArgs = append(f.queryRowArgs, append([]any(nil), args...))
	if f.queryRowResult == nil {
		return &fakeRow{err: pgx.ErrNoRows}
	}
	return f.queryRowResult
}

func (f *fakeQueryer) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.querySQL = append(f.querySQL, sql)
	f.queryArgs = append(f.queryArgs, append([]any(nil), args...))
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	if f.queryRowsResult == nil {
		return &fakeRows{}, nil
	}
	return f.queryRowsResult, nil
}

type fakeRow struct {
	values []any
	err    error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("scan destination/value length mismatch")
	}
	for i, value := range r.values {
		if err := assignScanDest(dest[i], value); err != nil {
			return err
		}
	}
	return nil
}

type fakeRows struct {
	values [][]any
	index  int
	err    error
}

func (r *fakeRows) Close() {}

func (r *fakeRows) Err() error { return r.err }

func (r *fakeRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index += 1
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.values) {
		return errors.New("scan called without valid row")
	}
	row := r.values[r.index-1]
	if len(dest) != len(row) {
		return errors.New("scan destination/value length mismatch")
	}
	for i, value := range row {
		if err := assignScanDest(dest[i], value); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeRows) Values() ([]any, error) {
	if r.index == 0 || r.index > len(r.values) {
		return nil, errors.New("values called without valid row")
	}
	return r.values[r.index-1], nil
}

func (r *fakeRows) RawValues() [][]byte { return nil }

func (r *fakeRows) Conn() *pgx.Conn { return nil }

func assignScanDest(destination any, value any) error {
	destinationValue := reflect.ValueOf(destination)
	if destinationValue.Kind() != reflect.Pointer || destinationValue.IsNil() {
		return errors.New("destination must be a non-nil pointer")
	}
	target := destinationValue.Elem()
	if value == nil {
		if target.Kind() != reflect.Pointer {
			return errors.New("nil value requires pointer destination")
		}
		target.Set(reflect.Zero(target.Type()))
		return nil
	}

	source := reflect.ValueOf(value)
	if target.Kind() == reflect.Pointer {
		return assignPointerValue(target, source)
	}
	return assignValue(target, source)
}

func assignPointerValue(target, source reflect.Value) error {
	elemType := target.Type().Elem()
	if source.Type().AssignableTo(elemType) {
		assigned := reflect.New(elemType)
		assigned.Elem().Set(source)
		target.Set(assigned)
		return nil
	}
	if source.Type().ConvertibleTo(elemType) {
		assigned := reflect.New(elemType)
		assigned.Elem().Set(source.Convert(elemType))
		target.Set(assigned)
		return nil
	}
	return fmt.Errorf("cannot assign %s to %s", source.Type(), target.Type())
}

func assignValue(target, source reflect.Value) error {
	if source.Type().AssignableTo(target.Type()) {
		target.Set(source)
		return nil
	}
	if source.Type().ConvertibleTo(target.Type()) {
		target.Set(source.Convert(target.Type()))
		return nil
	}
	return fmt.Errorf("cannot assign %s to %s", source.Type(), target.Type())
}

func TestGetUserAuthRecordLooksUpUsername(t *testing.T) {
	createdAt := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{userID, "admin", "hash", "admin", true, "engram-vault", createdAt}},
	}

	record, err := GetUserAuthRecord(context.Background(), db, "admin")
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, userID, record.UserID)
	requireEqual(t, "admin", record.Username)
	requireEqual(t, "hash", record.PasswordHash)
	requireEqual(t, 1, len(db.queryRowArgs))
	requireEqual(t, 1, len(db.queryRowArgs[0]))
	requireEqual(t, "admin", db.queryRowArgs[0][0])
}

func TestListUsersReturnsRecords(t *testing.T) {
	createdAt := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000011")
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{{userID, "viewer_1", "viewer", true, nil, createdAt}}},
	}

	result, err := ListUsers(context.Background(), db, 1, 4)
	requireNoError(t, err)
	requireEqual(t, 1, len(result))
	requireEqual(t, "viewer_1", result[0].Username)
	requireEqual(t, models.UserRoleViewer, result[0].Role)
	requireEqual(t, 1, len(db.queryArgs))
	if !reflect.DeepEqual(db.queryArgs[0], []any{1, 4}) {
		t.Fatalf("expected query args [1 4], got %#v", db.queryArgs[0])
	}
}

func TestCreateUserReturnsCreatedUser(t *testing.T) {
	createdAt := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	createdUserID := uuid.MustParse("00000000-0000-0000-0000-000000000111")
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{createdUserID, "analyst_1", "analyst", true, "engram-vault", createdAt}},
	}

	originalUUID := newUUID
	newUUID = func() uuid.UUID { return createdUserID }
	t.Cleanup(func() { newUUID = originalUUID })

	record, err := CreateUser(
		context.Background(),
		db,
		CreateUserInput{
			Username:     "analyst_1",
			PasswordHash: "hash-value",
			Role:         models.UserRoleAnalyst,
			IsActive:     true,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, createdUserID, record.UserID)
	requireEqual(t, models.UserRoleAnalyst, record.Role)
	requireEqual(t, 1, len(db.queryRowArgs))
	expectedArgs := []any{createdUserID, "analyst_1", "hash-value", "analyst", true}
	if !reflect.DeepEqual(db.queryRowArgs[0], expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestCreateUserReturnsUsernameExistsOnDuplicate(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: &pgconn.PgError{Code: "23505", Message: "duplicate username"}},
	}

	record, err := CreateUser(
		context.Background(),
		db,
		CreateUserInput{
			Username:     "admin",
			PasswordHash: "hash-value",
			Role:         models.UserRoleAdmin,
			IsActive:     true,
		},
	)
	if !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("expected ErrUsernameExists, got %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil user record on duplicate")
	}
}

func TestUpdateUserReturnsNilWhenNotFound(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	targetUserID := uuid.MustParse("00000000-0000-0000-0000-000000000999")

	record, err := UpdateUser(context.Background(), db, targetUserID, UserUpdateInput{})
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when user not found")
	}
}

func TestUpdateUserAppliesProvidedFields(t *testing.T) {
	createdAt := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	targetUserID := uuid.MustParse("00000000-0000-0000-0000-000000000212")
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{targetUserID, "viewer_2", "viewer", false, nil, createdAt}},
	}

	role := models.UserRoleViewer
	isActive := false
	passwordHash := "new-hash"
	record, err := UpdateUser(
		context.Background(),
		db,
		targetUserID,
		UserUpdateInput{Role: &role, IsActive: &isActive, PasswordHash: &passwordHash},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, targetUserID, record.UserID)
	requireEqual(t, models.UserRoleViewer, record.Role)
	requireEqual(t, 1, len(db.queryRowArgs))
	expectedArgs := []any{"viewer", &isActive, &passwordHash, targetUserID}
	if !reflect.DeepEqual(db.queryRowArgs[0], expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
}

func requireNotNil(t *testing.T, value any) {
	t.Helper()
	if value == nil {
		t.Fatalf("expected non-nil value")
	}
}

func requireEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}
