# Architecture

This document describes the current architecture of GitHub Release Notifier using the C4 model. The diagrams reflect the Docker Compose deployment and the Go modules under `services/` and `shared/`. The diagrams use Mermaid flowcharts for better rendering compatibility in Markdown, while the section levels still follow the C4 model.

## Diagram Legend

```mermaid
flowchart LR
    person(["Person / Client"])
    service["Service / Container"]
    app["Application Core"]
    domain["Domain Model"]
    adapter["Infrastructure Adapter"]
    database[("Database")]
    queue[("Queue / Broker")]
    external["External System"]

    classDef personCls fill:#fff2cc,stroke:#d6b656,color:#111827
    classDef serviceCls fill:#dae8fc,stroke:#6c8ebf,color:#111827
    classDef appCls fill:#dbeafe,stroke:#2563eb,color:#111827
    classDef domainCls fill:#fff7ed,stroke:#c2410c,color:#111827
    classDef infraCls fill:#f5f5f5,stroke:#666666,color:#111827
    classDef dbCls fill:#d5e8d4,stroke:#82b366,color:#111827
    classDef queueCls fill:#ffe6cc,stroke:#d79b00,color:#111827
    classDef externalCls fill:#e5e7eb,stroke:#6b7280,color:#111827
    class person personCls
    class service serviceCls
    class app appCls
    class domain domainCls
    class adapter infraCls
    class database dbCls
    class queue queueCls
    class external externalCls
```

## Level 1: System Context

The system lets users subscribe to GitHub repositories, confirms subscriptions by email, periodically checks GitHub releases, and sends release notifications.

```mermaid
flowchart LR
    user["Subscriber<br/>Subscribes to public GitHub repositories"]
    system["GitHub Release Notifier<br/>Tracks releases and emails subscribers"]
    github["GitHub API<br/>Repository and release data"]
    smtp["SMTP Server / Mailpit<br/>Email delivery"]
    observability["Observability Tools<br/>Metrics and logs"]

    user -->|"Uses web UI and subscription endpoints<br/>HTTP"| system
    system -->|"Reads repository and release data<br/>HTTPS/REST"| github
    system -->|"Sends confirmation and release emails<br/>SMTP"| smtp
    observability -->|"Reads metrics and logs"| system

    classDef personCls fill:#fff2cc,stroke:#d6b656,color:#111827
    classDef systemCls fill:#dae8fc,stroke:#6c8ebf,color:#111827
    classDef externalCls fill:#e5e7eb,stroke:#6b7280,color:#111827
    class user personCls
    class system systemCls
    class github,smtp,observability externalCls
```

## Level 2: Containers

The application is split into independently deployable services. Each stateful service owns its database. Cross-service business events use RabbitMQ, while synchronous repository and subscription queries use HTTP or gRPC.

