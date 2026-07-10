//go:build integration

package repository_test

import (
	"errors"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func (s *RepositorySuite) TestOutboxStore_MarksFailedAfterAttemptLimit() {
	store := subscriptionpostgresql.NewOutboxStore(postgresqladapter.NewContextQueryable(s.pg.DB))
	s.Require().NoError(
		store.Create(
			s.ctx,
			events.SubscriptionConfirmationRequestedType,
			events.SubscriptionConfirmationRequested{EventID: "event-id"},
		),
		"create outbox message",
	)

	var id int
	s.Require().NoError(
		s.pg.DB.QueryRowContext(s.ctx, "SELECT id FROM outbox_messages").Scan(&id),
		"get outbox message ID",
	)

	const maxAttempts = 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		s.Require().NoError(
			store.RecordOutboxFailure(s.ctx, id, errors.New("RabbitMQ unavailable"), maxAttempts),
			"record outbox failure",
		)

		var status domain.OutboxStatus
		var attempts int
		s.Require().NoError(
			s.pg.DB.QueryRowContext(
				s.ctx,
				"SELECT outbox_status, attempts FROM outbox_messages WHERE id = $1",
				id,
			).Scan(&status, &attempts),
			"get outbox state",
		)
		s.Equal(attempt, attempts)
		if attempt < maxAttempts {
			s.Equal(domain.OutboxStatusPending, status)
		} else {
			s.Equal(domain.OutboxStatusFailed, status)
		}
	}

	messages, err := store.FetchPendingOutbox(s.ctx, 50)
	s.Require().NoError(err, "fetch pending outbox")
	s.Empty(messages, "failed outbox message must not be fetched again")
}

func (s *RepositorySuite) TestOutboxStore_PermanentFailureStopsImmediately() {
	store := subscriptionpostgresql.NewOutboxStore(postgresqladapter.NewContextQueryable(s.pg.DB))
	s.Require().NoError(store.Create(s.ctx, "unknown", map[string]string{}), "create outbox message")

	var id int
	s.Require().NoError(
		s.pg.DB.QueryRowContext(s.ctx, "SELECT id FROM outbox_messages").Scan(&id),
		"get outbox message ID",
	)
	s.Require().NoError(
		store.RecordOutboxFailure(s.ctx, id, errors.New("unsupported event type"), 1),
		"record permanent outbox failure",
	)

	var status domain.OutboxStatus
	var attempts int
	s.Require().NoError(
		s.pg.DB.QueryRowContext(
			s.ctx,
			"SELECT outbox_status, attempts FROM outbox_messages WHERE id = $1",
			id,
		).Scan(&status, &attempts),
		"get failed outbox state",
	)
	s.Equal(domain.OutboxStatusFailed, status)
	s.Equal(1, attempts)
}
