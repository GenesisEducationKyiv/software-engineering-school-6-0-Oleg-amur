package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestConsumerHandleDelivery_AcksHandledEvents(t *testing.T) {
	tests := []struct {
		name       string
		routingKey string
		body       []byte
		assert     func(t *testing.T, handler *mockNotificationHandler)
	}{
		{
			name:       "subscription confirmation",
			routingKey: events.SubscriptionConfirmationRequestedType,
			body: mustMarshal(t, events.SubscriptionConfirmationRequested{
				SchemaVersion:     events.NotificationSchemaVersion,
				Email:             "user@example.com",
				ConfirmationToken: "confirm-token",
			}),
			assert: func(t *testing.T, handler *mockNotificationHandler) {
				t.Helper()

				if !handler.confirmationCalled {
					t.Fatal("expected subscription confirmation handler to be called")
				}
				if handler.confirmation.Email != "user@example.com" {
					t.Errorf("got email %q, want user@example.com", handler.confirmation.Email)
				}
				if handler.confirmation.ConfirmationToken != "confirm-token" {
					t.Errorf(
						"got confirmation token %q, want confirm-token",
						handler.confirmation.ConfirmationToken,
					)
				}
			},
		},
		{
			name:       "release notification",
			routingKey: events.ReleaseNotificationRequestedType,
			body: mustMarshal(t, events.ReleaseNotificationRequested{
				SchemaVersion:    events.NotificationSchemaVersion,
				Email:            "user@example.com",
				Repo:             "owner/repo",
				Tag:              "v1.0.0",
				UnsubscribeToken: "unsubscribe-token",
			}),
			assert: func(t *testing.T, handler *mockNotificationHandler) {
				t.Helper()

				if !handler.releaseCalled {
					t.Fatal("expected release notification handler to be called")
				}
				if handler.release.Repo != "owner/repo" {
					t.Errorf("got repo %q, want owner/repo", handler.release.Repo)
				}
				if handler.release.Tag != "v1.0.0" {
					t.Errorf("got tag %q, want v1.0.0", handler.release.Tag)
				}
				if handler.release.UnsubscribeToken != "unsubscribe-token" {
					t.Errorf(
						"got unsubscribe token %q, want unsubscribe-token",
						handler.release.UnsubscribeToken,
					)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := &Consumer{log: testLogger()}
			handler := &mockNotificationHandler{}
			acknowledger := &mockAcknowledger{}

			consumer.handleDelivery(context.Background(), handler, amqp.Delivery{
				Acknowledger: acknowledger,
				DeliveryTag:  10,
				RoutingKey:   tt.routingKey,
				Body:         tt.body,
			})

			assertAcked(t, acknowledger, 10)
			assertNotNacked(t, acknowledger)
			tt.assert(t, handler)
		})
	}
}

func TestConsumerHandleDelivery_NacksFailedEvents(t *testing.T) {
	handlerErr := errors.New("smtp unavailable")

	tests := []struct {
		name       string
		routingKey string
		body       []byte
		handlerErr error
	}{
		{
			name:       "malformed json",
			routingKey: events.SubscriptionConfirmationRequestedType,
			body:       []byte("{"),
		},
		{
			name:       "unknown routing key",
			routingKey: "notification.unknown",
			body:       []byte("{}"),
		},
		{
			name:       "handler error",
			routingKey: events.SubscriptionConfirmationRequestedType,
			body: mustMarshal(t, events.SubscriptionConfirmationRequested{
				SchemaVersion:     events.NotificationSchemaVersion,
				Email:             "user@example.com",
				ConfirmationToken: "confirm-token",
			}),
			handlerErr: handlerErr,
		},
		{
			name:       "unsupported schema version",
			routingKey: events.SubscriptionConfirmationRequestedType,
			body: mustMarshal(t, events.SubscriptionConfirmationRequested{
				SchemaVersion:     events.NotificationSchemaVersion + 1,
				Email:             "user@example.com",
				ConfirmationToken: "confirm-token",
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := &Consumer{log: testLogger()}
			handler := &mockNotificationHandler{err: tt.handlerErr}
			acknowledger := &mockAcknowledger{}

			consumer.handleDelivery(context.Background(), handler, amqp.Delivery{
				Acknowledger: acknowledger,
				DeliveryTag:  20,
				RoutingKey:   tt.routingKey,
				Body:         tt.body,
			})

			assertNotAcked(t, acknowledger)
			assertNackedWithoutRequeue(t, acknowledger, 20)
		})
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return body
}

func assertAcked(t *testing.T, acknowledger *mockAcknowledger, tag uint64) {
	t.Helper()

	if len(acknowledger.acked) != 1 {
		t.Fatalf("got %d acknowledgements, want 1", len(acknowledger.acked))
	}
	if acknowledger.acked[0] != tag {
		t.Errorf("got ack tag %d, want %d", acknowledger.acked[0], tag)
	}
}

func assertNotAcked(t *testing.T, acknowledger *mockAcknowledger) {
	t.Helper()

	if len(acknowledger.acked) != 0 {
		t.Fatalf("got %d acknowledgements, want 0", len(acknowledger.acked))
	}
}

func assertNackedWithoutRequeue(t *testing.T, acknowledger *mockAcknowledger, tag uint64) {
	t.Helper()

	if len(acknowledger.nacked) != 1 {
		t.Fatalf("got %d negative acknowledgements, want 1", len(acknowledger.nacked))
	}
	got := acknowledger.nacked[0]
	if got.tag != tag {
		t.Errorf("got nack tag %d, want %d", got.tag, tag)
	}
	if got.multiple {
		t.Error("got nack multiple=true, want false")
	}
	if got.requeue {
		t.Error("got nack requeue=true, want false")
	}
}

func assertNotNacked(t *testing.T, acknowledger *mockAcknowledger) {
	t.Helper()

	if len(acknowledger.nacked) != 0 {
		t.Fatalf("got %d negative acknowledgements, want 0", len(acknowledger.nacked))
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockNotificationHandler struct {
	confirmation       events.SubscriptionConfirmationRequested
	confirmationCalled bool
	release            events.ReleaseNotificationRequested
	releaseCalled      bool
	err                error
}

func (h *mockNotificationHandler) HandleSubscriptionConfirmationRequested(
	ctx context.Context,
	event events.SubscriptionConfirmationRequested,
) error {
	h.confirmation = event
	h.confirmationCalled = true
	return h.err
}

func (h *mockNotificationHandler) HandleReleaseNotificationRequested(
	ctx context.Context,
	event events.ReleaseNotificationRequested,
) error {
	h.release = event
	h.releaseCalled = true
	return h.err
}

type nackCall struct {
	tag      uint64
	multiple bool
	requeue  bool
}

type mockAcknowledger struct {
	acked  []uint64
	nacked []nackCall
}

func (a *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	a.acked = append(a.acked, tag)
	return nil
}

func (a *mockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	a.nacked = append(a.nacked, nackCall{
		tag:      tag,
		multiple: multiple,
		requeue:  requeue,
	})
	return nil
}

func (a *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	return nil
}
