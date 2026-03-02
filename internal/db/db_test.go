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

func TestWithTransaction(t *testing.T) {
	testCases := []struct {
		name           string
		fn             func(tx Transaction) error
		expectErr      bool
		expectedCommit int
		expectedRB     int
	}{
		{
			name:           "commits on success",
			fn:             func(_ Transaction) error { return nil },
			expectErr:      false,
			expectedCommit: 1,
			expectedRB:     0,
		},
		{
			name:           "rolls back on failure",
			fn:             func(_ Transaction) error { return errors.New("boom") },
			expectErr:      true,
			expectedCommit: 0,
			expectedRB:     1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tx := &fakeTx{}
			runner := &fakeBeginner{tx: tx}

			err := WithTransaction(context.Background(), runner, testCase.fn)
			assertTransactionError(t, err, testCase.expectErr)
			assertTransactionCounters(
				t,
				runner,
				tx,
				transactionCounterExpectation{
					commit:   testCase.expectedCommit,
					rollback: testCase.expectedRB,
				},
			)
		})
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
		SchemaInitializationOptions{
			Settings:     settings,
			SchemaPath:   schemaPath,
			HashPassword: func(password string) (string, error) { return "hashed-" + password, nil },
		},
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

func TestHardenBootstrapAdminCredentials(t *testing.T) {
	testCases := []struct {
		name         string
		storedHash   string
		expectUpdate bool
	}{
		{
			name:         "updates default hash",
			storedHash:   defaultBootstrapAdminHash,
			expectUpdate: true,
		},
		{
			name:         "skips non-default hash",
			storedHash:   "custom-password-hash",
			expectUpdate: false,
		},
	}

	settings := config.Settings{
		AppEnv:                            "production",
		UIDemoUsername:                    "admin",
		UIDemoPassword:                    "StrongPassword-12345",
		OAuthRequireProtectedRegistration: true,
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tx := &fakeTx{
				queryRowResult: &fakeRow{value: testCase.storedHash},
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
			assertPasswordHashLookupQuery(t, tx.querySQL)
			updated := hasUserUpdateExec(tx.execSQL)
			if updated != testCase.expectUpdate {
				t.Fatalf("expected update=%t, got %t", testCase.expectUpdate, updated)
			}
		})
	}
}

func assertTransactionError(t *testing.T, err error, expectErr bool) {
	t.Helper()
	if expectErr {
		if err == nil {
			t.Fatalf("expected transaction to fail")
		}
		if !strings.Contains(err.Error(), "boom") {
			t.Fatalf("expected wrapped error to include cause, got %q", err.Error())
		}
		return
	}
	if err != nil {
		t.Fatalf("expected transaction to succeed: %v", err)
	}
}

type transactionCounterExpectation struct {
	commit   int
	rollback int
}

func assertTransactionCounters(
	t *testing.T,
	runner *fakeBeginner,
	tx *fakeTx,
	expectation transactionCounterExpectation,
) {
	t.Helper()
	if runner.calls != 1 {
		t.Fatalf("expected begin to be called once, got %d", runner.calls)
	}
	if tx.commitCount != expectation.commit {
		t.Fatalf("expected %d commit(s), got %d", expectation.commit, tx.commitCount)
	}
	if tx.rollbackCount != expectation.rollback {
		t.Fatalf("expected %d rollback(s), got %d", expectation.rollback, tx.rollbackCount)
	}
}

func assertPasswordHashLookupQuery(t *testing.T, queries []string) {
	t.Helper()
	if len(queries) == 0 || !strings.Contains(queries[0], "SELECT password_hash") {
		t.Fatalf("expected password hash lookup query to run")
	}
}

func hasUserUpdateExec(statements []string) bool {
	for _, statement := range statements {
		if strings.Contains(statement, "UPDATE users") {
			return true
		}
	}
	return false
}

func TestResolveDefaultSchemaPath(t *testing.T) {
	testCases := []struct {
		name       string
		callerPath string
		workingDir string
		existing   map[string]bool
		expected   string
	}{
		{
			name:       "prefers working directory schema path",
			workingDir: "/srv",
			existing: map[string]bool{
				filepath.Clean("/srv/db/init/001_schema.sql"): true,
			},
			expected: filepath.Clean("/srv/db/init/001_schema.sql"),
		},
		{
			name:       "resolves trimpath caller path via repository root fallback",
			callerPath: filepath.Clean("engram/internal/db/db.go"),
			workingDir: "/srv",
			existing: map[string]bool{
				filepath.Clean("db/init/001_schema.sql"): true,
			},
			expected: filepath.Clean("db/init/001_schema.sql"),
		},
		{
			name:       "falls back to default relative path when nothing exists",
			callerPath: filepath.Clean("engram/internal/db/db.go"),
			workingDir: "/srv",
			existing:   map[string]bool{},
			expected:   filepath.Clean("db/init/001_schema.sql"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pathExists := func(path string) bool {
				return testCase.existing[filepath.Clean(path)]
			}
			resolved := resolveDefaultSchemaPath(testCase.callerPath, testCase.workingDir, pathExists)
			if filepath.Clean(resolved) != filepath.Clean(testCase.expected) {
				t.Fatalf("expected %q, got %q", testCase.expected, resolved)
			}
		})
	}
}
