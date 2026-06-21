package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/domain"
	githubclient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/repository"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type RepositoryStore interface {
	Create(context.Context, string, string) (*domain.Repository, error)
	GetByName(context.Context, string) (*domain.Repository, error)
	GetAll(context.Context) ([]domain.Repository, error)
	UpdateTag(context.Context, int, string) error
}

type GitHub interface {
	RepositoryExists(context.Context, string) (bool, error)
	LatestTag(context.Context, string) (string, error)
}

type Subscriptions interface {
	ListActiveByRepository(context.Context, string) ([]domain.ActiveSubscription, error)
}

type NotificationPublisher interface {
	Publish(context.Context, events.ReleaseNotificationRequested) error
}

type Service struct {
	log           *slog.Logger
	repositories  RepositoryStore
	github        GitHub
	subscriptions Subscriptions
	publisher     NotificationPublisher
}

func New(
	log *slog.Logger,
	repositories RepositoryStore,
	github GitHub,
	subscriptions Subscriptions,
	publisher NotificationPublisher,
) *Service {
	return &Service{
		log:           log,
		repositories:  repositories,
		github:        github,
		subscriptions: subscriptions,
		publisher:     publisher,
	}
}

func (s *Service) EnsureTracked(ctx context.Context, name string) (*domain.Repository, error) {
	tracked, err := s.repositories.GetByName(ctx, name)
	if err == nil {
		return tracked, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	exists, err := s.github.RepositoryExists(ctx, name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, githubclient.ErrNotFound
	}

	tag, err := s.github.LatestTag(ctx, name)
	if err != nil && !errors.Is(err, githubclient.ErrNotFound) {
		return nil, err
	}
	return s.repositories.Create(ctx, name, tag)
}

func (s *Service) GetRepository(ctx context.Context, name string) (*domain.Repository, error) {
	return s.repositories.GetByName(ctx, name)
}

func (s *Service) Scan(ctx context.Context) {
	repositories, err := s.repositories.GetAll(ctx)
	if err != nil {
		s.log.Error("list repositories for scan", "err", err)
		return
	}
	for _, tracked := range repositories {
		if err := s.scanRepository(ctx, tracked); err != nil {
			s.log.Error("scan repository", "repository", tracked.Name, "err", err)
			if errors.Is(err, githubclient.ErrRateLimit) {
				return
			}
		}
	}
}

func (s *Service) scanRepository(ctx context.Context, tracked domain.Repository) error {
	tag, err := s.github.LatestTag(ctx, tracked.Name)
	if err != nil {
		return err
	}
	if tag == "" || tag == tracked.LastSeenTag {
		return nil
	}

	subscriptions, err := s.subscriptions.ListActiveByRepository(ctx, tracked.Name)
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
		if err := s.publisher.Publish(ctx, event); err != nil {
			return fmt.Errorf("publish release notification: %w", err)
		}
	}
	return s.repositories.UpdateTag(ctx, tracked.ID, tag)
}

func (s *Service) RunScheduler(ctx context.Context, interval time.Duration) {
	s.Scan(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Scan(ctx)
		}
	}
}
