//go:build integration

package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/test/integration/testkit"
)

func TestEnsureAndGetRepository(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg := testkit.NewPostgres(ctx, t)
	pg.Reset(t)
	app := testkit.NewApp(t, pg, &testkit.GitHub{Exists: true, Tag: "v1.0.0"})

	ensureRequest := httptest.NewRequest(
		http.MethodPost,
		"/internal/v1/repositories/ensure",
		bytes.NewBufferString(`{"repository":"owner/repo"}`),
	)
	ensureResponse := httptest.NewRecorder()
	app.HTTPHandler.ServeHTTP(ensureResponse, ensureRequest)
	if ensureResponse.Code != http.StatusOK {
		t.Fatalf("ensure status = %d, body = %s", ensureResponse.Code, ensureResponse.Body.String())
	}
	var ensured struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(ensureResponse.Body).Decode(&ensured); err != nil {
		t.Fatalf("decode ensure response: %v", err)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/internal/v1/repositories?id=%d", ensured.ID),
		nil,
	)
	getResponse := httptest.NewRecorder()
	app.HTTPHandler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getResponse.Code, getResponse.Body.String())
	}

	var response struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		LastSeenTag string `json:"last_seen_tag"`
	}
	if err := json.NewDecoder(getResponse.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != ensured.ID || response.Name != "owner/repo" || response.LastSeenTag != "v1.0.0" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