```mermaid
flowchart LR
    user["Subscriber"]
    github["GitHub API<br/>Repository releases"]
    smtp["SMTP Server / Mailpit<br/>Email delivery"]

    subgraph app["GitHub Release Notifier"]
        direction TB
        subscription["Subscription Service<br/>Go, Chi, gRPC<br/>UI, subscription API, confirmation sagas, outbox"]
        releaseTracker["Release Tracker<br/>Go, Chi<br/>Repository tracking and GitHub polling"]
        notificationWorker["Notification Worker<br/>Go<br/>Email delivery from durable events"]
        subscriptionDb[("Subscription DB<br/>PostgreSQL<br/>Subscribers, subscriptions, sagas, outbox")]
        releaseDb[("Release Tracker DB<br/>PostgreSQL<br/>Tracked repositories and last seen tags")]
        rabbitmq[("RabbitMQ<br/>AMQP<br/>Notification, saga, and DLQ queues")]
    end

    user -->|"Uses static UI and public API<br/>HTTP"| subscription
    subscription -->|"Ensures repositories and reads metadata<br/>HTTP"| releaseTracker
    releaseTracker -->|"Reads active subscribers<br/>gRPC"| subscription
    subscription -->|"Reads/Writes<br/>SQL"| subscriptionDb
    releaseTracker -->|"Reads/Writes<br/>SQL"| releaseDb
    releaseTracker -->|"Polls latest release tags<br/>HTTPS"| github
    subscription -->|"Publishes confirmation events<br/>AMQP"| rabbitmq
    rabbitmq -->|"Delivers saga results<br/>AMQP"| subscription
    releaseTracker -->|"Publishes release notification events<br/>AMQP"| rabbitmq
    rabbitmq -->|"Delivers notification events<br/>AMQP"| notificationWorker
    notificationWorker -->|"Sends emails<br/>SMTP"| smtp

    classDef personCls fill:#fff2cc,stroke:#d6b656,color:#111827
    classDef serviceCls fill:#dae8fc,stroke:#6c8ebf,color:#111827
    classDef dbCls fill:#d5e8d4,stroke:#82b366,color:#111827
    classDef queueCls fill:#ffe6cc,stroke:#d79b00,color:#111827
    classDef externalCls fill:#e5e7eb,stroke:#6b7280,color:#111827
    class user personCls
    class subscription,releaseTracker,notificationWorker serviceCls
    class subscriptionDb,releaseDb dbCls
    class rabbitmq queueCls
    class github,smtp externalCls
    style app fill:#ffffff,stroke:#94a3b8,color:#111827
```

Observability containers are available through the Compose `observability` profile and scrape `/metrics` from the Go services.

## Level 3: Subscription Service Components

The subscription service uses vertical module slices. Transport adapters call use cases, use cases depend on ports, and infrastructure adapters are wired in the composition root.

```mermaid
flowchart TB
    releaseTracker["Release Tracker<br/>Go service<br/>Repository metadata"]
    rabbitmq[("RabbitMQ<br/>AMQP<br/>Events")]
    subscriptionDb[("Subscription DB<br/>PostgreSQL<br/>Subscription data")]

    subgraph subscription["Subscription Service"]
        direction TB
        httpTransport["HTTP Transport<br/>Chi / Go<br/>Public API, static UI, health, metrics"]
        grpcTransport["gRPC Transport<br/>Go gRPC<br/>Active subscription query"]
        eventbus["RabbitMQ Adapter<br/>AMQP<br/>Confirmation and saga events"]
        releaseClient["Release Tracker Client<br/>Go HTTP client<br/>Repository metadata"]
        usecases["Use Cases<br/>Go<br/>Subscribe, confirm, unsubscribe, list"]
        workflows["Workflow + Outbox<br/>Go<br/>Confirmation saga and outbox relay"]
        domain["Domain Models<br/>Go<br/>Subscriber, subscription, saga models"]
        repository["Repository<br/>Go / SQL<br/>Subscribers, subscriptions, sagas, outbox"]
    end

    httpTransport -->|"Calls subscription use cases"| usecases
    grpcTransport -->|"Calls active-subscription query"| usecases
    eventbus -->|"Delivers saga result events"| workflows
    releaseClient -->|"Calls repository APIs<br/>HTTP"| releaseTracker
    usecases -->|"Uses models"| domain
    usecases -->|"Starts confirmation saga through port"| workflows
    usecases -->|"Reads/Writes subscription state"| repository
    workflows -->|"Persists saga and outbox state"| repository
    workflows -->|"Publishes confirmation events"| eventbus
    repository -->|"Reads/Writes<br/>SQL"| subscriptionDb
    eventbus -->|"Uses<br/>AMQP"| rabbitmq

    classDef externalCls fill:#e5e7eb,stroke:#6b7280,color:#111827
    classDef transportCls fill:#dae8fc,stroke:#6c8ebf,color:#111827
    classDef appCls fill:#dbeafe,stroke:#2563eb,color:#111827
    classDef domainCls fill:#fff7ed,stroke:#c2410c,color:#111827
    classDef infraCls fill:#f5f5f5,stroke:#666666,color:#111827
    classDef dbCls fill:#d5e8d4,stroke:#82b366,color:#111827
    classDef queueCls fill:#ffe6cc,stroke:#d79b00,color:#111827
    class releaseTracker externalCls
    class httpTransport,grpcTransport transportCls
    class usecases,workflows appCls
    class domain domainCls
    class repository,eventbus,releaseClient infraCls
    class subscriptionDb dbCls
    class rabbitmq queueCls
    style subscription fill:#ffffff,stroke:#94a3b8,color:#111827
```

