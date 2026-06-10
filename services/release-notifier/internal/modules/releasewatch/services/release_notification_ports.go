package services

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasewatch/models"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

type subscriptionRepoForReleaseNotifications interface {
	GetActiveRecipientsByRepoID(ctx context.Context, repoID int) ([]models.NotificationRecipient, error)
}

type releaseNotificationPublisher interface {
	PublishReleaseNotification(ctx context.Context, event events.ReleaseNotificationRequested) error
}
