# Architecture

This document describes the current architecture of GitHub Release Notifier using the C4 model. The diagrams reflect the Docker Compose deployment and the Go modules under `services/` and `shared/`.

## Level 1: System Context

The system lets users subscribe to GitHub repositories, confirms subscriptions by email, periodically checks GitHub releases, and sends release notifications.

```mermaid
C4Context
    title System Context Diagram for GitHub Release Notifier

    Person(user, "Subscriber", "Subscribes to public GitHub repositories")
    System_Ext(github, "GitHub API", "Repository and release data")
    System_Ext(smtp, "SMTP Server / Mailpit", "Email delivery")
    System_Ext(observability, "Observability Tools", "Metrics and logs")
    System(system, "GitHub Release Notifier", "Tracks releases and emails subscribers")

    Rel_R(user, system, "")
    Rel_U(system, github, "")
    Rel_U(system, smtp, "")
    Rel_D(observability, system, "")

    UpdateLayoutConfig($c4ShapeInRow="4", $c4BoundaryInRow="1")
```

| From | To | Interaction |
| --- | --- | --- |
| Subscriber | GitHub Release Notifier | Uses web UI and subscription endpoints over HTTP |
| GitHub Release Notifier | GitHub API | Reads repository and release data over HTTPS/REST |
| GitHub Release Notifier | SMTP Server / Mailpit | Sends confirmation and release emails over SMTP |
| Observability Tools | GitHub Release Notifier | Reads metrics and logs |

## Level 2: Containers

The application is split into independently deployable services. Each stateful service owns its database. Cross-service business events use RabbitMQ, while synchronous repository and subscription queries use HTTP or gRPC.

```mermaid
C4Container
    title Container Diagram for GitHub Release Notifier

    Person(user, "Subscriber", "Uses the web UI and public subscription API")
    System_Ext(smtp, "SMTP Server / Mailpit", "Email delivery")

    System_Boundary(app, "GitHub Release Notifier") {
        Container(subscription, "Subscription Service", "Go, Chi, gRPC", "UI, subscription API, confirmation sagas, and outbox")
        Container(syncLane, " ", " ", " ")
        Container(releaseTracker, "Release Tracker", "Go, Chi", "Repository tracking and GitHub polling")

        Container(spacerLeft, " ", " ", " ")
        Container(notificationWorker, "Notification Worker", "Go", "Email delivery from durable events")
        Container(spacerRight, " ", " ", " ")

        ContainerDb(subscriptionDb, "Subscription DB", "PostgreSQL", "Subscribers, subscriptions, sagas, outbox")
        ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification, saga, and DLQ queues")
        ContainerDb(releaseDb, "Release Tracker DB", "PostgreSQL", "Tracked repositories and last seen tags")
    }

    System_Ext(github, "GitHub API", "Repository releases")

    Rel_D(user, subscription, "")
    Rel_R(subscription, releaseTracker, "")
    Rel_L(releaseTracker, subscription, "")
    Rel_D(subscription, subscriptionDb, "")
    Rel_D(releaseTracker, releaseDb, "")
    Rel_D(releaseTracker, github, "")
    Rel_R(subscription, rabbitmq, "")
    Rel_L(releaseTracker, rabbitmq, "")
    Rel_U(rabbitmq, notificationWorker, "")
    Rel_U(notificationWorker, smtp, "")

    UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="1")
    UpdateElementStyle(syncLane, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(spacerLeft, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(spacerRight, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
```

| From | To | Interaction |
| --- | --- | --- |
| Subscriber | Subscription Service | Uses static UI and public subscription endpoints over HTTP |
| Subscription Service | Release Tracker | Ensures repositories and reads metadata over HTTP |
| Release Tracker | Subscription Service | Reads active subscribers over gRPC; HTTP adapter remains for comparison |
| Subscription Service | Subscription DB | Reads and writes subscribers, subscriptions, sagas, and outbox records over SQL |
| Release Tracker | Release Tracker DB | Reads and writes tracked repositories over SQL |
| Release Tracker | GitHub API | Polls latest release tags over HTTPS |
| Subscription Service | RabbitMQ | Publishes confirmation events and consumes saga results over AMQP |
| Release Tracker | RabbitMQ | Publishes release notification events over AMQP |
| Notification Worker | RabbitMQ | Consumes notification events and publishes saga results over AMQP |
| Notification Worker | SMTP Server / Mailpit | Sends emails over SMTP |

Observability containers are available through the Compose `observability` profile and scrape `/metrics` from the Go services.

## Code Architecture Style

At the system level, the application is split into independently deployable services. Each service owns its runtime process and, where needed, its own PostgreSQL database.

`subscription-service` and `release-tracker` use a modular ports-and-adapters style. Their business capabilities live under `internal/modules/...`; inside a module, transport adapters, use cases, workflows, domain models, and persistence stay close to the capability they serve. Use cases define application behavior and depend on local ports, while concrete PostgreSQL, RabbitMQ, GitHub, HTTP, and gRPC adapters are wired in `cmd/server`.

