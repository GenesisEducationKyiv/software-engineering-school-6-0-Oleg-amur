package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type ScanRepositories struct {
	log           *slog.Logger
	repositories  repositoryStore
	github        githubClient
	subscriptions subscriptionClient
	publisher     notificationPublisher
}

func NewScanRepositories(
	log *slog.Logger,
	repositories repositoryStore,
	github githubClient,
	subscriptions subscriptionClient,
	publisher notificationPublisher,
) *ScanRepositories {
	return &ScanRepositories{
		log:           log,
		repositories:  repositories,
		github:        github,
		subscriptions: subscriptions,
		publisher:     publisher,
	}
}

func (u *ScanRepositories) Execute(ctx context.Context) {
	repositories, err := u.repositories.GetAll(ctx)
	if err != nil {
		u.log.Error("list repositories for scan", "err", err)
		return
	}
	for _, tracked := range repositories {
		if err := u.scanRepository(ctx, tracked); err != nil {
			u.log.Error("scan repository", "repository", tracked.Name, "err", err)
			if errors.Is(err, apperr.ErrRateLimitExceeded) {
				return
			}
		}
	}
}

func (u *ScanRepositories) scanRepository(ctx context.Context, tracked domain.Repository) error {
	tag, err := u.github.LatestTag(ctx, tracked.Name)
	if err != nil {
		return err
	}
	if tag == "" || tag == tracked.LastSeenTag {
		return nil
	}

	subscriptions, err := u.subscriptions.ListActiveByRepository(ctx, tracked.Name)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		event := events.ReleaseNotificationRequested{
			EventID:          uuid.NewString(),
			SchemaVersion:    events.NotificationSchemaVersion,
			Email:            subscription.Email,
			Repo:             tracked.Name,
			Tag:              tag,
			UnsubscribeToken: subscription.UnsubscribeToken,
		}
		if err := u.publisher.Publish(ctx, event); err != nil {
			return fmt.Errorf("publish release notification: %w", err)
		}
	}
	return u.repositories.UpdateTag(ctx, tracked.ID, tag)
}
