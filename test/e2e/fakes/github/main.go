package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultTag = "v1.0.0"

type serverState struct {
	mu   sync.Mutex
	tags map[string]string
}

func main() {
	state := &serverState{
		tags: map[string]string{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/test/latest-tag", state.handleSetLatestTag)
	mux.HandleFunc("/repos/", state.handleRepository)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func (s *serverState) handleRepository(w http.ResponseWriter, r *http.Request) {
	repo := strings.TrimPrefix(r.URL.Path, "/repos/")
	if repo == "" {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(repo, "/releases/latest") {
		repo = strings.TrimSuffix(repo, "/releases/latest")
		writeJSON(w, map[string]string{"tag_name": s.latestTag(repo)})
		return
	}

	if strings.Count(repo, "/") != 1 {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, map[string]string{"full_name": repo})
}

func (s *serverState) latestTag(repo string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tag := s.tags[repo]; tag != "" {
		return tag
	}
	return defaultTag
}

func (s *serverState) handleSetLatestTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Repo string `json:"repo"`
		Tag  string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Repo == "" || req.Tag == "" {
		http.Error(w, "repo and tag are required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.tags[req.Repo] = req.Tag
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
