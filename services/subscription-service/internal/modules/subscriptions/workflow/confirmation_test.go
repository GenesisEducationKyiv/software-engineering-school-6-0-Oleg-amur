package workflow

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func TestSubscriptionConfirmationSagaStarts(t *testing.T) {
	subscriptions := &confirmationSubscriptionStore{id: 31}
	sagas := &confirmationSagaStore{saga: &domain.SubscriptionSaga{ID: 47}}
	outbox := &confirmationOutboxStore{}
	transactionManager := &confirmationTransactionManager{}

	err := NewSubscriptionConfirmationSaga(
		discardLogger(),
		transactionManager,
		subscriptions,
		sagas,
		outbox,
	).StartSubscriptionConfirmation(
		context.Background(),
		11,
		7,
		"user@example.com",
		"confirmation-token",
	)
	if err != nil {
		t.Fatalf("start confirmation: %v", err)
	}
	if transactionManager.calls != 1 {
		t.Fatalf("got %d transactions, want 1", transactionManager.calls)
	}
	if subscriptions.subscriberID != 11 || subscriptions.repositoryID != 7 {
		t.Fatalf("unexpected subscription input: %+v", subscriptions)
	}
	if sagas.createdForSubscriptionID != 31 {
		t.Fatalf("created saga for subscription %d, want 31", sagas.createdForSubscriptionID)
	}
	if outbox.eventType != events.SubscriptionConfirmationRequestedType {
		t.Fatalf("got event type %q", outbox.eventType)
	}
	event, ok := outbox.payload.(events.SubscriptionConfirmationRequested)
	if !ok {
		t.Fatalf("got outbox payload %T", outbox.payload)
	}
	if event.SagaID != 47 || event.SubscriptionID != 31 || event.Email != "user@example.com" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if event.EventID == "" || event.SchemaVersion != events.NotificationSchemaVersion {
		t.Fatalf("event metadata is missing: %+v", event)
	}
}

func TestSubscriptionConfirmationSagaStopsBeforeOutboxWhenSagaCreationFails(t *testing.T) {
	wantErr := errors.New("create saga")
	outbox := &confirmationOutboxStore{}
	transactionManager := &confirmationTransactionManager{}

	err := NewSubscriptionConfirmationSaga(
		discardLogger(),
		transactionManager,
		&confirmationSubscriptionStore{id: 31},
		&confirmationSagaStore{createErr: wantErr},
		outbox,
	).StartSubscriptionConfirmation(
		context.Background(),
		11,
		7,
		"user@example.com",
		"confirmation-token",
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	if outbox.createCalls != 0 {
		t.Fatalf("got %d outbox writes, want 0", outbox.createCalls)
	}
}

func TestSubscriptionConfirmationSagaCompletesSaga(t *testing.T) {
	sagas := &confirmationSagaStore{}
	workflow := NewSubscriptionConfirmationSaga(
		discardLogger(),
		&confirmationTransactionManager{},
		&confirmationSubscriptionStore{},
		sagas,
		&confirmationOutboxStore{},
	)

	err := workflow.HandleSubscriptionConfirmationSucceeded(
		context.Background(),
		events.SubscriptionConfirmationSucceeded{SagaID: 47},
	)
	if err != nil {
		t.Fatalf("complete saga: %v", err)
	}
	if sagas.completedSagaID != 47 {
		t.Fatalf("completed saga %d, want 47", sagas.completedSagaID)
	}
}

func TestSubscriptionConfirmationSagaCompensatesInTransaction(t *testing.T) {
	subscriptions := &confirmationSubscriptionStore{}
	sagas := &confirmationSagaStore{}
	transactionManager := &confirmationTransactionManager{}
	workflow := NewSubscriptionConfirmationSaga(
		discardLogger(),
		transactionManager,
		subscriptions,
		sagas,
		&confirmationOutboxStore{},
	)

	err := workflow.HandleSubscriptionConfirmationFailed(
		context.Background(),
		events.SubscriptionConfirmationFailed{
			SagaID:         47,
			SubscriptionID: 31,
			Reason:         "smtp unavailable",
		},
	)
	if err != nil {
		t.Fatalf("compensate saga: %v", err)
	}
	if transactionManager.calls != 1 {
		t.Fatalf("got %d transactions, want 1", transactionManager.calls)
	}
	if subscriptions.deletedID != 31 {
		t.Fatalf("deleted subscription %d, want 31", subscriptions.deletedID)
	}
	if sagas.compensatedSagaID != 47 || sagas.failureReason != "smtp unavailable" {
		t.Fatalf("unexpected compensation: %+v", sagas)
	}
}

type confirmationTransactionManager struct {
	calls int
}

func (m *confirmationTransactionManager) Run(ctx context.Context, work func(context.Context) error) error {
	m.calls++
	return work(ctx)
}

type confirmationSubscriptionStore struct {
	id           int
	subscriberID int
	repositoryID int64
	token        string
	deletedID    int
}

func (s *confirmationSubscriptionStore) Create(
	_ context.Context,
	subscriberID int,
	repositoryID int64,
	token string,
) (int, error) {
	s.subscriberID = subscriberID
	s.repositoryID = repositoryID
	s.token = token
	return s.id, nil
}

func (s *confirmationSubscriptionStore) DeleteByID(_ context.Context, subscriptionID int) error {
	s.deletedID = subscriptionID
	return nil
}

type confirmationSagaStore struct {
	saga                     *domain.SubscriptionSaga
	createErr                error
	createdForSubscriptionID int
	completedSagaID          int
	compensatedSagaID        int
	failureReason            string
}

func (s *confirmationSagaStore) CreateStarted(
	_ context.Context,
	subscriptionID int,
) (*domain.SubscriptionSaga, error) {
	s.createdForSubscriptionID = subscriptionID
	return s.saga, s.createErr
}

func (s *confirmationSagaStore) CompleteSubscriptionConfirmation(_ context.Context, sagaID int) error {
	s.completedSagaID = sagaID
	return nil
}

func (s *confirmationSagaStore) MarkCompensated(_ context.Context, sagaID int, reason string) error {
	s.compensatedSagaID = sagaID
	s.failureReason = reason
	return nil
}

type confirmationOutboxStore struct {
	createCalls int
	eventType   string
	payload     any
}

func (s *confirmationOutboxStore) Create(_ context.Context, eventType string, payload any) error {
	s.createCalls++
	s.eventType = eventType
	s.payload = payload
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
