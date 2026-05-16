package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/repos/", handleRepository)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func handleRepository(w http.ResponseWriter, r *http.Request) {
	repo := strings.TrimPrefix(r.URL.Path, "/repos/")
	if repo == "" {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(repo, "/releases/latest") {
		writeJSON(w, map[string]string{"tag_name": "v1.0.0"})
		return
	}

	if strings.Count(repo, "/") != 1 {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, map[string]string{"full_name": repo})
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
