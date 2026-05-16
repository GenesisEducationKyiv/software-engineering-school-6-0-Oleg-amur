//go:build integration

package http_test

import (
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http/dto"
)

func (s *HTTPSuite) TestSubscribe_CreatesPendingSubscription() {
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}

	s.postSubscribe(subscribeRequest, http.StatusOK)

	event := s.receiveSubscriptionEvent()
	s.Equal(subscribeRequest.Email, event.Email)
	s.NotEmpty(event.Token)

	response := s.getSubscriptions(subscribeRequest.Email)
	s.Empty(response, "active subscriptions before confirmation")
}

func (s *HTTPSuite) TestSubscribe_ReturnsConflictForDuplicateSubscription() {
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	s.postSubscribe(subscribeRequest, http.StatusOK)
	_ = s.receiveSubscriptionToken()

	s.postSubscribe(subscribeRequest, http.StatusConflict)
}

func (s *HTTPSuite) TestSubscribe_ReturnsNotFoundForMissingGitHubRepository() {
	s.github.SetRepoExists("missing/repo", false)

	s.postSubscribe(dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "missing/repo",
	}, http.StatusNotFound)
}

func (s *HTTPSuite) TestSubscribe_ReturnsTooManyRequestsForGitHubRateLimit() {
	s.github.SetRepoStatus("owner/repo", http.StatusTooManyRequests)

	s.postSubscribe(dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}, http.StatusTooManyRequests)
}

func (s *HTTPSuite) TestSubscribe_ReturnsBadRequestForInvalidInput() {
	s.postSubscribe(dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "invalid-repo",
	}, http.StatusBadRequest)
}

func (s *HTTPSuite) TestSubscribe_ReturnsBadRequestForMalformedJSON() {
	s.postSubscribeRaw([]byte("{"), http.StatusBadRequest)
}
