package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"

	"github.com/sund3RRR/ion/pkg/ion/store/migrations"
	"github.com/sund3RRR/ion/pkg/ion/store/sqlc"

	_ "github.com/mattn/go-sqlite3"
)

const sqliteDriver = "sqlite3"

// Option configures Store creation.
type Option func(*options)

type options struct {
	skipMigrations bool
}

// WithoutMigrations prevents Open from applying embedded migrations.
func WithoutMigrations() Option {
	return func(opts *options) {
		opts.skipMigrations = true
	}
}

// Store owns an ION SQLite database handle and sqlc query set.
type Store struct {
	db      *sql.DB
	queries *sqlc.Queries
}

// Open opens an ION SQLite store at path, creates its parent directory, and
// applies embedded migrations unless disabled with WithoutMigrations.
func Open(ctx context.Context, path string, opts ...Option) (*Store, error) {
	if path == "" {
		return nil, errors.New("store: open: empty database path")
	}

	var cfg options
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("store: create database directory %q: %w", filepath.Dir(path), err)
	}

	db, err := sql.Open(sqliteDriver, sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("store: open database %q: %w", path, err)
	}

	db.SetMaxOpenConns(1)

	store := &Store{
		db:      db,
		queries: sqlc.New(db),
	}

	if err := db.PingContext(ctx); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("store: ping database %q: %w", path, err)
	}

	if !cfg.skipMigrations {
		if err := store.Migrate(ctx); err != nil {
			_ = store.Close()
			return nil, err
		}
	}

	return store, nil
}

// Close closes the underlying SQLite database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB returns the underlying SQLite database handle.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Queries returns the sqlc-generated query set bound to the store database.
func (s *Store) Queries() *sqlc.Queries {
	return s.queries
}

// Migrate applies all embedded store migrations.
func (s *Store) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect(sqliteDriver); err != nil {
		return fmt.Errorf("store: set migration dialect: %w", err)
	}

	if err := goose.UpContext(ctx, s.db, "."); err != nil {
		return fmt.Errorf("store: migrate database: %w", err)
	}

	return nil
}

// WithTx runs fn inside a SQLite transaction using sqlc queries bound to that
// transaction. The transaction commits when fn returns nil and rolls back when
// fn returns an error.
func (s *Store) WithTx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin transaction: %w", err)
	}

	if err := fn(s.queries.WithTx(tx)); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("store: rollback transaction after error %w: %w", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit transaction: %w", err)
	}

	return nil
}

func sqliteDSN(path string) string {
	values := url.Values{}
	values.Set("_busy_timeout", "5000")
	values.Set("_foreign_keys", "on")
	values.Set("_journal_mode", "WAL")

	return (&url.URL{
		Scheme:   "file",
		Path:     path,
		RawQuery: values.Encode(),
	}).String()
}
