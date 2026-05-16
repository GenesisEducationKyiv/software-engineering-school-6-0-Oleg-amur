//go:build integration

package http_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http/dto"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

func (s *HTTPSuite) createSubscription(req dto.SubscribeRequest) string {
	s.T().Helper()

	s.postJSON("/api/v1/subscribe", req, http.StatusOK)

	return s.receiveSubscriptionToken()
}

func (s *HTTPSuite) assertActiveSubscription(want dto.Subscription) {
	s.T().Helper()

	s.assertSubscriptions(want.Email, func(response []dto.Subscription) {
		s.Require().Len(response, 1, "active subscriptions")
		got := response[0]
		s.Equal(want.Email, got.Email)
		s.Equal(want.Repo, got.Repo)
		s.Equal(want.Confirmed, got.Confirmed)
		s.Equal(want.LastSeenTag, got.LastSeenTag)
	})
}

func (s *HTTPSuite) assertSubscriptions(email string, assertResponse func([]dto.Subscription)) {
	s.T().Helper()

	query := url.Values{}
	query.Set("email", email)

	s.get("/api/v1/subscriptions?"+query.Encode(), http.StatusOK, func(body []byte) {
		response := decodeSubscriptions(s.T(), body)
		assertResponse(response)
	})
}

func (s *HTTPSuite) assertSubscriptionStatus(token string, want int) {
	s.T().Helper()

	var got int
	err := s.app.DB.QueryRowContext(
		s.T().Context(),
		`SELECT subscription_status FROM subscriptions WHERE token = $1`,
		token,
	).Scan(&got)
	s.Require().NoError(err, "get subscription status by token")
	s.Equal(want, got)
}

func (s *HTTPSuite) postJSON(path string, body any, wantStatus int) []byte {
	s.T().Helper()

	payload, err := json.Marshal(body)
	s.Require().NoError(err, "marshal request body")

	return s.postRaw(path, payload, wantStatus)
}

func (s *HTTPSuite) postRaw(path string, body []byte, wantStatus int) []byte {
	s.T().Helper()

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.app.HTTPHandler.ServeHTTP(rec, req)

	s.Equal(wantStatus, rec.Code, rec.Body.String())

	return rec.Body.Bytes()
}

func (s *HTTPSuite) get(path string, wantStatus int, assertBody func([]byte)) []byte {
	s.T().Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()

	s.app.HTTPHandler.ServeHTTP(rec, req)

	s.Equal(wantStatus, rec.Code, rec.Body.String())

	body := rec.Body.Bytes()
	if assertBody != nil {
		assertBody(body)
	}
	return body
}

func (s *HTTPSuite) receiveSubscriptionEvent() model.SubscriptionEvent {
	s.T().Helper()

	select {
	case event := <-s.app.Events:
		return event
	default:
		s.T().Fatal("want subscription event to be queued")
	}

	return model.SubscriptionEvent{}
}

func (s *HTTPSuite) receiveSubscriptionToken() string {
	s.T().Helper()

	event := s.receiveSubscriptionEvent()
	s.Require().NotEmpty(event.Token, "subscription event token")
	return event.Token
}

func (s *HTTPSuite) assertSubscriptionMissing(token string) {
	s.T().Helper()

	var id int
	err := s.app.DB.QueryRowContext(
		s.T().Context(),
		`SELECT id FROM subscriptions WHERE token = $1`,
		token,
	).Scan(&id)
	s.Require().True(errors.Is(err, sql.ErrNoRows), "expected subscription to be deleted, got id=%d err=%v", id, err)
}

func decodeSubscriptions(t *testing.T, body []byte) []dto.Subscription {
	t.Helper()

	var response []dto.Subscription
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode subscriptions response: %v", err)
	}

	return response
}
