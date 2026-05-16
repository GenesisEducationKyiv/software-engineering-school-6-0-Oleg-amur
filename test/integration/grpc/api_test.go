//go:build integration

package grpc_test

import (
	"context"
	"net"
	"os"
	"testing"

	grpcapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/grpc/pb"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/test/integration/testkit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const grpcBufferSize = 1024 * 1024

var suite *testkit.Suite

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m, func(s *testkit.Suite) {
		suite = s
	}))
}

func TestGRPCSubscribe_CreatesPendingSubscription(t *testing.T) {
	suite.ResetDatabase(t)

	client, events, cleanup := newGRPCTestClient(t)
	defer cleanup()

	_, err := client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	})
	assertGRPCCode(t, err, codes.OK)

	event := receiveSubscriptionEvent(t, events)
	if event.Email != "user@example.com" {
		t.Fatalf("got subscription event email %q, want %q", event.Email, "user@example.com")
	}
	if event.Token == "" {
		t.Fatal("want subscription event token to be set")
	}

	resp, err := client.GetSubscriptions(context.Background(), &pb.GetSubscriptionsRequest{
		Email: "user@example.com",
	})
	assertGRPCCode(t, err, codes.OK)
	if len(resp.GetSubscriptions()) != 0 {
		t.Fatalf("got %d active subscriptions before confirmation, want 0", len(resp.GetSubscriptions()))
	}
}

func TestGRPCGetSubscriptions_ReturnsActiveSubscription(t *testing.T) {
	suite.ResetDatabase(t)

	client, events, cleanup := newGRPCTestClient(t)
	defer cleanup()

	_, err := client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	})
	assertGRPCCode(t, err, codes.OK)
	token := receiveSubscriptionToken(t, events)

	_, err = client.Confirm(context.Background(), &pb.ConfirmRequest{Token: token})
	assertGRPCCode(t, err, codes.OK)

	resp, err := client.GetSubscriptions(context.Background(), &pb.GetSubscriptionsRequest{
		Email: "user@example.com",
	})
	assertGRPCCode(t, err, codes.OK)
	if len(resp.GetSubscriptions()) != 1 {
		t.Fatalf("got %d active subscriptions, want 1", len(resp.GetSubscriptions()))
	}

	sub := resp.GetSubscriptions()[0]
	if sub.GetEmail() != "user@example.com" {
		t.Fatalf("got email %q, want %q", sub.GetEmail(), "user@example.com")
	}
	if sub.GetRepo() != "owner/repo" {
		t.Fatalf("got repo %q, want %q", sub.GetRepo(), "owner/repo")
	}
	if !sub.GetConfirmed() {
		t.Fatal("want subscription to be confirmed")
	}
	if sub.GetLastSeenTag() != "v1.0.0" {
		t.Fatalf("got last seen tag %q, want %q", sub.GetLastSeenTag(), "v1.0.0")
	}
}

func TestGRPCUnsubscribe_RemovesSubscription(t *testing.T) {
	suite.ResetDatabase(t)

	client, events, cleanup := newGRPCTestClient(t)
	defer cleanup()

	_, err := client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	})
	assertGRPCCode(t, err, codes.OK)
	token := receiveSubscriptionToken(t, events)

	_, err = client.Unsubscribe(context.Background(), &pb.UnsubscribeRequest{Token: token})
	assertGRPCCode(t, err, codes.OK)

	afterUnsubscribe, err := client.GetSubscriptions(context.Background(), &pb.GetSubscriptionsRequest{
		Email: "user@example.com",
	})
	assertGRPCCode(t, err, codes.OK)
	if len(afterUnsubscribe.GetSubscriptions()) != 0 {
		t.Fatalf("got %d active subscriptions after unsubscribe, want 0", len(afterUnsubscribe.GetSubscriptions()))
	}
}

