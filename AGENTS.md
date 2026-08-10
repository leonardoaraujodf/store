# Online Store Backend

## Purpose

This is a learning-by-doing backend for an Amazon-like online store. It uses Go, hexagonal architecture, microservices, gRPC, Kubernetes, Kafka, and observability tooling. Work progresses in small milestones so concepts are learned before boilerplate is automated.

## Working Agreement

- The human implements concept-heavy work when they want to learn it. Provide explanations, review, and requested boilerplate.
- Plan a feature before implementation. Track each independently testable feature in a GitHub Issue in `leonardoaraujodf/store`.
- One issue may contain several small tasks. Keep issues focused on a single vertical capability.
- Preserve existing user changes. Do not add services, dependencies, infrastructure, or external integrations beyond the active issue without agreement.

## Architecture Rules

- Introduce services incrementally: Catalog, Inventory, Cart, Order, Payment, then Identity only if required.
- Each service owns its business logic and data. Do not share databases or domain models across services.
- Use hexagonal boundaries: domain and application code depend on ports; gRPC, PostgreSQL, Kafka, and observability are adapters.
- Use gRPC for synchronous service APIs and Kafka for asynchronous domain events. Keep protobuf contracts backward compatible.
- PostgreSQL is the initial datastore. Introduce Redis only for a measured, documented need.

## Quality Rules

- Apply TDD to every behavior change: write one failing test, verify the expected failure, write the minimum code to pass, then refactor with tests green.
- Add unit tests for domain/application behavior. Add integration tests whenever a real adapter, database, broker, or network boundary is introduced.
- Run formatting, static analysis, tests, and generated-code checks before considering a feature complete.
- Early CI reports checks without blocking merges while the tooling is being learned. Tighten enforcement only by an explicit project decision.

## Documentation

- Read `docs/architecture.md` for the system model and milestone boundaries.
- Read `docs/development-workflow.md` for TDD, testing, and issue conventions.
- Read `docs/quality-and-delivery.md` for CI/CD and quality-gate strategy.
- Read `docs/roadmap.md` for the delivery order.
