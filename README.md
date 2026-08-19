# Store

A learning-by-doing backend for an Amazon-like online store. The project uses Go, hexagonal architecture, incremental microservices, gRPC, Kubernetes, Kafka, PostgreSQL, and observability tooling.

## Current State

The project is in **Milestone 1: Foundation**. Catalog exposes versioned gRPC product creation and retrieval by ID, wired through its `CreateProduct`/`GetProduct` use cases and PostgreSQL repository adapter. REST, Kafka, product listing, and category management remain future work.

## Local quality checks

Go **1.26.5** is required. Run the complete local quality contract with:

```bash
make check
```

Individual commands are also available:

```bash
make fmt-check
make vet
make test
```

## Local Catalog database

Podman (with the `podman compose` command) is required for the PostgreSQL integration workflow; the Makefile defaults to it via `COMPOSE ?= podman compose`. Docker and Docker Compose work too — pass `make COMPOSE="docker compose" ...` to override. The commands below use a local PostgreSQL database named `catalog` and versioned Catalog migrations.

```bash
# Start PostgreSQL and wait until it is healthy.
make db-up

# Apply pending Catalog migrations, then exit.
make migrate-up

# Run the PostgreSQL repository integration test.
make test-integration

# Run the gRPC-to-PostgreSQL integration test.
make test-grpc-integration
```

`make test-integration` starts PostgreSQL, applies pending migrations, and runs the PostgreSQL repository integration test. `make test-grpc-integration` uses the same database workflow, starts an in-process gRPC server, and validates that a gRPC request persists a product through PostgreSQL. `make check` remains database-free and does not start Podman/Docker or run integration tests.

To stop containers while retaining local database data, run:

```bash
make db-down
```

To remove local database data, recreate PostgreSQL, and reapply migrations, run:

```bash
make db-reset
```

## Catalog gRPC contract

The versioned Catalog contract is at `proto/catalog/v1/catalog.proto`. Generated Go files in `gen/go` are committed: edit the protobuf source and regenerate; never edit generated files.

```bash
make proto-format
make proto-lint
make proto-generate
make proto-check
```

`make proto-check` verifies protobuf formatting and linting, regenerates Go code, and fails if committed generated files are stale.

## Run Catalog gRPC locally

Catalog requires `CATALOG_GRPC_ADDR` and `CATALOG_DATABASE_URL`; it exits clearly if either is absent or the database URL is invalid. For an optional local direnv setup:

```bash
cp .envrc.example .envrc
direnv allow
make db-up
make migrate-up
make run-catalog
```

`.envrc` is local and ignored. Production and other environments export the same variables; the Go program does not load `.envrc`.

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
