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

func TestHTTPSubscribe_CreatesPendingSubscription(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}

	postJSON(t, app.handler, "/api/v1/subscribe", subscribeRequest, http.StatusOK)

	event := receiveSubscriptionEvent(t, app.subscriptionEvents)
	if event.Email != subscribeRequest.Email {
		t.Fatalf("got subscription event email %q, want %q", event.Email, subscribeRequest.Email)
	}
	if event.Token == "" {
		t.Fatal("want subscription event token to be set")
	}

	assertSubscriptions(t, app.handler, subscribeRequest.Email, func(t *testing.T, response []dto.Subscription) {
		if len(response) != 0 {
			t.Fatalf("got %d active subscriptions before confirmation, want 0", len(response))
		}
	})
}

func TestHTTPConfirm_ActivatesSubscription(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	token := createSubscription(t, app, subscribeRequest)

	get(t, app.handler, "/api/v1/confirm/"+token, http.StatusOK, nil)

	assertSubscriptions(t, app.handler, subscribeRequest.Email, func(t *testing.T, response []dto.Subscription) {
		if len(response) != 1 {
			t.Fatalf("got %d active subscriptions after confirmation, want 1", len(response))
		}
	})
}

func TestHTTPGetSubscriptions_ReturnsActiveSubscription(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	token := createSubscription(t, app, subscribeRequest)
	get(t, app.handler, "/api/v1/confirm/"+token, http.StatusOK, nil)

	assertActiveSubscription(t, app.handler, dto.Subscription{
		Email:       subscribeRequest.Email,
		Repo:        subscribeRequest.Repo,
		Confirmed:   true,
		LastSeenTag: "v1.0.0",
	})
}

func TestHTTPUnsubscribe_RemovesSubscription(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	token := createSubscription(t, app, subscribeRequest)
	get(t, app.handler, "/api/v1/confirm/"+token, http.StatusOK, nil)

	get(t, app.handler, "/api/v1/unsubscribe/"+token, http.StatusOK, nil)

	assertSubscriptions(t, app.handler, subscribeRequest.Email, func(t *testing.T, response []dto.Subscription) {
		if len(response) != 0 {
			t.Fatalf("got %d active subscriptions after unsubscribe, want 0", len(response))
		}
	})
}

func TestHTTPSubscribe_ReturnsConflictForDuplicateSubscription(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	_ = createSubscription(t, app, subscribeRequest)

	postJSON(t, app.handler, "/api/v1/subscribe", subscribeRequest, http.StatusConflict)
}

func TestHTTPConfirm_ReturnsNotFoundForMissingToken(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()

	get(t, app.handler, "/api/v1/confirm/missing-token", http.StatusNotFound, nil)
}

func TestHTTPGetSubscriptions_ReturnsBadRequestForMissingEmail(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()

	get(t, app.handler, "/api/v1/subscriptions", http.StatusBadRequest, nil)
}

func TestHTTPSubscribe_ReturnsNotFoundForMissingGitHubRepository(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	app.github.Exists["missing/repo"] = false

	postJSON(t, app.handler, "/api/v1/subscribe", dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "missing/repo",
	}, http.StatusNotFound)
}

func TestHTTPSubscribe_ReturnsTooManyRequestsForGitHubRateLimit(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()
	app.github.CheckErr["owner/repo"] = apperr.ErrRateLimitExceeded

	postJSON(t, app.handler, "/api/v1/subscribe", dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}, http.StatusTooManyRequests)
}

func TestHTTPSubscribe_ReturnsBadRequestForInvalidInput(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()

	postJSON(t, app.handler, "/api/v1/subscribe", dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "invalid-repo",
	}, http.StatusBadRequest)
}

func TestHTTPSubscribe_ReturnsBadRequestForMalformedJSON(t *testing.T) {
	suite.ResetDatabase(t)

	app := newHTTPTestApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscribe", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d with body %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func createSubscription(t *testing.T, app *httpTestApp, req dto.SubscribeRequest) string {
	t.Helper()

	postJSON(t, app.handler, "/api/v1/subscribe", req, http.StatusOK)

	return receiveSubscriptionToken(t, app.subscriptionEvents)
}

func assertActiveSubscription(t *testing.T, handler http.Handler, want dto.Subscription) {
	t.Helper()

	assertSubscriptions(t, handler, want.Email, func(t *testing.T, response []dto.Subscription) {
		if len(response) != 1 {
			t.Fatalf("got %d active subscriptions, want 1", len(response))
		}
		got := response[0]
		if got.Email != want.Email {
			t.Fatalf("got email %q, want %q", got.Email, want.Email)
		}
		if got.Repo != want.Repo {
			t.Fatalf("got repo %q, want %q", got.Repo, want.Repo)
		}
		if got.Confirmed != want.Confirmed {
			t.Fatalf("got confirmed %t, want %t", got.Confirmed, want.Confirmed)
		}
		if got.LastSeenTag != want.LastSeenTag {
			t.Fatalf("got last seen tag %q, want %q", got.LastSeenTag, want.LastSeenTag)
		}
	})
}

func assertSubscriptions(
	t *testing.T,
	handler http.Handler,
	email string,
	assertResponse func(*testing.T, []dto.Subscription),
) {
	t.Helper()

	get(t, handler, "/api/v1/subscriptions?email="+email, http.StatusOK, func(t *testing.T, body []byte) {
		t.Helper()

		response := decodeSubscriptions(t, body)
		assertResponse(t, response)
	})
}

func decodeSubscriptions(t *testing.T, body []byte) []dto.Subscription {
	t.Helper()

	var response []dto.Subscription
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode subscriptions response: %v", err)
	}

	return response
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

func postJSON(t *testing.T, handler http.Handler, path string, body any, wantStatus int) []byte {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("got status %d, want %d with body %s", rec.Code, wantStatus, rec.Body.String())
	}

	return rec.Body.Bytes()
}

func get(
	t *testing.T,
	handler http.Handler,
	path string,
	wantStatus int,
	assertBody func(*testing.T, []byte),
) []byte {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("got status %d, want %d with body %s", rec.Code, wantStatus, rec.Body.String())
	}

	body := rec.Body.Bytes()
	if assertBody != nil {
		assertBody(t, body)
	}
	return body
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
