package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func TestPublishSubscriptionOutboxExecutePublishesValidMessage(t *testing.T) {
	event := events.SubscriptionConfirmationRequested{EventID: "event-id", Email: "user@example.com"}
	store := &outboxStoreMock{messages: []domain.OutboxMessage{
		{ID: 10, EventType: events.SubscriptionConfirmationRequestedType, Payload: marshalOutboxEvent(t, event)},
	}}
	publisher := &confirmationPublisherMock{}
	relay := NewPublishSubscriptionOutbox(outboxTestLogger(), store, publisher)

	relay.Execute(context.Background())

	if len(publisher.events) != 1 || publisher.events[0].EventID != event.EventID {
		t.Fatalf("published events = %+v, want event %q", publisher.events, event.EventID)
	}
	if len(store.publishedIDs) != 1 || store.publishedIDs[0] != 10 {
		t.Fatalf("published outbox IDs = %v, want [10]", store.publishedIDs)
	}
	if len(store.failures) != 0 {
		t.Fatalf("recorded failures = %+v, want none", store.failures)
	}
}

func TestPublishSubscriptionOutboxExecutePermanentlyFailsUnknownEvent(t *testing.T) {
	store := &outboxStoreMock{messages: []domain.OutboxMessage{
		{ID: 11, EventType: "subscription.unknown", Payload: json.RawMessage(`{}`)},
	}}
	publisher := &confirmationPublisherMock{}
	relay := NewPublishSubscriptionOutbox(outboxTestLogger(), store, publisher)

	relay.Execute(context.Background())

	assertSingleOutboxFailure(t, store, 11, 1)
	if len(publisher.events) != 0 {
		t.Fatalf("published %d events, want 0", len(publisher.events))
	}
}

func TestPublishSubscriptionOutboxExecutePermanentlyFailsInvalidPayload(t *testing.T) {
	store := &outboxStoreMock{messages: []domain.OutboxMessage{
		{ID: 12, EventType: events.SubscriptionConfirmationRequestedType, Payload: json.RawMessage(`{`)},
	}}
	relay := NewPublishSubscriptionOutbox(outboxTestLogger(), store, &confirmationPublisherMock{})

	relay.Execute(context.Background())

	assertSingleOutboxFailure(t, store, 12, 1)
}

func TestPublishSubscriptionOutboxExecuteRetriesPublishFailure(t *testing.T) {
	publishErr := errors.New("RabbitMQ unavailable")
	store := &outboxStoreMock{messages: []domain.OutboxMessage{
		{
			ID:        13,
			EventType: events.SubscriptionConfirmationRequestedType,
			Payload:   marshalOutboxEvent(t, events.SubscriptionConfirmationRequested{EventID: "event-id"}),
		},
	}}
	publisher := &confirmationPublisherMock{err: publishErr}
	relay := NewPublishSubscriptionOutbox(outboxTestLogger(), store, publisher)

	relay.Execute(context.Background())

	assertSingleOutboxFailure(t, store, 13, subscriptionOutboxMaxAttempts)
	if !errors.Is(store.failures[0].cause, publishErr) {
		t.Fatalf("recorded error = %v, want publish error", store.failures[0].cause)
	}
}

func assertSingleOutboxFailure(t *testing.T, store *outboxStoreMock, id, maxAttempts int) {
	t.Helper()

	if len(store.failures) != 1 {
		t.Fatalf("recorded %d failures, want 1", len(store.failures))
	}
	if store.failures[0].id != id || store.failures[0].maxAttempts != maxAttempts {
		t.Errorf("recorded failure = %+v, want id %d and max attempts %d", store.failures[0], id, maxAttempts)
	}
}

func marshalOutboxEvent(t *testing.T, event events.SubscriptionConfirmationRequested) json.RawMessage {
	t.Helper()

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return payload
}

func outboxTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type recordedOutboxFailure struct {
	id          int
	cause       error
	maxAttempts int
}

type outboxStoreMock struct {
	messages     []domain.OutboxMessage
	fetchErr     error
	publishedIDs []int
	publishErr   error
	failures     []recordedOutboxFailure
	failureErr   error
}

func (s *outboxStoreMock) FetchPendingOutbox(context.Context, int) ([]domain.OutboxMessage, error) {
	return s.messages, s.fetchErr
}

func (s *outboxStoreMock) MarkOutboxPublished(_ context.Context, id int) error {
	s.publishedIDs = append(s.publishedIDs, id)
	return s.publishErr
}

func (s *outboxStoreMock) RecordOutboxFailure(_ context.Context, id int, cause error, maxAttempts int) error {
	s.failures = append(s.failures, recordedOutboxFailure{id: id, cause: cause, maxAttempts: maxAttempts})
	return s.failureErr
}

type confirmationPublisherMock struct {
	events []events.SubscriptionConfirmationRequested
	err    error
}

func (p *confirmationPublisherMock) PublishSubscriptionConfirmation(
	_ context.Context,
	event events.SubscriptionConfirmationRequested,
) error {
	p.events = append(p.events, event)
	return p.err
}
