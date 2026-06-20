package postgresql

import (
	"context"
	"encoding/json"
	"fmt"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type OutboxRepository struct {
	db postgresqladapter.Queryable
}

func NewOutboxRepository(db postgresqladapter.Queryable) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) FetchPendingOutbox(
	ctx context.Context,
	limit int,
) ([]subscriptiondomain.OutboxMessage, error) {
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
	defer rows.Close()

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

func (r *OutboxRepository) MarkOutboxPublished(ctx context.Context, id int) error {
	query := `
		UPDATE outbox_messages
		SET outbox_status = $1, published_at = CURRENT_TIMESTAMP
		WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, subscriptiondomain.OutboxStatusPublished, id)
	return err
}

func (r *OutboxRepository) MarkOutboxPublishFailed(ctx context.Context, id int, cause error) error {
	query := `
		UPDATE outbox_messages
		SET attempts = attempts + 1, last_error = $1
		WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, cause.Error(), id)
	return err
}

func (r *OutboxRepository) Create(ctx context.Context, eventType string, payload any) error {
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
