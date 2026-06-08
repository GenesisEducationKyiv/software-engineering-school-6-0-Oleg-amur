//go:build integration

package repository_test

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
)

func (s *RepositorySuite) TestRepositoryRepository_CreateAndGetByName() {
	created, err := s.repositoryRepo.Create(s.ctx, "owner/repo", "v1.0.0")
	s.Require().NoError(err, "create repository")
	s.NotZero(created.ID, "created repository id")
	s.Equal("owner/repo", created.Name)
	s.Equal("v1.0.0", created.LastSeenTag)

	got, err := s.repositoryRepo.GetByName(s.ctx, "owner/repo")
	s.Require().NoError(err, "get repository by name")
	s.Equal(created.ID, got.ID)
}

func (s *RepositorySuite) TestRepositoryRepository_UpdateTag() {
	created, err := s.repositoryRepo.Create(s.ctx, "owner/repo", "v1.0.0")
	s.Require().NoError(err, "create repository")

	err = s.repositoryRepo.UpdateTag(s.ctx, created.ID, "v2.0.0")
	s.Require().NoError(err, "update repository tag")
	updated, err := s.repositoryRepo.GetByName(s.ctx, "owner/repo")
	s.Require().NoError(err, "get updated repository by name")
	s.Equal("v2.0.0", updated.LastSeenTag)
}

func (s *RepositorySuite) TestRepositoryRepository_GetAll() {
	_, err := s.repositoryRepo.Create(s.ctx, "owner/repo", "v1.0.0")
	s.Require().NoError(err, "create repository")

	all, err := s.repositoryRepo.GetAll(s.ctx)
	s.Require().NoError(err, "get all repositories")
	s.Require().Len(all, 1, "repositories")
	s.Equal("owner/repo", all[0].Name)
}

func (s *RepositorySuite) TestRepositoryRepository_GetByNameReturnsNotFound() {
	_, err := s.repositoryRepo.GetByName(s.ctx, "missing/repo")

	s.ErrorIs(err, apperr.ErrNotFound)
}
