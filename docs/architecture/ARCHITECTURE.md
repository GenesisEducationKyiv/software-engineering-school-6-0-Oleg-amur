# Architecture

This document describes the current architecture of GitHub Release Notifier using the C4 model. The diagrams reflect the Docker Compose deployment and the Go modules under `services/` and `shared/`.

## Level 1: System Context

The system lets users subscribe to GitHub repositories, confirms subscriptions by email, periodically checks GitHub releases, and sends release notifications.

```mermaid
C4Context
    title System Context Diagram for GitHub Release Notifier

    Person(user, "Subscriber", "Subscribes to public GitHub repositories")
    System(system, "GitHub Release Notifier", "Tracks releases and emails subscribers")
    System_Ext(github, "GitHub API", "Repository and release data")
    System_Ext(smtp, "SMTP Server / Mailpit", "Email delivery")
    System_Ext(observability, "Observability Tools", "Metrics and logs")

    Rel_R(user, system, "Uses", "HTTP")
    Rel_D(system, github, "Reads releases", "HTTPS/REST")
    Rel_D(system, smtp, "Sends emails", "SMTP")
    Rel_L(observability, system, "Reads telemetry", "HTTP / logs")

    UpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")
```

## Level 2: Containers

The application is split into independently deployable services. Each stateful service owns its database. Cross-service business events use RabbitMQ, while synchronous repository and subscription queries use HTTP or gRPC.

```mermaid
C4Container
    title Container Diagram for GitHub Release Notifier

    Person(user, "Subscriber", "Uses the web UI and public subscription API")

    System_Boundary(app, "GitHub Release Notifier") {
        Container(subscription, "Subscription Service", "Go, Chi, gRPC", "UI, subscription API, confirmation sagas, and outbox")
        Container(releaseTracker, "Release Tracker", "Go, Chi", "Repository tracking and GitHub polling")
        Container(notificationWorker, "Notification Worker", "Go", "Email delivery from durable events")
        ContainerDb(subscriptionDb, "Subscription DB", "PostgreSQL", "Subscribers, subscriptions, sagas, outbox")
        ContainerDb(releaseDb, "Release Tracker DB", "PostgreSQL", "Tracked repositories and last seen tags")
        ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification, saga, and DLQ queues")
    }

    System_Ext(github, "GitHub API", "Repository releases")
    System_Ext(smtp, "SMTP Server / Mailpit", "Email delivery")

    Rel_R(user, subscription, "Uses", "HTTP")
    Rel_R(subscription, releaseTracker, "Ensures repos", "HTTP")
    Rel_L(releaseTracker, subscription, "Reads subscribers", "gRPC")
    Rel_D(subscription, subscriptionDb, "Reads/Writes", "SQL")
    Rel_D(releaseTracker, releaseDb, "Reads/Writes", "SQL")
    Rel_R(releaseTracker, github, "Polls releases", "HTTPS")
    Rel_D(subscription, rabbitmq, "Confirmation events", "AMQP")
    Rel_D(releaseTracker, rabbitmq, "Release events", "AMQP")
    Rel_U(notificationWorker, rabbitmq, "Consumes events", "AMQP")
    Rel_R(notificationWorker, smtp, "Sends emails", "SMTP")

    UpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")
```

Observability containers are available through the Compose `observability` profile and scrape `/metrics` from the Go services.

## Level 3: Subscription Service Components

The subscription service uses vertical module slices. Transport adapters call use cases, use cases depend on ports, and infrastructure adapters are wired in the composition root.

