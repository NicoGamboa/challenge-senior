package db

import "context"

type Row interface {
	Scan(dest ...any) error
}

type Client interface {
	Exec(ctx context.Context, query string, args ...any) error
	QueryRow(ctx context.Context, query string, args ...any) (Row, error)
}
