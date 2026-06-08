package eventbus

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

type NotificationPublisher interface {
	PublishSubscriptionConfirmation(
		ctx context.Context,
		event events.SubscriptionConfirmationRequested,
	) error
	PublishReleaseNotification(
		ctx context.Context,
		event events.ReleaseNotificationRequested,
	) error
}

type NotificationHandler interface {
	HandleSubscriptionConfirmationRequested(
		ctx context.Context,
		event events.SubscriptionConfirmationRequested,
	) error
	HandleReleaseNotificationRequested(
		ctx context.Context,
		event events.ReleaseNotificationRequested,
	) error
}

type NotificationConsumer interface {
	Subscribe(ctx context.Context, handler NotificationHandler) error
}
