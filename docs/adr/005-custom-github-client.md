# ADR 0005: Custom GitHub Client

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-05-08

## Context
The application needs to interact with the GitHub API to verify repositories and fetch releases.

## Proposed Options

### Option 1: Custom HTTP Client
A lightweight wrapper around `net/http` tailored specifically for our needs.
- **Pros:** No external dependencies, full control over request/response handling, educational value.
- **Cons:** Requires manual implementation of features like pagination or rate limit handling.

### Option 2: `google/go-github` library
The official Go client library for the GitHub API.
- **Pros:** Comprehensive, well-tested, handles most edge cases automatically.
- **Cons:** Adds a large dependency for just a few API calls.

## Decision
We chose **Option 1: Custom HTTP Client** to keep the project lightweight and to gain a deeper understanding of the GitHub API interactions.

## Consequences
- **Pros:**
    - Minimal external dependency.
    - More flexibility for development.
    - No unused functionality.
- **Cons:**
    - More maintenance if the API evolves.
    - Requires manual handling of GitHub-specific headers and status codes.
