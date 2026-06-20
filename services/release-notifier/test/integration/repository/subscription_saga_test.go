//go:build integration

package repository_test

import (
	"context"
	"errors"
	"io"
	"log/slog"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionworkflow "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/workflow"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func (s *RepositorySuite) TestTransactionManager_RollsBack() {
	subscriber, err := s.subscriberStore.Create(s.ctx, "user@example.com")
	s.Require().NoError(err, "create subscriber")
	repository, err := s.repositoryStore.Create(s.ctx, "owner/repo", "v1.0.0")
	s.Require().NoError(err, "create repository")

	wantErr := errors.New("abort transaction")
	transactionManager := postgresqladapter.NewTransactionManager(s.pg.DB)
	err = transactionManager.Run(
		s.ctx,
		func(txCtx context.Context) error {
			_, createErr := s.subscriptionStore.Create(
				txCtx,
				subscriber.ID,
				repository.ID,
				"confirmation-token",
			)
			s.Require().NoError(createErr, "create subscription in transaction")
			return wantErr
		},
	)
	s.ErrorIs(err, wantErr)

	var count int
	s.Require().NoError(
		s.pg.DB.QueryRowContext(s.ctx, "SELECT COUNT(*) FROM subscriptions").Scan(&count),
		"count subscriptions",
	)
	s.Zero(count, "rolled back subscriptions")
}

func (s *RepositorySuite) TestTransactionManager_RejectsNestedTransaction() {
	transactionManager := postgresqladapter.NewTransactionManager(s.pg.DB)

	err := transactionManager.Run(s.ctx, func(txCtx context.Context) error {
		return transactionManager.Run(txCtx, func(context.Context) error {
			return nil
		})
	})

	s.ErrorIs(err, postgresqladapter.ErrTransactionAlreadyActive)
}

func (s *RepositorySuite) TestSubscriptionConfirmationSaga_CompensationIsAtomicAndIdempotent() {
	subscriber, err := s.subscriberStore.Create(s.ctx, "user@example.com")
	s.Require().NoError(err, "create subscriber")
	repository, err := s.repositoryStore.Create(s.ctx, "owner/repo", "v1.0.0")
	s.Require().NoError(err, "create repository")

	queryable := postgresqladapter.NewContextQueryable(s.pg.DB)
	transactionManager := postgresqladapter.NewTransactionManager(s.pg.DB)
	sagaStore := subscriptionpostgresql.NewSubscriptionSagaStore(queryable)
	outboxStore := subscriptionpostgresql.NewOutboxStore(queryable)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	saga := subscriptionworkflow.NewSubscriptionConfirmationSaga(
		logger,
		transactionManager,
		s.subscriptionStore,
		sagaStore,
		outboxStore,
	)
	err = saga.StartSubscriptionConfirmation(
		s.ctx,
		subscriber.ID,
		repository.ID,
		subscriber.Email,
		"confirmation-token",
	)
	s.Require().NoError(err, "start subscription confirmation")

	var sagaID, subscriptionID int
	s.Require().NoError(
		s.pg.DB.QueryRowContext(
			s.ctx,
			"SELECT id, subscription_id FROM subscription_sagas",
		).Scan(&sagaID, &subscriptionID),
		"get started saga",
	)

	event := events.SubscriptionConfirmationFailed{
		SagaID:         sagaID,
		SubscriptionID: subscriptionID,
		Reason:         "smtp unavailable",
	}
	s.Require().NoError(
		saga.HandleSubscriptionConfirmationFailed(s.ctx, event),
		"compensate subscription confirmation",
	)
	s.Require().NoError(
		saga.HandleSubscriptionConfirmationFailed(s.ctx, event),
		"repeat subscription compensation",
	)

	var subscriptionCount int
	s.Require().NoError(
		s.pg.DB.QueryRowContext(s.ctx, "SELECT COUNT(*) FROM subscriptions").Scan(&subscriptionCount),
		"count subscriptions",
	)
	s.Zero(subscriptionCount, "compensated subscription")

	var status domain.SagaStatus
	var reason string
	s.Require().NoError(
		s.pg.DB.QueryRowContext(
			s.ctx,
			"SELECT saga_status, failure_reason FROM subscription_sagas WHERE id = $1",
			sagaID,
		).Scan(&status, &reason),
		"get compensated saga",
	)
	s.Equal(domain.SagaStatusCompensated, status)
	s.Equal("smtp unavailable", reason)

	err = saga.HandleSubscriptionConfirmationSucceeded(
		s.ctx,
		events.SubscriptionConfirmationSucceeded{SagaID: sagaID},
	)
	s.ErrorIs(err, domain.ErrInvalidSagaTransition)
}
