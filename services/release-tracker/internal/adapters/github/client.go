package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/apperr"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func NewClient(httpClient *http.Client, baseURL, token string) *Client {
	return &Client{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/"), token: token}
}

func (c *Client) RepositoryExists(ctx context.Context, name string) (bool, error) {
	resp, err := c.get(ctx, c.baseURL+"/repos/"+name)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GitHub repository request returned %d", resp.StatusCode)
	}
	return true, nil
}

func (c *Client) LatestTag(ctx context.Context, name string) (string, error) {
	resp, err := c.get(ctx, c.baseURL+"/repos/"+name+"/releases/latest")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return "", apperr.ErrRepositoryNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub release request returned %d", resp.StatusCode)
	}
	var result struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode GitHub release: %w", err)
	}
	return result.Tag, nil
}

func (c *Client) get(ctx context.Context, endpoint string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		_ = resp.Body.Close()
		return nil, apperr.ErrRateLimitExceeded
	}
	return resp, nil
}
