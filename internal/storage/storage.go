package storage

import (
	"context"
)

type Row interface {
	Scan(dest ...any) error
}

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type Storage interface {
	ExecContext(ctx context.Context, query string, args ...any) error
	QueryRowContext(ctx context.Context, query string, args ...any) Row
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	MigrateUp(ctx context.Context) error
	Close()
}
