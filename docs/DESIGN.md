# 1. Introduction and Requerenments:

Introduction: This is high-level summary of the project and some kind of decision log.

Requirements:
- The service must match the API described in the swagger documentation.
- All functionality (API, Scanner, Notifier) must be implemented within a single service (monolith). Splitting into microservices is not allowed at this stage.
- All application data must be stored in a database. Database schema migrations must run on service startup.
- The repository must contain Dockerfile and docker-compose.yml that allow running the entire system in Docker.
- The service must regularly check for new releases for all active subscriptions. When a new release is detected, send an email to the subscriber. For each repository, store last_seen_tag and only notify if a new release appears.
- When creating a subscription, the service must verify the repository exists via GitHub API. Parameter format: owner/repo (e.g., golang/go). If the repository is not found - return 404. If the format is invalid - return 400.
- The service must correctly handle 429 Too Many Requests from GitHub API (rate limit: 60 req/hour without token, 5000 with token).
- You may use frameworks, but only “thin” solutions. High-level frameworks are prohibited: Nest.js (Node.js), Revel or Fx (Go), Laravel (PHP). Allowed: Fastify or Express (Node.js), Gin / Chi / net/http (Go), Slim or built-in language capabilities (PHP).
- Unit tests for business logic are mandatory. Integration tests are a bonus.
- You may add comments or logic descriptions in README.md. Correct logic can be an advantage in evaluation if you don’t fully complete the task.
- Expected languages: Golang, Node.js, or PHP.

# 2. Decision Log:

- Language: Go
    - Reason: I'm interested in this language. For this task this is enough:)
- Database: PostgreSQL
    - Reason: I have experience with it. Also it quite common to use posgre + go. Mongo does not fit there as it's overcomplicating due to absence of real relations and foreign keys there. SQLite is also good option here.
- Web framework: net/http 
    - Reason: For learning purposes it's better 
- Subscription: timer
    - Reason: Timer is good enough and it avoids external dependecies (if we compare with cron)
- Mail sending: I think we can use go-smpt and MailPit as kind of mock
    - Reason: Do we really need real emails? Mailpit will be more convinient for such task
- Github API: use own client
    - Reason: there is go-github google lib, but own implementation makes more sense for education purposes
- DataSchema: use two tables, subscribprions and repositories:
    - Reason: to separate concerns, avoid duplication of data

# 3. Project Structure:

--.github/workflows
--docs/
--cmd/server/main.go
--internal/
        --api/
        --scanner/
        --notifier/
        --repository/
        --models/
--migrations/
--tests/

# 4. Main business logic:

## Scaning logic:
- Select all unique repositories from db
- loop through them and get latest release
- compare with last_seen_tag
- notify all subscribers to this repository if different