`notification-worker` is intentionally simpler because it has one main capability: notification delivery. It uses a thin application service in `internal/notification`, with SMTP and RabbitMQ adapters around it, instead of the full module structure used by the stateful services.

The dependency direction stays inward: transports and infrastructure adapters call use cases or application services, and domain models do not depend on infrastructure. The architecture lint configuration captures the most important import boundaries.

## Level 3: Subscription Service Components

The subscription service uses vertical module slices. Transport adapters call use cases, use cases depend on ports, and infrastructure adapters are wired in the composition root.

```mermaid
C4Component
    title Component Diagram for Subscription Service

    Container(releaseTracker, "Release Tracker", "Go service", "Repository metadata")
    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Events")

    Container_Boundary(subscription, "Subscription Service") {
        Component(httpTransport, "HTTP Transport", "Chi / Go", "Public API, static UI, health, metrics")
        Component(grpcTransport, "gRPC Transport", "Go gRPC", "Active subscription query")
        Component(eventbus, "RabbitMQ Adapter", "AMQP", "Confirmation and saga events")
        Component(releaseClient, "Release Tracker Client", "Go HTTP client", "Repository metadata")
        Component(subscriptionTopSpacer, " ", " ", " ")

        Component(subscriptionUsecaseSpacerLeft, " ", " ", " ")
        Component(subscriptionUsecaseSpacerMidLeft, " ", " ", " ")
        Component(usecases, "Use Cases", "Go", "Subscribe, confirm, unsubscribe, list")
        Component(subscriptionUsecaseSpacerMidRight, " ", " ", " ")
        Component(subscriptionUsecaseSpacerRight, " ", " ", " ")

        Component(workflows, "Workflow + Outbox", "Go", "Confirmation saga and outbox relay")
        Component(domain, "Domain Models", "Go", "Subscriber, subscription, saga models")
        Component(repository, "Repository", "Go / SQL", "Subscribers, subscriptions, sagas, outbox")
    }

    ContainerDb(subscriptionDb, "Subscription DB", "PostgreSQL", "Subscription data")

    Rel_D(httpTransport, usecases, "")
    Rel_D(grpcTransport, usecases, "")
    Rel_D(eventbus, workflows, "")
    Rel_D(releaseClient, releaseTracker, "")
    Rel_R(usecases, domain, "")
    Rel_D(usecases, workflows, "")
    Rel_D(workflows, repository, "")
    Rel_D(usecases, repository, "")
    Rel_D(usecases, releaseClient, "")
    Rel_R(workflows, eventbus, "")
    Rel_D(repository, subscriptionDb, "")
    Rel_D(eventbus, rabbitmq, "")

    UpdateLayoutConfig($c4ShapeInRow="5", $c4BoundaryInRow="1")
    UpdateElementStyle(subscriptionTopSpacer, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(subscriptionUsecaseSpacerLeft, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(subscriptionUsecaseSpacerMidLeft, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(subscriptionUsecaseSpacerMidRight, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(subscriptionUsecaseSpacerRight, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
```

| From | To | Interaction |
| --- | --- | --- |
| HTTP Transport | Use Cases | Calls subscription use cases |
| gRPC Transport | Use Cases | Calls active-subscription query use case |
| RabbitMQ Adapter | Workflow + Outbox | Delivers saga result events |
| Release Tracker Client | Release Tracker | Calls repository APIs over HTTP |
| Use Cases | Domain Models | Uses subscription, subscriber, and saga models |
| Use Cases | Workflow + Outbox | Starts confirmation saga through a port |
| Use Cases | Repository | Reads and writes subscription state |
| Workflow + Outbox | Repository | Persists saga and outbox state |
| Workflow + Outbox | RabbitMQ Adapter | Publishes confirmation events |
| Repository | Subscription DB | Reads and writes data over SQL |
| RabbitMQ Adapter | RabbitMQ | Uses AMQP |

## Level 3: Release Tracker Components

The release tracker owns repository tracking and scanning. The subscription query client is selected in the composition root; gRPC is the current default and HTTP remains available for comparison.

