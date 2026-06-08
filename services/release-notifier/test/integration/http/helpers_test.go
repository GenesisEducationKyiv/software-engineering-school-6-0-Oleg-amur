//go:build integration

package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/http/dto"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

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

func (s *HTTPSuite) postSubscribe(req dto.SubscribeRequest, wantStatus int) []byte {
	s.T().Helper()

	return s.postJSON("/api/v1/subscribe", req, wantStatus)
}

func (s *HTTPSuite) postSubscribeRaw(body []byte, wantStatus int) []byte {
	s.T().Helper()

	return s.postRaw("/api/v1/subscribe", body, wantStatus)
}

func (s *HTTPSuite) getConfirm(token string, wantStatus int) []byte {
	s.T().Helper()

	return s.get("/api/v1/confirm/"+token, wantStatus, nil)
}

func (s *HTTPSuite) getUnsubscribe(token string, wantStatus int) []byte {
	s.T().Helper()

	return s.get("/api/v1/unsubscribe/"+token, wantStatus, nil)
}

func (s *HTTPSuite) getSubscriptions(email string) []dto.Subscription {
	s.T().Helper()

	query := url.Values{}
	query.Set("email", email)

	body := s.get("/api/v1/subscriptions?"+query.Encode(), http.StatusOK, nil)
	return decodeSubscriptions(s.T(), body)
}

func (s *HTTPSuite) getSubscriptionsWithoutEmail(wantStatus int) []byte {
	s.T().Helper()

	return s.get("/api/v1/subscriptions", wantStatus, nil)
}

func (s *HTTPSuite) receiveSubscriptionEvent() events.SubscriptionConfirmationRequested {
	s.T().Helper()

	select {
	case event := <-s.app.Events:
		return event
	default:
		s.T().Fatal("want subscription event to be queued")
	}

	return events.SubscriptionConfirmationRequested{}
}

func (s *HTTPSuite) receiveSubscriptionToken() string {
	s.T().Helper()

	event := s.receiveSubscriptionEvent()
	s.Require().NotEmpty(event.Token, "subscription event token")
	return event.Token
}

func decodeSubscriptions(t *testing.T, body []byte) []dto.Subscription {
	t.Helper()

	var response []dto.Subscription
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode subscriptions response: %v", err)
	}

	return response
}
