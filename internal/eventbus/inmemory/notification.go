package inmemory

import (
	"context"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/contracts/events"
)

var ErrQueueFull = errors.New("in-memory event queue is full")

type NotificationPublisher struct {
	confirmations chan<- events.SubscriptionConfirmationRequested
	releases      chan<- events.ReleaseNotificationRequested
}

func NewNotificationPublisher(
	confirmations chan<- events.SubscriptionConfirmationRequested,
	releases chan<- events.ReleaseNotificationRequested,
) *NotificationPublisher {
	return &NotificationPublisher{
		confirmations: confirmations,
		releases:      releases,
	}
}

func (p *NotificationPublisher) PublishSubscriptionConfirmation(
	ctx context.Context,
	event events.SubscriptionConfirmationRequested,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.confirmations <- event:
		return nil
	default:
		return ErrQueueFull
	}
}

func (p *NotificationPublisher) PublishReleaseNotification(
	ctx context.Context,
	event events.ReleaseNotificationRequested,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.releases <- event:
		return nil
	default:
		return ErrQueueFull
	}
}