## Level 3: Release Tracker Components

The release tracker owns repository tracking and scanning. The subscription query client is selected in the composition root; gRPC is the current default and HTTP remains available for comparison.

```mermaid
flowchart TB
    subscription["Subscription Service<br/>Go service<br/>Active subscribers"]
    rabbitmq[("RabbitMQ<br/>AMQP<br/>Notification events")]
    github["GitHub API<br/>Release metadata"]
    releaseDb[("Release Tracker DB<br/>PostgreSQL<br/>Tracked repositories")]

    subgraph releaseTracker["Release Tracker"]
        direction TB
        httpTransport["HTTP Transport<br/>Chi / Go<br/>Repository API, health, metrics"]
        scanner["Worker Scheduler<br/>Go worker<br/>Periodic release scans"]
        subscriptionClient["Subscription Client<br/>Go gRPC / HTTP<br/>Active subscriptions"]
        githubClient["GitHub Client<br/>Go HTTP client<br/>Repository releases"]
        publisher["RabbitMQ Publisher<br/>AMQP<br/>Release notification jobs"]
        usecases["Use Cases<br/>Go<br/>Ensure repo, get repo, scan releases"]
        domain["Domain Models<br/>Go<br/>Repository and subscriber models"]
        repository["Repository<br/>Go / SQL<br/>Tracked repositories"]
    end

    httpTransport -->|"Calls repository ensure/read use cases"| usecases
    scanner -->|"Triggers periodic release scans"| usecases
    subscriptionClient -->|"Calls active-subscription query<br/>gRPC by default"| subscription
    githubClient -->|"Calls GitHub<br/>HTTPS"| github
    publisher -->|"Publishes notification jobs<br/>AMQP"| rabbitmq
    usecases -->|"Uses models"| domain
    usecases -->|"Reads/Writes tracked repositories"| repository
    usecases -->|"Checks existence and latest tags"| githubClient
    usecases -->|"Requests active subscribers"| subscriptionClient
    usecases -->|"Publishes release notifications"| publisher
    repository -->|"Reads/Writes<br/>SQL"| releaseDb

    classDef externalCls fill:#e5e7eb,stroke:#6b7280,color:#111827
    classDef transportCls fill:#dae8fc,stroke:#6c8ebf,color:#111827
    classDef appCls fill:#dbeafe,stroke:#2563eb,color:#111827
    classDef domainCls fill:#fff7ed,stroke:#c2410c,color:#111827
    classDef infraCls fill:#f5f5f5,stroke:#666666,color:#111827
    classDef dbCls fill:#d5e8d4,stroke:#82b366,color:#111827
    classDef queueCls fill:#ffe6cc,stroke:#d79b00,color:#111827
    class subscription,github externalCls
    class httpTransport transportCls
    class usecases,scanner appCls
    class domain domainCls
    class repository,subscriptionClient,githubClient,publisher infraCls
    class releaseDb dbCls
    class rabbitmq queueCls
    style releaseTracker fill:#ffffff,stroke:#94a3b8,color:#111827
```

## Level 3: Notification Worker Components

The notification worker has no database. It consumes durable events, sends emails, and reports confirmation success or failure back to the subscription saga.

