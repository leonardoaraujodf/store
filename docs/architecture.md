# Architecture

## Product Goal

The project is an Amazon-like backend built as a sequence of small, deployable learning exercises. It starts with product catalog capabilities and evolves into a marketplace workflow without prematurely scaffolding empty services.

## Service Evolution

| Milestone | Service or capability | Primary responsibility |
| --- | --- | --- |
| 1–2 | Catalog | Products and categories; product lookup and listing |
| 4 | Inventory | Stock availability and inventory changes |
| 5 | Cart and Order | Customer carts and order creation |
| 6 | Payment | Simulated payment authorization and outcome events |
| Later | Identity | Authentication/authorization when a concrete requirement needs it |

Each service is introduced only when the preceding capability is working and understood. A service owns its database schema and never reads or writes another service's database.

## Hexagonal Design

Every service keeps business rules independent from delivery and infrastructure:

- **Domain:** entities, value objects, invariants, and domain errors.
- **Application:** use cases that coordinate domain behavior through ports.
- **Ports:** interfaces required by application code, such as repositories, event publishers, clocks, or external clients.
- **Adapters:** gRPC handlers/clients, PostgreSQL repositories, Kafka producers/consumers, configuration, and telemetry implementations.

Dependencies point inward. Domain code does not import gRPC, database, Kafka, or Kubernetes packages. Application code depends on port interfaces rather than concrete adapters.

## Communication and Data

- gRPC is the synchronous API between services and for service-level client access.
- Protobuf files define versioned service contracts. Additive changes are preferred; breaking changes require an explicit compatibility plan.
- Kafka is introduced with the first cross-service workflow. Events describe facts that already happened and are owned by their producing service.
- PostgreSQL is the first persistence choice. Redis remains a later optimization, not a default dependency.
- A service reacts to remote data through its API or events; it does not join or query another service's storage.

## Runtime Strategy

Develop and test locally first with Docker Compose. Kubernetes is introduced after service contracts and local workflows are stable. Health checks, metrics, tracing, and structured logs are part of the production-shaped design, introduced in the observability milestone rather than faked at project start.
