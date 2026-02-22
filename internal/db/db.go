package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"engram/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultBootstrapAdminHash = "pbkdf2_sha256$390000$00112233445566778899aabbccddeeff$45c0bdc16f1609a69d14a4e8d89974fd24556c45af75ee01c34bdb9de1c1832a"

// RowScanner scans a single row result.
type RowScanner interface {
	Scan(dest ...any) error
}

// Transaction captures the DB operations needed by migration bootstrap logic.
type Transaction interface {
	Exec(ctx context.Context, sql string, args ...any) error
	QueryRow(ctx context.Context, sql string, args ...any) RowScanner
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TxBeginner starts a transaction.
type TxBeginner interface {
	Begin(ctx context.Context) (Transaction, error)
}

// PasswordHasher creates a password hash for bootstrap hardening.
type PasswordHasher func(password string) (string, error)

type pgxTxAdapter struct {
	tx pgx.Tx
}

func (adapter pgxTxAdapter) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := adapter.tx.Exec(ctx, sql, args...)
	return err
}

func (adapter pgxTxAdapter) QueryRow(ctx context.Context, sql string, args ...any) RowScanner {
	return adapter.tx.QueryRow(ctx, sql, args...)
}

func (adapter pgxTxAdapter) Commit(ctx context.Context) error {
	return adapter.tx.Commit(ctx)
}

func (adapter pgxTxAdapter) Rollback(ctx context.Context) error {
	return adapter.tx.Rollback(ctx)
}

// PgxPoolBeginner adapts pgxpool to the TxBeginner contract used by migration bootstrap logic.
type PgxPoolBeginner struct {
	Pool *pgxpool.Pool
}

// Begin starts a pgx transaction and wraps it with the migration transaction abstraction.
func (runner PgxPoolBeginner) Begin(ctx context.Context) (Transaction, error) {
	if runner.Pool == nil {
		return nil, errors.New("pgx pool is nil")
	}
	tx, err := runner.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return pgxTxAdapter{tx: tx}, nil
}

// NewPool creates a pgx connection pool from settings.
func NewPool(ctx context.Context, settings config.Settings) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(settings.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect pgx pool: %w", err)
	}
	return pool, nil
}

// Ping checks database connectivity.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("pgx pool is nil")
	}
	return pool.Ping(ctx)
}

// DefaultSchemaPath resolves db/init/001_schema.sql from the repository root.
func DefaultSchemaPath() string {
	_, filePath, _, ok := runtime.Caller(0)
	if !ok {
		return "db/init/001_schema.sql"
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filePath), "..", "..", "db", "init", "001_schema.sql"))
}

// WithTransaction starts a transaction, runs fn, commits on success, and rolls back on failure.
func WithTransaction(ctx context.Context, beginner TxBeginner, fn func(tx Transaction) error) error {
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil {
			return fmt.Errorf("transaction failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// EnsureSchemaInitialized executes schema bootstrap SQL and production bootstrap credential hardening.
func EnsureSchemaInitialized(
	ctx context.Context,
	beginner TxBeginner,
	settings config.Settings,
	schemaPath string,
	hashPassword PasswordHasher,
) error {
	if hashPassword == nil {
		return errors.New("hash password function is required")
	}
	if strings.TrimSpace(schemaPath) == "" {
		schemaPath = DefaultSchemaPath()
	}
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema file: %w", err)
	}

	return WithTransaction(ctx, beginner, func(tx Transaction) error {
		if err := tx.Exec(ctx, string(schemaSQL)); err != nil {
			return err
		}
		if err := hardenBootstrapAdminCredentials(ctx, tx, settings, hashPassword); err != nil {
			return err
		}
		return nil
	})
}

func hardenBootstrapAdminCredentials(
	ctx context.Context,
	tx Transaction,
	settings config.Settings,
	hashPassword PasswordHasher,
) error {
	if !config.IsProductionEnv(settings) {
		return nil
	}

	username := strings.TrimSpace(settings.UIDemoUsername)
	if username == "" {
		username = "admin"
	}

	var currentHash string
	err := tx.QueryRow(
		ctx,
		`SELECT password_hash FROM users WHERE username = $1`,
		username,
	).Scan(&currentHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if currentHash != defaultBootstrapAdminHash {
		return nil
	}

	replacementHash := strings.TrimSpace(settings.UIDemoPasswordHash)
	if replacementHash == "" {
		replacementHash, err = hashPassword(settings.UIDemoPassword)
		if err != nil {
			return err
		}
	}
	return tx.Exec(
		ctx,
		`UPDATE users SET password_hash = $1 WHERE username = $2`,
		replacementHash,
		username,
	)
}
