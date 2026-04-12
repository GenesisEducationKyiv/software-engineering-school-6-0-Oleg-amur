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

type Repository struct {
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

func (c *Client) CheckIfRepoExists(ctx context.Context, repoAddr string, log *slog.Logger) (bool, error) {

	url := fmt.Sprintf("%s/repos/%s", c.baseUrl, repoAddr)
	log.Info(url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return false, apperr.ErrRateLimitExceeded
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	return true, nil
}

func (c *Client) GetRepositoryLatestTag(ctx context.Context, repoAddr string, log *slog.Logger) (string, error) {

	latestTag := ""

	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.baseUrl, repoAddr)
	log.Info(url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return latestTag, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return latestTag, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return latestTag, apperr.ErrRateLimitExceeded
	}

	if resp.StatusCode == http.StatusNotFound {
		return latestTag, apperr.ErrRepoNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return latestTag, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var repo ReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return latestTag, err
	}

	return repo.TagName, nil
}