func TestGRPCSubscribe_ReturnsTransportErrors(t *testing.T) {
	tests := []struct {
		name      string
		mutateApp func(*testkit.FakeGithubClient)
		req       *pb.SubscribeRequest
		wantCode  codes.Code
	}{
		{
			name:     "invalid repo format",
			req:      &pb.SubscribeRequest{Email: "user@example.com", Repo: "invalid"},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing GitHub repository",
			mutateApp: func(github *testkit.FakeGithubClient) {
				github.Exists["missing/repo"] = false
			},
			req:      &pb.SubscribeRequest{Email: "user@example.com", Repo: "missing/repo"},
			wantCode: codes.NotFound,
		},
		{
			name: "GitHub rate limit",
			mutateApp: func(github *testkit.FakeGithubClient) {
				github.CheckErr["owner/repo"] = apperr.ErrRateLimitExceeded
			},
			req:      &pb.SubscribeRequest{Email: "user@example.com", Repo: "owner/repo"},
			wantCode: codes.ResourceExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite.ResetDatabase(t)

			client, githubClient, cleanup := newGRPCTestClientWithGithub(t)
			defer cleanup()
			if tt.mutateApp != nil {
				tt.mutateApp(githubClient)
			}

			_, err := client.Subscribe(context.Background(), tt.req)

			assertGRPCCode(t, err, tt.wantCode)
		})
	}
}

func TestGRPCConfirm_ReturnsNotFoundForMissingToken(t *testing.T) {
	suite.ResetDatabase(t)

	client, _, cleanup := newGRPCTestClient(t)
	defer cleanup()

	_, err := client.Confirm(context.Background(), &pb.ConfirmRequest{Token: "missing-token"})

	assertGRPCCode(t, err, codes.NotFound)
}

func receiveSubscriptionEvent(t *testing.T, events <-chan model.SubscriptionEvent) model.SubscriptionEvent {
	t.Helper()

	select {
	case event := <-events:
		return event
	default:
		t.Fatal("want subscription event to be queued")
	}

	return model.SubscriptionEvent{}
}

func receiveSubscriptionToken(t *testing.T, events <-chan model.SubscriptionEvent) string {
	t.Helper()

	event := receiveSubscriptionEvent(t, events)
	if event.Token == "" {
		t.Fatal("want subscription event token to be set")
	}
	return event.Token
}

func newGRPCTestClient(t *testing.T) (
	pb.ReleaseNotifierClient,
	<-chan model.SubscriptionEvent,
	func(),
) {
	client, _, events, cleanup := newGRPCTestClientFull(t)
	return client, events, cleanup
}

func newGRPCTestClientWithGithub(t *testing.T) (
	pb.ReleaseNotifierClient,
	*testkit.FakeGithubClient,
	func(),
) {
	client, githubClient, _, cleanup := newGRPCTestClientFull(t)
	return client, githubClient, cleanup
}

func newGRPCTestClientFull(t *testing.T) (
	pb.ReleaseNotifierClient,
	*testkit.FakeGithubClient,
	chan model.SubscriptionEvent,
	func(),
) {
	t.Helper()

	subscriptionService, githubClient, events := suite.NewSubscriptionService()
	handler := grpcapi.NewGrpcHandler(suite.Logger, subscriptionService)

	listener := bufconn.Listen(grpcBufferSize)
	server := grpc.NewServer()
	pb.RegisterReleaseNotifierServer(server, handler)

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

	return pb.NewReleaseNotifierClient(conn), githubClient, events, cleanup
}

func assertGRPCCode(t *testing.T, gotErr error, wantCode codes.Code) {
	t.Helper()

	if wantCode == codes.OK {
		if gotErr != nil {
			t.Fatalf("got gRPC error %v, want nil", gotErr)
		}
		return
	}
	if gotCode := status.Code(gotErr); gotCode != wantCode {
		t.Fatalf("got gRPC code %s, want %s with error %v", gotCode, wantCode, gotErr)
	}
}