```mermaid
flowchart TB
    rabbitmq[("RabbitMQ<br/>AMQP<br/>Notification and saga events")]
    smtp["SMTP Server / Mailpit<br/>Email delivery"]

    subgraph notificationWorker["Notification Worker"]
        direction TB
        consumer["RabbitMQ Consumer<br/>AMQP<br/>Decodes notification events"]
        resultPublisher["Saga Result Publisher<br/>AMQP<br/>Reports confirmation outcome"]
        emailClient["Email Client<br/>SMTP<br/>Sends with retry policy"]
        builder["Message Builder<br/>Go<br/>Builds email content"]
        service["Notification Service<br/>Go<br/>Coordinates delivery"]
    end

    rabbitmq -->|"Delivers notification events<br/>AMQP"| consumer
    consumer -->|"Dispatches decoded events"| service
    service -->|"Builds email subject and body"| builder
    service -->|"Sends email requests"| emailClient
    service -->|"Reports confirmation outcome"| resultPublisher
    emailClient -->|"Sends email<br/>SMTP"| smtp
    resultPublisher -->|"Publishes saga result events<br/>AMQP"| rabbitmq

    classDef appCls fill:#dbeafe,stroke:#2563eb,color:#111827
    classDef infraCls fill:#f5f5f5,stroke:#666666,color:#111827
    classDef queueCls fill:#ffe6cc,stroke:#d79b00,color:#111827
    classDef externalCls fill:#e5e7eb,stroke:#6b7280,color:#111827
    class service appCls
    class consumer,resultPublisher,emailClient,builder infraCls
    class rabbitmq queueCls
    class smtp externalCls
    style notificationWorker fill:#ffffff,stroke:#94a3b8,color:#111827
```

## Code Architecture Style

The system is split into independently deployable services. Each service owns its runtime process and, where needed, its own PostgreSQL database. Inside the stateful services, code is organized around business capabilities in a style similar to vertical slices. Each slice keeps its handlers, use cases, workflows, domain models, and persistence close together, while dependencies still point inward toward the application and domain code.

This is a pragmatic clean architecture / ports-and-adapters approach rather than a strict framework. `subscription-service` and `release-tracker` keep business slices under `internal/modules/...`. `notification-worker` is intentionally simpler: it has one application service in `internal/notification`, with SMTP and RabbitMQ adapters around it.

The intended dependency direction is inward:

| Layer | Responsibility | May depend on |
| --- | --- | --- |
| `cmd/server` | Composition root: loads config, creates databases, clients, adapters, use cases, servers, and workers | Any service-local layer |
| HTTP/gRPC transports and API routers | Accept external requests and translate transport DTOs into use-case calls | Use cases, domain models, support packages |
| Use cases and application services | Implement application behavior and define the ports they need | Domain models, local support packages, allowed shared event contracts |
| Workflows | Coordinate longer business processes such as confirmation saga and outbox publishing | Domain models, local ports, allowed shared event contracts |
| Domain models | Represent business state and rules | Other domain code only |
| Persistence | Store and load domain state from PostgreSQL | Domain models, database support |
| Infrastructure adapters | Implement ports for RabbitMQ, GitHub, SMTP, HTTP, and gRPC clients | Ports or domain models they adapt to |
| Support packages | Configuration, database bootstrap, observability, schedulers, migrations, and static assets | Other support code and explicitly allowed local layers |

The key rule is that domain and application code do not point outward to concrete infrastructure. For example, a domain model must not import PostgreSQL, RabbitMQ, HTTP, gRPC, or Prometheus. Use cases should depend on interfaces such as repositories, publishers, or external clients, while `cmd/server` wires those interfaces to concrete adapters.

The service-local `.go-arch-lint.yml` files make these boundaries executable. They define the layer components, allowed import directions, and vendor-import policy. Outer layers may use vendor packages freely, while domain packages cannot use vendor packages and application-core packages only allow the shared event contracts they publish or consume.
