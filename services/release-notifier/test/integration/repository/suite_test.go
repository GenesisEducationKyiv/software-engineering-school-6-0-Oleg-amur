//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/repository/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/test/integration/testkit"
	"github.com/stretchr/testify/suite"
)

type RepositorySuite struct {
	suite.Suite

	pg               *testkit.Postgres
	ctx              context.Context
	cancel           context.CancelFunc
	repositoryRepo   *postgresql.RepositoryRepository
	subscriberRepo   *postgresql.SubscriberRepository
	subscriptionRepo *postgresql.SubscriptionRepository
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

func (s *RepositorySuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	startCtx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()
	s.pg = testkit.NewPostgres(startCtx, s.T())
}

func (s *RepositorySuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *RepositorySuite) SetupTest() {
	s.pg.Reset(s.T())
	s.repositoryRepo = postgresql.NewRepositoryRepository(s.pg.DB)
	s.subscriberRepo = postgresql.NewSubscriberRepository(s.pg.DB)
	s.subscriptionRepo = postgresql.NewSubscriptionRepository(s.pg.DB)
}
