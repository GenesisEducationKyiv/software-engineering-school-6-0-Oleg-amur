package scanner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
)

type Scanner struct {
	log              *slog.Logger
	repoRepository   RepositoryRepo
	subscriptionRepo SubscriptionRepo
	githubClient     GithubClient
	notifier         Notifier
	interval         time.Duration
}

type RepositoryRepo interface {
	GetAll(ctx context.Context) ([]models.Repository, error)
	UpdateTag(ctx context.Context, id int, tag string) error
}

type SubscriptionRepo interface {
	GetActiveByRepoID(ctx context.Context, repoID int) ([]models.Subscription, error)
}

type Notifier interface {
	SendReleaseNotification(ctx context.Context, email, repo, tag string) error
}

type GithubClient interface {
	GetRepositoryLatestTag(ctx context.Context, repoAddr string, log *slog.Logger) (string, error)
}

func NewScanner(
	log *slog.Logger,
	repo RepositoryRepo,
	subscription SubscriptionRepo,
	gh GithubClient,
	notifier Notifier,
	interval time.Duration,
) *Scanner {
	return &Scanner{
		log:              log,
		repoRepository:   repo,
		subscriptionRepo: subscription,
		githubClient:     gh,
		notifier:         notifier,
		interval:         interval,
	}
}

func (s *Scanner) Start(ctx context.Context) {
	s.log.Info("background scanner started", "interval", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.Scan(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("background scanner stopping")
			return
		case <-ticker.C:
			s.Scan(ctx)
		}
	}
}

func (s *Scanner) Scan(ctx context.Context) {
	s.log.Debug("starting repository scan")

	repos, err := s.repoRepository.GetAll(ctx)
	if err != nil {
		s.log.Error("failed to fetch repositories from db", "err", err)
		return
	}

	for _, repo := range repos {
		stopScan, err := s.processRepo(ctx, repo)
		if err != nil {
			s.log.Error("failed to process repository", "repo", repo.Name, "err", err)
		}
		if stopScan {
			break
		}
	}
}

func (s *Scanner) processRepo(ctx context.Context, repo models.Repository) (bool, error) {
	latestTag, err := s.githubClient.GetRepositoryLatestTag(ctx, repo.Name, s.log)
	if err != nil {
		if errors.Is(err, apperr.ErrRateLimitExceeded) {
			s.log.Warn("rate limit reached", "error", err)
			return true, nil
		}
		return false, err
	}

	if latestTag == "" || repo.LastSeenTag == latestTag {
		return false, nil
	}

	s.log.Info("new release found", "repo", repo.Name, "old", repo.LastSeenTag, "new", latestTag)

	if err := s.repoRepository.UpdateTag(ctx, repo.ID, latestTag); err != nil {
		return false, fmt.Errorf("failed to update tag: %w", err)
	}

	subs, err := s.subscriptionRepo.GetActiveByRepoID(ctx, repo.ID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch subscribers: %w", err)
	}

	s.notifySubscribers(ctx, repo.Name, latestTag, subs)
	return false, nil
}

func (s *Scanner) notifySubscribers(ctx context.Context, repo, tag string, subs []models.Subscription) {
	for _, sub := range subs {
		s.log.Info("sending notification", "email", sub.Subscriber.Email, "repo", repo, "tag", tag)
		if err := s.notifier.SendReleaseNotification(ctx, sub.Subscriber.Email, repo, tag); err != nil {
			s.log.Error("failed to send notification", "email", sub.Subscriber.Email, "err", err)
		}
	}
}
