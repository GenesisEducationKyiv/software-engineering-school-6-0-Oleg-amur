package postgresql

import (
	"context"
	"database/sql"
)

type ContextQueryable struct {
	db *sql.DB
}

func NewContextQueryable(db *sql.DB) *ContextQueryable {
	return &ContextQueryable{db: db}
}

func (q *ContextQueryable) QueryContext(
	ctx context.Context,
	query string,
	args ...any,
) (*sql.Rows, error) {
	return queryableFromContext(ctx, q.db).QueryContext(ctx, query, args...)
}

func (q *ContextQueryable) QueryRowContext(
	ctx context.Context,
	query string,
	args ...any,
) *sql.Row {
	return queryableFromContext(ctx, q.db).QueryRowContext(ctx, query, args...)
}

func (q *ContextQueryable) ExecContext(
	ctx context.Context,
	query string,
	args ...any,
) (sql.Result, error) {
	return queryableFromContext(ctx, q.db).ExecContext(ctx, query, args...)
}

func queryableFromContext(ctx context.Context, fallback Queryable) Queryable {
	if tx, ok := transactionFromContext(ctx); ok {
		return tx
	}
	return fallback
}