```mermaid
C4Component
    title Component Diagram for Release Tracker

    Container(subscription, "Subscription Service", "Go service", "Active subscribers")
    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification events")
    System_Ext(github, "GitHub API", "Release metadata")

    Container_Boundary(releaseTracker, "Release Tracker") {
        Component(httpTransport, "HTTP Transport", "Chi / Go", "Repository API, health, metrics")
        Component(scanner, "Worker Scheduler", "Go worker", "Periodic release scans")
        Component(subscriptionClient, "Subscription Client", "Go gRPC / HTTP", "Active subscriptions")
        Component(githubClient, "GitHub Client", "Go HTTP client", "Repository releases")
        Component(publisher, "RabbitMQ Publisher", "AMQP", "Release notification jobs")

        Component(releaseUsecaseSpacerLeft, " ", " ", " ")
        Component(releaseUsecaseSpacerMidLeft, " ", " ", " ")
        Component(usecases, "Use Cases", "Go", "Ensure repo, get repo, scan releases")
        Component(releaseUsecaseSpacerMidRight, " ", " ", " ")
        Component(releaseUsecaseSpacerRight, " ", " ", " ")

        Component(scanWorkflow, "Scan Workflow", "Go", "Finds release deltas and plans notifications")
        Component(domain, "Domain Models", "Go", "Repository and subscriber models")
        Component(repository, "Repository", "Go / SQL", "Tracked repositories")
    }

    ContainerDb(releaseDb, "Release Tracker DB", "PostgreSQL", "Tracked repositories")

    Rel_D(httpTransport, usecases, "")
    Rel_D(scanner, usecases, "")
    Rel_D(subscriptionClient, subscription, "")
    Rel_D(githubClient, github, "")
    Rel_D(publisher, rabbitmq, "")
    Rel_R(usecases, domain, "")
    Rel_D(usecases, scanWorkflow, "")
    Rel_D(usecases, repository, "")
    Rel_D(scanWorkflow, repository, "")
    Rel_R(usecases, githubClient, "")
    Rel_D(usecases, subscriptionClient, "")
    Rel_R(usecases, publisher, "")
    Rel_D(repository, releaseDb, "")

    UpdateLayoutConfig($c4ShapeInRow="5", $c4BoundaryInRow="1")
    UpdateElementStyle(releaseUsecaseSpacerLeft, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(releaseUsecaseSpacerMidLeft, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(releaseUsecaseSpacerMidRight, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(releaseUsecaseSpacerRight, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
```

| From | To | Interaction |
| --- | --- | --- |
| HTTP Transport | Use Cases | Calls repository ensure/read use cases |
| Worker Scheduler | Use Cases | Triggers periodic release scans |
| Subscription Client | Subscription Service | Calls gRPC by default; HTTP client is retained |
| GitHub Client | GitHub API | Calls GitHub over HTTPS |
| RabbitMQ Publisher | RabbitMQ | Publishes release notification jobs over AMQP |
| Use Cases | Domain Models | Uses repository and active-subscriber models |
| Use Cases | Scan Workflow | Finds release deltas and plans notifications |
| Use Cases | Repository | Reads and writes tracked repositories |
| Scan Workflow | Repository | Persists scan progress |
| Use Cases | GitHub Client | Checks repository existence and latest tags |
| Use Cases | Subscription Client | Requests active subscribers for changed repositories |
| Use Cases | RabbitMQ Publisher | Publishes release notification jobs |
| Repository | Release Tracker DB | Reads and writes data over SQL |

## Level 3: Notification Worker Components

The notification worker has no database. It consumes durable events, sends emails, and reports confirmation success or failure back to the subscription saga.

```mermaid
C4Component
    title Component Diagram for Notification Worker

    ContainerQueue(rabbitmq, "RabbitMQ", "AMQP", "Notification and saga events")
    System_Ext(smtp, "SMTP Server / Mailpit", "Email delivery")

    Container_Boundary(notificationWorker, "Notification Worker") {
        Component(consumer, "RabbitMQ Consumer", "AMQP", "Decodes notification events")
        Component(resultPublisher, "Saga Result Publisher", "AMQP", "Reports confirmation outcome")
        Component(emailClient, "Email Client", "SMTP", "Sends with retry policy")
        Component(builder, "Message Builder", "Go", "Builds email content")
        Component(notificationTopSpacer, " ", " ", " ")

        Component(notificationServiceSpacerLeft, " ", " ", " ")
        Component(notificationServiceSpacerMidLeft, " ", " ", " ")
        Component(service, "Notification Service", "Go", "Coordinates delivery")
        Component(notificationServiceSpacerMidRight, " ", " ", " ")
        Component(notificationServiceSpacerRight, " ", " ", " ")
    }

    Rel_D(rabbitmq, consumer, "")
    Rel_D(consumer, service, "")
    Rel_D(emailClient, smtp, "")
    Rel_D(resultPublisher, rabbitmq, "")
    Rel_R(service, builder, "")
    Rel_D(service, emailClient, "")
    Rel_R(service, resultPublisher, "")

    UpdateLayoutConfig($c4ShapeInRow="5", $c4BoundaryInRow="1")
    UpdateElementStyle(notificationTopSpacer, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(notificationServiceSpacerLeft, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(notificationServiceSpacerMidLeft, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(notificationServiceSpacerMidRight, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
    UpdateElementStyle(notificationServiceSpacerRight, $bgColor="transparent", $fontColor="transparent", $borderColor="transparent")
```

| From | To | Interaction |
| --- | --- | --- |
| RabbitMQ | RabbitMQ Consumer | Delivers notification events over AMQP |
| RabbitMQ Consumer | Notification Service | Dispatches decoded events |
| Email Client | SMTP Server / Mailpit | Sends email over SMTP |
| Saga Result Publisher | RabbitMQ | Publishes saga result events over AMQP |
| Notification Service | Message Builder | Builds email subject and body |
| Notification Service | Email Client | Sends email requests |
| Notification Service | Saga Result Publisher | Reports confirmation outcome |
