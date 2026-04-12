This is Case Task for Software Engineering School 6.0


# Requerenments:

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

# Decision Log:

- Language: Go
    - Reason: I'm interested in this language. For this task this is enough:)
- Database: PostgreSQL
    - Reason: I have experience with it. Also it quite common to use posgre + go. Mongo does not fit there as it's overcomplicating due to absence of real relations and foreign keys there. SQLite is also good option here.
- Web framework: net/http 
    - Reason: For learning purposes it's better 
- Subscription: timer
    - Reason: Timer is good enough and it avoids external dependecies (if we compare with cron)
- Mail sending: I think we can use go-smpt and MailPit as kind of mock
    - Reason: Do we really need real emails? Mailpit will be more convinient for such task. But as we will use smtp, maybe we could also send those request to some external API, to send real messages
- Github API: use own client
    - Reason: there is go-github google lib, but own implementation makes more sense for education purposes
- DataSchema: use tree tables, subscribprions, subscribers and repositories:
    - Reason: to separate concerns, avoid duplication of data
- Config: started with yaml parser, ended up with clearenv, because it allows both yaml, ENV and default configuration, which is convinient

# Project Structure:

--.github/workflows
--docs/
--cmd/server/main.go
--internal/
        --api/ // there are http and grpc handlers, dtos and so on
        --scanner/
        --notifier/
        --repository/
        --models/
        --server/

--migrations/

(inspired by this)[https://github.com/golang-standards/project-layout/tree/master]

# Main business logic:

## Scaning logic:
- Select all repositories from db (they are already unique, as the same "name" can not be twice in db)
- loop through them and get latest release
- compare with last_seen_tag
- notify all subscribers to this repository if different

# Issues I've faced with

## GitHub API
Probably the biggest and the most interesting issue here is that some repos does not have realses, for example golang/go. And probably there we should get latest tag, not release. But API does not have such endpoint, unlike releases with releases/latest. So probably we should get all gets, and get latest. But there is the catch, that we can not sort them correctly, because go version are not correct according to semver. And if we manage to sort them somehow, this may not work for other cases. As task says only about releases, and not about tags, I think it was not the main goal of this task, so I did not implement getting latest tag. But i've found some way how this can be done: \r\n
- We can get them via GraphQL GitHub API:
```curl -s -H "Authorization: Bearer <token>" -X POST -d '{      "query": "query { repository(owner: \"golang\", name: \"go\") { refs(refPrefix: \"refs/tags/\", first: 1, orderBy: {field: TAG_COMMIT_DATE, direction: DESC}) {     nodes { name } } } }" }' https://api.github.com/graphql```

The issue here is that API token is mandatory here to retrieve data. Requirements mention case without token, so it does not fit there, but it can do the task if needed. 
- We can get latest tag by parsing github UI or via Atom Feed
```curl -s https://github.com/golang/go/tags.atom```

It can work great here, does not need token, but it's not API, so maybe also does not fit there

Maybe those things can be done later as improvement

## Database Schema
I've splited Subscribers, Repositories and Subscriptions. Initially I thought it would be great, because it separate those entities, allowing further easier extention and so on. But it resulted in some JOINs, that looks like overhead in this task, so maybe one or two tables (subscriptions + repos) would be more approptiate for this task.

## Golang
I must say, it was not that easy (after dotnet), especially ?enums?. I knew golang syntax, as I went through go.dev/learn and tried to implement some projects, but this task was and interesting challenge and a great way to get event closer to it. I went through a lot of repos trying to find how goland project usually looks like, what and how people usually do things, and I glad I did this task

Author: Volkoboi Oleh
