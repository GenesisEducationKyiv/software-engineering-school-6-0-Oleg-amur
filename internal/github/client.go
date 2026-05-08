package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
)

const (
	headerAccept           = "Accept"
	headerGitHubApiVersion = "X-GitHub-Api-Version"
	headerAuthorization    = "Authorization"

	acceptValue     = "application/vnd.github+json"
	apiVersionValue = "2026-03-10"
)

type Client struct {
	httpClient *http.Client
	baseUrl    string
	apiToken   string
	log        *slog.Logger
}

type ReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func NewClient(url string, token string, timeout time.Duration, log *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseUrl:    url,
		apiToken:   token,
		log:        log,
	}
}

func (c *Client) do(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(headerAccept, acceptValue)
	req.Header.Set(headerGitHubApiVersion, apiVersionValue)
	if c.apiToken != "" {
		req.Header.Set(headerAuthorization, "Bearer "+c.apiToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if rErr := resp.Body.Close(); rErr != nil {
			c.log.Error("failed to close response body", "error", rErr)
		}

		return nil, apperr.ErrRateLimitExceeded
	}

	return resp, nil
}

func (c *Client) CheckIfRepoExists(
	ctx context.Context,
	repoAddr string,
) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s", c.baseUrl, repoAddr)
	c.log.Info("checking repository existence", "url", url)

	resp, err := c.do(ctx, http.MethodGet, url)
	if err != nil {
		return false, err
	}
	defer func() {
		if rErr := resp.Body.Close(); rErr != nil {
			c.log.Error("failed to close response body", "error", rErr)
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

func (c *Client) GetRepositoryLatestTag(
	ctx context.Context,
	repoAddr string,
) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.baseUrl, repoAddr)
	c.log.Info("fetching latest release", "url", url)

	resp, err := c.do(ctx, http.MethodGet, url)
	if err != nil {
		return "", err
	}
	defer func() {
		if rErr := resp.Body.Close(); rErr != nil {
			c.log.Error("failed to close response body", "error", rErr)
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
