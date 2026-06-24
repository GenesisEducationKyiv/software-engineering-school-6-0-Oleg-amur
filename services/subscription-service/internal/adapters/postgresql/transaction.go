package postgresql

import (
	"context"
	"database/sql"
	"errors"
)

var ErrTransactionAlreadyActive = errors.New("transaction already active")

type TransactionManager struct {
	db *sql.DB
}

func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (m *TransactionManager) Run(
	ctx context.Context,
	work func(context.Context) error,
) (err error) {
	if _, ok := transactionFromContext(ctx); ok {
		return ErrTransactionAlreadyActive
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTransaction(tx, &err)

	if err = work(context.WithValue(ctx, transactionContextKey{}, tx)); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

type transactionContextKey struct{}

func transactionFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(transactionContextKey{}).(*sql.Tx)
	return tx, ok
}

func rollbackTransaction(tx *sql.Tx, err *error) {
	rollbackErr := tx.Rollback()
	if *err != nil && rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
		*err = errors.Join(*err, rollbackErr)
	}
}
