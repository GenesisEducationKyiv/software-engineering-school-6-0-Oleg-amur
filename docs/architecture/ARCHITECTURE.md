# Architecture

This document describes the current architecture of GitHub Release Notifier using the C4 model. The diagrams reflect the Docker Compose deployment and the Go modules under `services/` and `shared/`.

## Level 1: System Context

The system lets users subscribe to GitHub repositories, confirms subscriptions by email, periodically checks GitHub releases, and sends release notifications.

```mermaid
C4Context
    title System Context Diagram for GitHub Release Notifier

    Person(user, "Subscriber", "Registers an email address and subscribes to public GitHub repositories")
    System(system, "GitHub Release Notifier", "Tracks GitHub releases and emails confirmed subscribers")
    System_Ext(github, "GitHub API", "Provides repository metadata and latest release tags")
    System_Ext(smtp, "SMTP Server / Mailpit", "Delivers confirmation and release notification emails")
    System_Ext(observability, "Observability Tools", "Prometheus, Grafana, Filebeat, Elasticsearch, and Kibana")

    Rel(user, system, "Subscribes, confirms, lists, and unsubscribes", "HTTPS/HTTP")
    Rel(system, github, "Checks repository existence and latest releases", "HTTPS/REST")
    Rel(system, smtp, "Sends confirmation and release emails", "SMTP")
    Rel(observability, system, "Scrapes metrics and collects logs", "HTTP / Docker logs")

    UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="1")
```

## Level 2: Containers

The application is split into independently deployable services. Each stateful service owns its database. Cross-service business events use RabbitMQ, while synchronous repository and subscription queries use HTTP or gRPC.

```mermaid
C4Container
    title Container Diagram for GitHub Release Notifier

    Person(user, "Subscriber", "Uses the web UI and public subscription API")

    System_Boundary(app, "GitHub Release Notifier") {
        Container(subscription, "Subscription Service", "Go, Chi, gRPC", "Serves the static UI and subscription APIs, owns subscribers, subscriptions, confirmation sagas, and the outbox")
        Container(releaseTracker, "Release Tracker", "Go, Chi", "Stores tracked repositories, polls GitHub, and publishes release notification jobs")
        Container(notificationWorker, "Notification Worker", "Go", "Consumes notification jobs, sends emails, and publishes subscription saga results")
        ContainerDb(subscriptionDb, "Subscription DB", "PostgreSQL", "Stores subscribers, subscriptions, confirmation sagas, and outbox records")
        ContainerDb(releaseDb, "Release Tracker DB", "PostgreSQL", "Stores tracked repositories and last seen release tags")
        ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Durable notification queue, saga result queue, and dead-letter queues")
    }

    System_Ext(github, "GitHub API", "Repository and release metadata")
    System_Ext(smtp, "SMTP Server / Mailpit", "Email delivery")
    System_Ext(prometheus, "Prometheus", "Metrics scraping")

    Rel(user, subscription, "Uses web UI and subscription endpoints", "HTTP")
    Rel(subscription, releaseTracker, "Ensures repositories and reads metadata", "HTTP/REST")
    Rel(subscription, subscriptionDb, "Reads and writes subscription state", "SQL")
    Rel(subscription, rabbitmq, "Publishes confirmation requests and consumes saga results", "AMQP")
    Rel(releaseTracker, releaseDb, "Reads and writes tracked repositories", "SQL")
    Rel(releaseTracker, github, "Checks existence and latest release tags", "HTTPS/REST")
    Rel(releaseTracker, subscription, "Lists active subscriptions for changed repositories", "gRPC; HTTP adapter retained for comparison")
    Rel(releaseTracker, rabbitmq, "Publishes release notification requests", "AMQP")
    Rel(notificationWorker, rabbitmq, "Consumes notification requests and publishes saga results", "AMQP")
    Rel(notificationWorker, smtp, "Sends emails", "SMTP")
    Rel(prometheus, subscription, "Scrapes metrics", "HTTP /metrics")
    Rel(prometheus, releaseTracker, "Scrapes metrics", "HTTP /metrics")

    UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="1")
```

## Level 3: Subscription Service Components

The subscription service uses vertical module slices. Transport adapters call use cases, use cases depend on ports, and infrastructure adapters are wired in the composition root.

```mermaid
C4Component
    title Component Diagram for Subscription Service

    Container(releaseTracker, "Release Tracker", "Go service", "Repository metadata service")
    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification and saga event broker")
    ContainerDb(subscriptionDb, "Subscription DB", "PostgreSQL", "Subscription-owned data")

    Container_Boundary(subscription, "Subscription Service") {
        Component(httpRouter, "HTTP Router and Handlers", "Chi / Go", "Serves static UI, public REST endpoints, internal REST endpoint, health, and metrics")
        Component(grpcHandler, "gRPC Handler", "Go gRPC", "Exposes active-subscription queries for release tracking")
        Component(usecases, "Subscription Use Cases", "Go", "Coordinates subscribe, confirm, unsubscribe, list, and active-subscription queries")
        Component(workflows, "Confirmation Workflow and Outbox", "Go", "Starts confirmation saga, stores outbox messages, and handles saga results")
        Component(stores, "PostgreSQL Stores", "Go / SQL", "Persists subscribers, subscriptions, sagas, and outbox messages")
        Component(releaseClient, "Release Tracker Client", "Go HTTP client", "Ensures repositories and reads repository metadata")
        Component(eventbus, "RabbitMQ Adapters", "AMQP", "Publishes confirmation requests and consumes saga result events")
        Component(domain, "Subscription Domain", "Go", "Subscription, subscriber, saga, and outbox domain models")
    }

    Rel(httpRouter, usecases, "Invokes")
    Rel(grpcHandler, usecases, "Invokes")
    Rel(usecases, domain, "Uses")
    Rel(usecases, workflows, "Starts confirmation through port")
    Rel(workflows, stores, "Persists saga and outbox state through ports")
    Rel(usecases, stores, "Reads and writes subscriptions through ports")
    Rel(usecases, releaseClient, "Ensures and reads repositories through port")
    Rel(releaseClient, releaseTracker, "Calls", "HTTP/REST")
    Rel(stores, subscriptionDb, "Reads/Writes", "SQL")
    Rel(eventbus, rabbitmq, "Publishes and consumes", "AMQP")
    Rel(workflows, eventbus, "Publishes outbox messages through port")

    UpdateLayoutConfig($c4ShapeInRow="4", $c4BoundaryInRow="1")
```

