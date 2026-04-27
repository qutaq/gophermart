package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qutaq/gophermart/internal/storage"
	"github.com/qutaq/gophermart/migrations"
)

const pingTimeout = 5 * time.Second

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, errors.New("postgres: пустой DATABASE_URI")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: создание пула: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return &Store{pool: pool}, nil
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *Store) ExecContext(ctx context.Context, query string, args ...any) error {
	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("postgres: exec: %w", err)
	}
	return nil
}

func (s *Store) QueryRowContext(ctx context.Context, query string, args ...any) storage.Row {
	return s.pool.QueryRow(ctx, query, args...)
}

func (s *Store) QueryContext(ctx context.Context, query string, args ...any) (storage.Rows, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: query: %w", err)
	}
	return rows, nil
}

func (s *Store) MigrateUp(ctx context.Context) error {
	migrationFiles, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		return fmt.Errorf("postgres: find migrations: %w", err)
	}

	for _, migrationFile := range migrationFiles {
		query, err := migrations.Files.ReadFile(migrationFile)
		if err != nil {
			return fmt.Errorf("postgres: read migration %q: %w", migrationFile, err)
		}

		if err := s.ExecContext(ctx, string(query)); err != nil {
			return fmt.Errorf("postgres: migrate %q: %w", migrationFile, err)
		}
	}
	return nil
}

func (s *Store) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}
