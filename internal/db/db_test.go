package db

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"engram/internal/config"

	"github.com/jackc/pgx/v5"
)

type fakeBeginner struct {
	tx       *fakeTx
	beginErr error
	calls    int
}

func (runner *fakeBeginner) Begin(_ context.Context) (Transaction, error) {
	runner.calls += 1
	if runner.beginErr != nil {
		return nil, runner.beginErr
	}
	return runner.tx, nil
}

type fakeTx struct {
	execSQL        []string
	execParams     [][]any
	querySQL       []string
	queryParams    [][]any
	queryRowResult *fakeRow
	commitCount    int
	rollbackCount  int
	execErr        error
	commitErr      error
	rollbackErr    error
}

func (tx *fakeTx) Exec(_ context.Context, sql string, args ...any) error {
	tx.execSQL = append(tx.execSQL, sql)
	copied := append([]any(nil), args...)
	tx.execParams = append(tx.execParams, copied)
	if tx.execErr != nil {
		return tx.execErr
	}
	return nil
}

func (tx *fakeTx) QueryRow(_ context.Context, sql string, args ...any) RowScanner {
	tx.querySQL = append(tx.querySQL, sql)
	copied := append([]any(nil), args...)
	tx.queryParams = append(tx.queryParams, copied)
	if tx.queryRowResult == nil {
		tx.queryRowResult = &fakeRow{scanErr: pgx.ErrNoRows}
	}
	return tx.queryRowResult
}

func (tx *fakeTx) Commit(_ context.Context) error {
	tx.commitCount += 1
	if tx.commitErr != nil {
		return tx.commitErr
	}
	return nil
}

func (tx *fakeTx) Rollback(_ context.Context) error {
	tx.rollbackCount += 1
	if tx.rollbackErr != nil {
		return tx.rollbackErr
	}
	return nil
}

type fakeRow struct {
	value   string
	scanErr error
}

func (row *fakeRow) Scan(dest ...any) error {
	if row.scanErr != nil {
		return row.scanErr
	}
	if len(dest) != 1 {
		return errors.New("expected one scan destination")
	}
	valuePtr, ok := dest[0].(*string)
	if !ok {
		return errors.New("expected *string destination")
	}
	*valuePtr = row.value
	return nil
}

func TestWithTransactionCommitsOnSuccess(t *testing.T) {
	tx := &fakeTx{}
	runner := &fakeBeginner{tx: tx}

	err := WithTransaction(context.Background(), runner, func(_ Transaction) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected transaction to succeed: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected begin to be called once, got %d", runner.calls)
	}
	if tx.commitCount != 1 {
		t.Fatalf("expected one commit, got %d", tx.commitCount)
	}
	if tx.rollbackCount != 0 {
		t.Fatalf("expected no rollback, got %d", tx.rollbackCount)
	}
}

func TestWithTransactionRollsBackOnError(t *testing.T) {
	tx := &fakeTx{}
	runner := &fakeBeginner{tx: tx}

	err := WithTransaction(context.Background(), runner, func(_ Transaction) error {
		return errors.New("boom")
	})
	if err == nil {
		t.Fatalf("expected transaction to fail")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped error to include cause, got %q", err.Error())
	}
	if tx.commitCount != 0 {
		t.Fatalf("expected no commit, got %d", tx.commitCount)
	}
	if tx.rollbackCount != 1 {
		t.Fatalf("expected one rollback, got %d", tx.rollbackCount)
	}
}

func TestEnsureSchemaInitializedExecutesSchemaSQL(t *testing.T) {
	tx := &fakeTx{}
	runner := &fakeBeginner{tx: tx}

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "001_schema.sql")
	if err := os.WriteFile(schemaPath, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	settings := config.Settings{
		AppEnv:                            "development",
		UIDemoUsername:                    "admin",
		UIDemoPassword:                    "admin123",
		OAuthRequireProtectedRegistration: true,
	}

	err := EnsureSchemaInitialized(
		context.Background(),
		runner,
		settings,
		schemaPath,
		func(password string) (string, error) { return "hashed-" + password, nil },
	)
	if err != nil {
		t.Fatalf("expected schema init to succeed: %v", err)
	}
	if len(tx.execSQL) == 0 {
		t.Fatalf("expected schema SQL to execute")
	}
	if tx.execSQL[0] != "SELECT 1;" {
		t.Fatalf("expected schema SQL to be executed first, got %q", tx.execSQL[0])
	}
	if tx.commitCount != 1 {
		t.Fatalf("expected one commit, got %d", tx.commitCount)
	}
}

func TestHardenBootstrapAdminCredentialsUpdatesDefaultHash(t *testing.T) {
	tx := &fakeTx{
		queryRowResult: &fakeRow{value: defaultBootstrapAdminHash},
	}

	settings := config.Settings{
		AppEnv:                            "production",
		UIDemoUsername:                    "admin",
		UIDemoPassword:                    "StrongPassword-12345",
		OAuthRequireProtectedRegistration: true,
	}

	err := hardenBootstrapAdminCredentials(
		context.Background(),
		tx,
		settings,
		func(_ string) (string, error) { return "hashed-replacement", nil },
	)
	if err != nil {
		t.Fatalf("expected credential hardening to succeed: %v", err)
	}
	if len(tx.querySQL) == 0 || !strings.Contains(tx.querySQL[0], "SELECT password_hash") {
		t.Fatalf("expected password hash lookup query to run")
	}
	if len(tx.execSQL) == 0 {
		t.Fatalf("expected update query to run")
	}
	if !strings.Contains(tx.execSQL[len(tx.execSQL)-1], "UPDATE users") {
		t.Fatalf("expected final query to update user hash, got %q", tx.execSQL[len(tx.execSQL)-1])
	}
}

func TestHardenBootstrapAdminCredentialsSkipsNonDefaultHash(t *testing.T) {
	tx := &fakeTx{
		queryRowResult: &fakeRow{value: "custom-password-hash"},
	}
	settings := config.Settings{
		AppEnv:                            "production",
		UIDemoUsername:                    "admin",
		UIDemoPassword:                    "StrongPassword-12345",
		OAuthRequireProtectedRegistration: true,
	}

	err := hardenBootstrapAdminCredentials(
		context.Background(),
		tx,
		settings,
		func(_ string) (string, error) { return "hashed-replacement", nil },
	)
	if err != nil {
		t.Fatalf("expected credential hardening to succeed: %v", err)
	}
	if len(tx.querySQL) == 0 || !strings.Contains(tx.querySQL[0], "SELECT password_hash") {
		t.Fatalf("expected password hash lookup query to run")
	}
	for _, sql := range tx.execSQL {
		if strings.Contains(sql, "UPDATE users") {
			t.Fatalf("did not expect update query for non-default hash")
		}
	}
}