## Level 3: Release Tracker Components

The release tracker owns repository tracking and scanning. The subscription query client is selected in the composition root; gRPC is the current default and HTTP remains available for comparison.

```mermaid
C4Component
    title Component Diagram for Release Tracker

    Container(subscription, "Subscription Service", "Go service", "Active subscriber source")
    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification broker")
    ContainerDb(releaseDb, "Release Tracker DB", "PostgreSQL", "Tracked repositories")
    System_Ext(github, "GitHub API", "Repository release metadata")

    Container_Boundary(releaseTracker, "Release Tracker") {
        Component(httpRouter, "HTTP Router and Handlers", "Chi / Go", "Repository ensure/read API, health, and metrics")
        Component(scanner, "Scan Scheduler", "Go worker", "Runs release scans on a configured interval")
        Component(usecases, "Release Tracker Use Cases", "Go", "Ensures repositories, reads repository metadata, and scans for release changes")
        Component(stores, "Repository Store", "Go / SQL", "Persists repository names and last seen release tags")
        Component(githubClient, "GitHub Client", "Go HTTP client", "Checks repository existence and latest release tags")
        Component(subscriptionClient, "Subscription Query Client", "Go gRPC / HTTP client", "Lists active subscriptions for repositories with new releases")
        Component(publisher, "Notification Publisher", "AMQP", "Publishes release notification requests")
        Component(domain, "Repository Domain", "Go", "Repository and active-subscription models")
    }

    Rel(httpRouter, usecases, "Invokes")
    Rel(scanner, usecases, "Triggers scheduled scan")
    Rel(usecases, domain, "Uses")
    Rel(usecases, stores, "Reads and writes repositories through port")
    Rel(stores, releaseDb, "Reads/Writes", "SQL")
    Rel(usecases, githubClient, "Checks repositories and tags through port")
    Rel(githubClient, github, "Calls", "HTTPS/REST")
    Rel(usecases, subscriptionClient, "Requests active subscriptions through port")
    Rel(subscriptionClient, subscription, "Calls", "gRPC default; HTTP fallback")
    Rel(usecases, publisher, "Publishes notification jobs through port")
    Rel(publisher, rabbitmq, "Publishes", "AMQP")

    UpdateLayoutConfig($c4ShapeInRow="4", $c4BoundaryInRow="1")
```

## Level 3: Notification Worker Components

The notification worker has no database. It consumes durable events, sends emails, and reports confirmation success or failure back to the subscription saga.

```mermaid
C4Component
    title Component Diagram for Notification Worker

    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification and saga result broker")
    System_Ext(smtp, "SMTP Server / Mailpit", "Email delivery")

    Container_Boundary(notificationWorker, "Notification Worker") {
        Component(consumer, "Notification Consumer", "AMQP", "Decodes confirmation and release notification events")
        Component(service, "Notification Service", "Go", "Coordinates email delivery and confirmation saga result publishing")
        Component(builder, "Message Builder", "Go", "Builds confirmation and release email subjects and bodies")
        Component(emailClient, "Email Client", "SMTP", "Sends email with retry policy")
        Component(resultPublisher, "Saga Result Publisher", "AMQP", "Publishes confirmation succeeded or failed events")
    }

    Rel(consumer, rabbitmq, "Consumes notification events from", "AMQP")
    Rel(consumer, service, "Dispatches events to")
    Rel(service, builder, "Builds messages with")
    Rel(service, emailClient, "Sends email through")
    Rel(emailClient, smtp, "Sends", "SMTP")
    Rel(service, resultPublisher, "Reports confirmation outcome through")
    Rel(resultPublisher, rabbitmq, "Publishes saga result events to", "AMQP")

    UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="1")
```

## Layering and Dependency Rules

The service code follows these dependency directions:

- `cmd/server`: composition root; may depend on all concrete packages to wire the process.
- `transport`: HTTP and gRPC adapters; may depend on use cases and domain models, but not persistence or event-bus adapters.
- `usecase`: application operations; may depend on domain models and local ports/interfaces, but not concrete infrastructure adapters.
- `workflow`: application workflows such as confirmation saga and outbox relay; may depend on domain models and shared event contracts, but not concrete infrastructure adapters.
- `domain`: business state and invariants; must not depend on service infrastructure packages.
- `persistence` and `adapters`: concrete PostgreSQL, RabbitMQ, GitHub, SMTP, and cross-service clients; implement ports and may depend inward on domain/usecase contracts.
- `shared/contracts` and `shared/messaging`: shared event, protobuf, and RabbitMQ utility modules used across service boundaries.

The `.go-arch-lint.yml` rules enforce the most important import boundaries as part of `make arch-lint` and `make lint`.
