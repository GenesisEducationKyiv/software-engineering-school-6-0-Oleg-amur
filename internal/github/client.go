package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/apperr"
)

type Client struct {
	httpClient *http.Client
	baseUrl    string
	apiToken   string
}

type ReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func NewClient(url string, token string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseUrl:    url,
		apiToken:   token,
	}
}

func (c *Client) do(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
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

func (c *Client) CheckIfRepoExists(ctx context.Context, repoAddr string, log *slog.Logger) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s", c.baseUrl, repoAddr)
	log.Info("checking repository existence", "url", url)

	resp, err := c.do(ctx, http.MethodGet, url)
	if err != nil {
		return false, err
	}
	defer func() {
		if rErr := resp.Body.Close(); rErr != nil {
			log.Error("failed to close response body", "err", rErr)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	return true, nil
}

func (c *Client) GetRepositoryLatestTag(ctx context.Context, repoAddr string, log *slog.Logger) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.baseUrl, repoAddr)
	log.Info("fetching latest release", "url", url)

	resp, err := c.do(ctx, http.MethodGet, url)
	if err != nil {
		return "", err
	}
	defer func() {
		if rErr := resp.Body.Close(); rErr != nil {
			log.Error("failed to close response body", "err", rErr)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return "", apperr.ErrRepoNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var release ReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode release response: %w", err)
	}

	return release.TagName, nil
}
