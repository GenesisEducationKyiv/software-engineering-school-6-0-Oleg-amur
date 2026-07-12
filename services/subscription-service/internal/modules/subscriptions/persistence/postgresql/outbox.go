package postgresql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/adapters/postgresql"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
)

type OutboxStore struct {
	db postgresqladapter.Queryable
}

func NewOutboxStore(db postgresqladapter.Queryable) *OutboxStore {
	return &OutboxStore{db: db}
}

func (r *OutboxStore) FetchPendingOutbox(
	ctx context.Context,
	limit int,
) (_ []subscriptiondomain.OutboxMessage, err error) {
	query := `
		SELECT id, event_type, payload
		FROM outbox_messages
		WHERE outbox_status = $1
		ORDER BY created_at
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, subscriptiondomain.OutboxStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	var messages []subscriptiondomain.OutboxMessage
	for rows.Next() {
		var message subscriptiondomain.OutboxMessage
		if err := rows.Scan(&message.ID, &message.EventType, &message.Payload); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *OutboxStore) MarkOutboxPublished(ctx context.Context, id int) error {
	query := `
		UPDATE outbox_messages
		SET outbox_status = $1, published_at = CURRENT_TIMESTAMP
		WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, subscriptiondomain.OutboxStatusPublished, id)
	return err
}

func (r *OutboxStore) RecordOutboxFailure(
	ctx context.Context,
	id int,
	cause error,
	maxAttempts int,
) error {
	if maxAttempts <= 0 {
		return fmt.Errorf("outbox max attempts must be positive")
	}

	query := `
		UPDATE outbox_messages
		SET attempts = attempts + 1,
			last_error = $1,
			outbox_status = CASE
				WHEN attempts + 1 >= $2 THEN $3
				ELSE outbox_status
			END
		WHERE id = $4 AND outbox_status = $5`
	_, err := r.db.ExecContext(
		ctx,
		query,
		cause.Error(),
		maxAttempts,
		subscriptiondomain.OutboxStatusFailed,
		id,
		subscriptiondomain.OutboxStatusPending,
	)
	return err
}

func (r *OutboxStore) Create(ctx context.Context, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	query := `
		INSERT INTO outbox_messages (event_type, payload, outbox_status)
		VALUES ($1, $2, $3)`
	_, err = r.db.ExecContext(ctx, query, eventType, body, subscriptiondomain.OutboxStatusPending)
	return err
}
