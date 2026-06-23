//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	releasetrackerpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/persistence/postgresql"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/test/integration/testkit"
	"github.com/stretchr/testify/suite"
)

type RepositorySuite struct {
	suite.Suite

	pg               *testkit.Postgres
	ctx              context.Context
	cancel           context.CancelFunc
	repositoryRepo   *releasetrackerpostgresql.RepositoryStore
	subscriberRepo   *subscriptionpostgresql.SubscriberRepository
	subscriptionRepo *subscriptionpostgresql.SubscriptionRepository
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
	s.repositoryRepo = releasetrackerpostgresql.NewRepositoryStore(s.pg.DB)
	s.subscriberRepo = subscriptionpostgresql.NewSubscriberRepository(s.pg.DB)
	s.subscriptionRepo = subscriptionpostgresql.NewSubscriptionRepository(s.pg.DB)
}
