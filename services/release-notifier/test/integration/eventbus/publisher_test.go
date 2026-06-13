//go:build integration

package eventbus_test

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

func (s *EventBusSuite) TestNotificationPublisherPublishesEventsToRabbitMQ() {
	s.Run("subscription confirmation", func() {
		event := events.SubscriptionConfirmationRequested{
			EventID:       uuid.NewString(),
			SchemaVersion: events.NotificationSchemaVersion,
			Email:         "user@example.com",
			Token:         "confirm-token",
		}

		s.Require().NoError(s.publisher.PublishSubscriptionConfirmation(s.ctx, event))

		delivery := requireDelivery(s.ctx, s.T(), s.ch, s.cfg.Queue)
		requireMessageProperties(s.T(), delivery, events.SubscriptionConfirmationRequestedType, event.EventID)
		requireJSONPayload(s.T(), delivery.Body, event)
	})

	s.Run("release notification", func() {
		event := events.ReleaseNotificationRequested{
			EventID:          uuid.NewString(),
			SchemaVersion:    events.NotificationSchemaVersion,
			Email:            "user@example.com",
			Repo:             "owner/repo",
			Tag:              "v1.2.3",
			UnsubscribeToken: "unsubscribe-token",
		}

		s.Require().NoError(s.publisher.PublishReleaseNotification(s.ctx, event))

		delivery := requireDelivery(s.ctx, s.T(), s.ch, s.cfg.Queue)
		requireMessageProperties(s.T(), delivery, events.ReleaseNotificationRequestedType, event.EventID)
		requireJSONPayload(s.T(), delivery.Body, event)
	})
}
