//go:build integration

package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/api/grpc/pb"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/test/integration/testkit"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const grpcBufferSize = 1024 * 1024

type GRPCSuite struct {
	suite.Suite

	ctx          context.Context
	cancel       context.CancelFunc
	pg           *testkit.Postgres
	githubServer *testkit.FakeGitHubServer
	app          *testkit.App
	client       pb.SubscriptionServiceClient
	cleanup      func()
}

func TestGRPCSuite(t *testing.T) {
	suite.Run(t, new(GRPCSuite))
}

func (s *GRPCSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	startCtx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()
	s.pg = testkit.NewPostgres(startCtx, s.T())
}

func (s *GRPCSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *GRPCSuite) SetupTest() {
	s.pg.Reset(s.T())
	s.githubServer = testkit.NewFakeGitHubServer(s.T())
	s.app = testkit.NewApp(s.T(), testkit.AppConfig{
		DB:        s.pg.DB,
		GitHubURL: s.githubServer.URL(),
	})
	s.client, s.cleanup = newGRPCTestClient(s.T(), s.app)
}

func (s *GRPCSuite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func newGRPCTestClient(
	t *testing.T,
	app *testkit.App,
) (
	pb.SubscriptionServiceClient,
	func(),
) {
	t.Helper()

	listener := bufconn.Listen(grpcBufferSize)
	server := grpc.NewServer()
	pb.RegisterSubscriptionServiceServer(server, app.GRPCHandler)

	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, address string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("create gRPC client: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
	}

	return pb.NewSubscriptionServiceClient(conn), cleanup
}
