package scanner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

type Scanner struct {
	log            *slog.Logger
	repoRepository RepositoryRepo
	githubClient   GithubClient
	releaseHandler ReleaseDetectedHandler
}

type RepositoryRepo interface {
	GetAll(ctx context.Context) ([]model.Repository, error)
	UpdateTag(ctx context.Context, id int, tag string) error
}

type GithubClient interface {
	GetRepositoryLatestTag(ctx context.Context, repoAddr string) (string, error)
}

type ReleaseDetectedHandler interface {
	HandleReleaseDetected(ctx context.Context, event model.ReleaseEvent) error
}

func NewScanner(
	log *slog.Logger,
	repo RepositoryRepo,
	gh GithubClient,
	releaseHandler ReleaseDetectedHandler,
) *Scanner {
	return &Scanner{
		log:            log,
		repoRepository: repo,
		githubClient:   gh,
		releaseHandler: releaseHandler,
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
		err := s.processRepo(ctx, repo)
		if err != nil {
			if errors.Is(err, apperr.ErrRateLimitExceeded) {
				s.log.Warn("rate limit reached, stopping scan", "error", err)
				break
			}
			s.log.Error("failed to process repository", "repo", repo.Name, "err", err)
		}
	}
}

func (s *Scanner) processRepo(ctx context.Context, repo model.Repository) error {
	latestTag, err := s.githubClient.GetRepositoryLatestTag(ctx, repo.Name)
	if err != nil {
		return err
	}

	if latestTag == "" || repo.LastSeenTag == latestTag {
		return nil
	}

	s.log.Info("new release found", "repo", repo.Name, "old", repo.LastSeenTag, "new", latestTag)

	event := model.ReleaseEvent{
		RepoID:   repo.ID,
		RepoName: repo.Name,
		Tag:      latestTag,
	}
	if err := s.releaseHandler.HandleReleaseDetected(ctx, event); err != nil {
		return fmt.Errorf("failed to handle release notification: %w", err)
	}

	if err := s.repoRepository.UpdateTag(ctx, repo.ID, latestTag); err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	return nil
}
