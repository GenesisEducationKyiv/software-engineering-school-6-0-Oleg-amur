package scanner

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/apperr"
	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/models"
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
		latestTag, err := s.githubClient.GetRepositoryLatestTag(ctx, repo.Name, s.log)
		if err != nil {
			if errors.Is(err, apperr.ErrRateLimitExceeded) {
				s.log.Warn("rate limit reached", "error", err)
				break
			}
			s.log.Error("failed to get latest release from github", "repo", repo.Name, "err", err)
			continue
		}

		if latestTag == "" {
			continue
		}

		if repo.LastSeenTag != latestTag {
			s.log.Info("new release found", "repo", repo.Name, "old", repo.LastSeenTag, "new", latestTag)

			if err := s.repoRepository.UpdateTag(ctx, repo.ID, latestTag); err != nil {
				s.log.Error("failed to update last_seen_tag", "repo", repo.Name, "err", err)
				continue
			}

			subs, err := s.subscriptionRepo.GetActiveByRepoID(ctx, repo.ID)
			if err != nil {
				s.log.Error("failed to fetch subscribers for repo", "repo", repo.Name, "err", err)
				continue
			}

			for _, sub := range subs {
				s.log.Info("sending release notification", "email", sub.Subscriber.Email, "repo", repo.Name, "tag", latestTag)
				if err := s.notifier.SendReleaseNotification(ctx, sub.Subscriber.Email, repo.Name, latestTag); err != nil {
					s.log.Error("failed to send notification", "email", sub.Subscriber.Email, "err", err)
				}
			}
		}
	}
}
