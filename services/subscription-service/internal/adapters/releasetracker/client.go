package releasetracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/usecase"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type repositoryRequest struct {
	Repository string `json:"repository"`
}

type repositoryResponse struct {
	Name        string `json:"name"`
	LastSeenTag string `json:"last_seen_tag"`
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	return &Client{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *Client) EnsureTracked(
	ctx context.Context,
	repoName string,
) (*subscriptionusecase.RepositoryView, error) {
	body, err := json.Marshal(repositoryRequest{Repository: repoName})
	if err != nil {
		return nil, fmt.Errorf("encode repository request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/internal/v1/repositories/ensure",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create release tracker request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.doRepositoryRequest(req)
}

func (c *Client) GetRepository(
	ctx context.Context,
	repoName string,
) (*subscriptionusecase.RepositoryView, error) {
	endpoint := c.baseURL + "/internal/v1/repositories?repository=" + url.QueryEscape(repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create release tracker request: %w", err)
	}

	return c.doRepositoryRequest(req)
}

func (c *Client) doRepositoryRequest(req *http.Request) (*subscriptionusecase.RepositoryView, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call release tracker: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return nil, apperr.ErrInvalidRepositoryFormat
		case http.StatusNotFound:
			return nil, apperr.ErrRepoNotFound
		case http.StatusTooManyRequests:
			return nil, apperr.ErrRateLimitExceeded
		default:
			return nil, fmt.Errorf("release tracker returned status %d", resp.StatusCode)
		}
	}

	var response repositoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode release tracker response: %w", err)
	}
	return &subscriptionusecase.RepositoryView{
		Name:        response.Name,
		LastSeenTag: response.LastSeenTag,
	}, nil
}
