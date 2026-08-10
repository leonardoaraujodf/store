# Store

A learning-by-doing backend for an Amazon-like online store. The project uses Go, hexagonal architecture, incremental microservices, gRPC, Kubernetes, Kafka, PostgreSQL, and observability tooling.

## Current State

The project is in **Milestone 1: Foundation**. The first planned feature is a transport-free Catalog service skeleton with a TDD-tested start/stop lifecycle. No application code or infrastructure configuration has been added yet.

## Project Guides

Read these documents in order before starting work:

| Document | Use it for |
| --- | --- |
| [AGENTS.md](AGENTS.md) | Project-wide rules, architecture constraints, and collaboration expectations |
| [Architecture](docs/architecture.md) | Service boundaries, data ownership, communication, and runtime strategy |
| [Development workflow](docs/development-workflow.md) | Local issue tracking, mandatory TDD, test levels, and definition of done |
| [Quality and delivery](docs/quality-and-delivery.md) | Advisory CI, local quality checks, and the progressive delivery strategy |
| [Roadmap](docs/roadmap.md) | Milestone sequence and learning progression |

## Working Principles

- Implement one independently testable feature at a time; track it in `issues/` until GitHub Issues are in use.
- Every behavior change follows red, green, refactor. Use unit tests for domain/application behavior and integration tests at real adapter boundaries.
- Services own their data. Use gRPC for synchronous communication and Kafka for asynchronous domain events.
- Keep the initial setup small. Add infrastructure only when the active milestone needs it.

## Planned Progression

1. Foundation and a small Catalog service skeleton.
2. Catalog product/category behavior and PostgreSQL persistence.
3. Observability.
4. Inventory and Kafka events.
5. Cart and Order workflows.
6. Payment and reliability.
7. Kubernetes deployment.

