//go:build integration

package http_test

import (
	"context"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/test/integration/testkit"
	"github.com/stretchr/testify/suite"
)

type HTTPSuite struct {
	suite.Suite

	ctx          context.Context
	cancel       context.CancelFunc
	pg           *testkit.Postgres
	githubServer *testkit.FakeGitHubServer
	app          *testkit.App
}

func TestHTTPSuite(t *testing.T) {
	suite.Run(t, new(HTTPSuite))
}

func (s *HTTPSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	startCtx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()
	s.pg = testkit.NewPostgres(startCtx, s.T())
}

func (s *HTTPSuite) SetupTest() {
	s.pg.Reset(s.T())
	s.githubServer = testkit.NewFakeGitHubServer(s.T())
	s.app = testkit.NewApp(s.T(), testkit.AppConfig{
		DB:        s.pg.DB,
		GitHubURL: s.githubServer.URL(),
	})
}

func (s *HTTPSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
}