```mermaid
C4Component
    title Component Diagram for Subscription Service

    Container(releaseTracker, "Release Tracker", "Go service", "Repository metadata")
    ContainerDb(subscriptionDb, "Subscription DB", "PostgreSQL", "Subscription data")
    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Events")

    Container_Boundary(subscription, "Subscription Service") {
        Component(httpRouter, "HTTP Handlers", "Chi / Go", "Public API, static UI, health, metrics")
        Component(grpcHandler, "gRPC Handler", "Go gRPC", "Active subscription query")
        Component(usecases, "Use Cases", "Go", "Subscribe, confirm, unsubscribe, list")
        Component(domain, "Domain", "Go", "Subscriber, subscription, saga models")
        Component(workflows, "Workflow + Outbox", "Go", "Confirmation saga and outbox relay")
        Component(stores, "PostgreSQL Stores", "Go / SQL", "Subscribers, subscriptions, sagas, outbox")
        Component(releaseClient, "Release Client", "Go HTTP client", "Repository metadata")
        Component(eventbus, "RabbitMQ Adapters", "AMQP", "Confirmation and saga events")
    }

    Rel_D(httpRouter, usecases, "Calls")
    Rel_D(grpcHandler, usecases, "Calls")
    Rel_R(usecases, domain, "Uses")
    Rel_D(usecases, workflows, "Starts saga")
    Rel_D(workflows, stores, "Persists")
    Rel_R(usecases, stores, "Reads/Writes")
    Rel_D(usecases, releaseClient, "Tracks repo")
    Rel_D(releaseClient, releaseTracker, "Calls", "HTTP")
    Rel_D(stores, subscriptionDb, "Reads/Writes", "SQL")
    Rel_R(workflows, eventbus, "Publishes")
    Rel_D(eventbus, rabbitmq, "AMQP")

    UpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")
```

## Level 3: Release Tracker Components

The release tracker owns repository tracking and scanning. The subscription query client is selected in the composition root; gRPC is the current default and HTTP remains available for comparison.

```mermaid
C4Component
    title Component Diagram for Release Tracker

    Container(subscription, "Subscription Service", "Go service", "Active subscribers")
    ContainerDb(releaseDb, "Release Tracker DB", "PostgreSQL", "Tracked repositories")
    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification events")
    System_Ext(github, "GitHub API", "Release metadata")

    Container_Boundary(releaseTracker, "Release Tracker") {
        Component(httpRouter, "HTTP Handlers", "Chi / Go", "Repository API, health, metrics")
        Component(scanner, "Scan Scheduler", "Go worker", "Periodic release scans")
        Component(usecases, "Use Cases", "Go", "Ensure repo, get repo, scan releases")
        Component(domain, "Domain", "Go", "Repository and subscriber models")
        Component(stores, "Repository Store", "Go / SQL", "Tracked repositories")
        Component(githubClient, "GitHub Client", "Go HTTP client", "Repository releases")
        Component(subscriptionClient, "Subscription Client", "Go gRPC / HTTP", "Active subscriptions")
        Component(publisher, "Publisher", "AMQP", "Release notification jobs")
    }

    Rel_D(httpRouter, usecases, "Calls")
    Rel_D(scanner, usecases, "Triggers")
    Rel_R(usecases, domain, "Uses")
    Rel_D(usecases, stores, "Reads/Writes")
    Rel_D(stores, releaseDb, "Reads/Writes", "SQL")
    Rel_R(usecases, githubClient, "Checks tags")
    Rel_D(githubClient, github, "Calls", "HTTPS")
    Rel_D(usecases, subscriptionClient, "Gets subscribers")
    Rel_D(subscriptionClient, subscription, "Calls", "gRPC")
    Rel_R(usecases, publisher, "Publishes")
    Rel_D(publisher, rabbitmq, "AMQP")

    UpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")
```

## Level 3: Notification Worker Components

The notification worker has no database. It consumes durable events, sends emails, and reports confirmation success or failure back to the subscription saga.

```mermaid
C4Component
    title Component Diagram for Notification Worker

    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification and saga events")
    System_Ext(smtp, "SMTP Server / Mailpit", "Email delivery")

    Container_Boundary(notificationWorker, "Notification Worker") {
        Component(consumer, "Consumer", "AMQP", "Decodes notification events")
        Component(service, "Notification Service", "Go", "Coordinates delivery")
        Component(builder, "Message Builder", "Go", "Builds email content")
        Component(emailClient, "Email Client", "SMTP", "Sends with retry policy")
        Component(resultPublisher, "Saga Result Publisher", "AMQP", "Reports confirmation outcome")
    }

    Rel_D(rabbitmq, consumer, "Delivers", "AMQP")
    Rel_D(consumer, service, "Dispatches")
    Rel_R(service, builder, "Builds")
    Rel_D(service, emailClient, "Sends")
    Rel_D(emailClient, smtp, "SMTP")
    Rel_R(service, resultPublisher, "Reports")
    Rel_D(resultPublisher, rabbitmq, "Publishes", "AMQP")

    UpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")
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

The layering rules above document the intended dependency direction between packages.
