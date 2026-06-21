package subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/domain"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	return &Client{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *Client) ListActiveByRepository(
	ctx context.Context,
	repository string,
) ([]domain.ActiveSubscription, error) {
	endpoint := c.baseURL + "/internal/v1/subscriptions?repository=" + url.QueryEscape(repository)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create subscriptions request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call subscriptions service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscriptions service returned status %d", resp.StatusCode)
	}

	var response struct {
		Subscriptions []struct {
			Email            string `json:"email"`
			UnsubscribeToken string `json:"unsubscribe_token"`
		} `json:"subscriptions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode subscriptions response: %w", err)
	}

	result := make([]domain.ActiveSubscription, 0, len(response.Subscriptions))
	for _, subscription := range response.Subscriptions {
		result = append(result, domain.ActiveSubscription{
			Email:            subscription.Email,
			UnsubscribeToken: subscription.UnsubscribeToken,
		})
	}
	return result, nil
}
