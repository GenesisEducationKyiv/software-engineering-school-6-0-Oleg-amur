//go:build integration

package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http/dto"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/test/integration/testkit"
)

var suite *testkit.Suite

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m, func(s *testkit.Suite) {
		suite = s
	}))
}

func TestHTTPSubscriptionFlow_Integration(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()

	postJSON(t, app.handler, "/api/v1/subscribe", dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}, http.StatusOK)

	token := receiveSubscriptionToken(t, app.subscriptionEvents)

	get(t, app.handler, "/api/v1/subscriptions?email=user@example.com", http.StatusOK, func(t *testing.T, body []byte) {
		t.Helper()

		var response []dto.Subscription
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("decode subscriptions response: %v", err)
		}
		if len(response) != 0 {
			t.Fatalf("expected no active subscriptions before confirmation, got %d", len(response))
		}
	})

	get(t, app.handler, "/api/v1/confirm/"+token, http.StatusOK, nil)

	get(t, app.handler, "/api/v1/subscriptions?email=user@example.com", http.StatusOK, func(t *testing.T, body []byte) {
		t.Helper()

		var response []dto.Subscription
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("decode subscriptions response: %v", err)
		}
		if len(response) != 1 {
			t.Fatalf("expected 1 active subscription after confirmation, got %d", len(response))
		}
		if response[0].Email != "user@example.com" {
			t.Fatalf("expected email user@example.com, got %s", response[0].Email)
		}
		if response[0].Repo != "owner/repo" {
			t.Fatalf("expected repo owner/repo, got %s", response[0].Repo)
		}
		if !response[0].Confirmed {
			t.Fatal("expected subscription to be confirmed")
		}
		if response[0].LastSeenTag != "v1.0.0" {
			t.Fatalf("expected last seen tag v1.0.0, got %s", response[0].LastSeenTag)
		}
	})

	postJSON(t, app.handler, "/api/v1/subscribe", dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}, http.StatusConflict)

	get(t, app.handler, "/api/v1/unsubscribe/"+token, http.StatusOK, nil)

	get(t, app.handler, "/api/v1/subscriptions?email=user@example.com", http.StatusOK, func(t *testing.T, body []byte) {
		t.Helper()

		var response []dto.Subscription
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("decode subscriptions response: %v", err)
		}
		if len(response) != 0 {
			t.Fatalf("expected no active subscriptions after unsubscribe, got %d", len(response))
		}
	})
}

func TestHTTPSubscribe_ReturnsNotFoundForMissingGitHubRepository_Integration(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	app.github.Exists["missing/repo"] = false

	postJSON(t, app.handler, "/api/v1/subscribe", dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "missing/repo",
	}, http.StatusNotFound)
}

func TestHTTPSubscribe_ReturnsTooManyRequestsForGitHubRateLimit_Integration(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	app.github.CheckErr["owner/repo"] = apperr.ErrRateLimitExceeded

	postJSON(t, app.handler, "/api/v1/subscribe", dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}, http.StatusTooManyRequests)
}

func TestHTTPSubscribe_ReturnsBadRequestForInvalidInput_Integration(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()

	postJSON(t, app.handler, "/api/v1/subscribe", dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "invalid-repo",
	}, http.StatusBadRequest)
}

type httpTestApp struct {
	handler            http.Handler
	github             *testkit.FakeGithubClient
	subscriptionEvents chan model.SubscriptionEvent
}

func newHTTPTestApp() *httpTestApp {
	subscriptionService, githubClient, subscriptionEvents := suite.NewSubscriptionService()

	return &httpTestApp{
		handler:            httpapi.NewRouter(suite.Logger, subscriptionService),
		github:             githubClient,
		subscriptionEvents: subscriptionEvents,
	}
}

func postJSON(t *testing.T, handler http.Handler, path string, body any, expectedStatus int) []byte {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d with body %s", expectedStatus, rec.Code, rec.Body.String())
	}

	return rec.Body.Bytes()
}

func get(
	t *testing.T,
	handler http.Handler,
	path string,
	expectedStatus int,
	assertBody func(*testing.T, []byte),
) []byte {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d with body %s", expectedStatus, rec.Code, rec.Body.String())
	}

	body := rec.Body.Bytes()
	if assertBody != nil {
		assertBody(t, body)
	}
	return body
}

func receiveSubscriptionToken(t *testing.T, events <-chan model.SubscriptionEvent) string {
	t.Helper()

	select {
	case event := <-events:
		if event.Token == "" {
			t.Fatal("expected subscription event token to be set")
		}
		return event.Token
	default:
		t.Fatal("expected subscription event to be queued")
	}

	return ""
}
