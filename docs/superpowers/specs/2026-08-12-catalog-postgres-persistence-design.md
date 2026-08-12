# Catalog PostgreSQL persistence design

## Goal

Implement the existing Catalog `ProductRepository` port with PostgreSQL. The slice introduces local PostgreSQL, versioned Catalog-owned migrations, and a real database integration test without adding a Catalog process or network API.

## Boundaries

```text
CreateProduct use case → ProductRepository port ← PostgreSQL repository adapter
                                                   ↓
                                            Catalog PostgreSQL schema
```

The application and domain packages remain unchanged by database-specific types. The adapter lives under `services/catalog/internal/adapter/postgres` and depends outward on a PostgreSQL client and SQL. It implements `Save(context.Context, product.Product) error`.

The integration test uses the adapter and a real database. It may query the table directly to verify persistence because it tests the adapter boundary; it must not add a product-read application port or use case.

## Local database and migration workflow

Root Docker Compose defines two services:

- `postgres`: a pinned PostgreSQL image, a named local volume, port `5432`, and a health check using `pg_isready`.
- `migrate`: a short-lived service using a pinned `migrate/migrate` image. It mounts Catalog migration files read-only and applies all pending migrations, then exits. It is not an application service and has no persistent process.

The migration runner connects to `postgres` through the Compose service hostname. Its command uses `golang-migrate` version 4 and a filesystem migration source. Local commands must start PostgreSQL before running migrations.

## Schema and migrations

Catalog owns migrations at:

```text
services/catalog/migrations/
  000001_create_products.up.sql
  000001_create_products.down.sql
```

The first up migration creates `products` with:

| Column | PostgreSQL type | Constraint |
| --- | --- | --- |
| `id` | `text` | primary key |
| `name` | `text` | not null |
| `description` | `text` | not null |
| `price_minor_units` | `bigint` | not null |
| `currency` | `char(3)` | not null |

The matching down migration drops only `products`. Domain validation remains the authority for product invariants; the database schema provides persistence constraints, not duplicate business validation.

## Configuration and adapter

Use `github.com/jackc/pgx/v5` with `pgxpool` as the PostgreSQL client. The adapter constructor receives a `*pgxpool.Pool`; connection configuration stays at the application composition boundary and is not introduced in this issue because no Catalog executable exists.

`Save` issues a parameterized `INSERT` for all product fields and returns the database error unchanged. The schema's primary key makes duplicate IDs fail through the same error path. Mapping database errors to application/domain errors is deferred until a caller has a concrete behavior requirement.

## Commands and tests

Add Makefile commands:

```text
db-up              Start PostgreSQL and wait for its health check.
migrate-up         Run the one-shot migration service.
db-down            Stop Compose services; retain the local database volume.
db-reset            Stop services and remove the local volume, then start and migrate a clean database.
test-integration   Start the database, apply migrations, then run Go tests tagged `integration`.
```

Integration tests are build-tagged (`//go:build integration`) and use `CATALOG_DATABASE_URL`, defaulting only in local Makefile invocation to the Compose-exposed localhost URL. `make check` stays database-free and does not execute integration tests.

The test must save a valid product with the adapter and verify the persisted row's five fields using a direct SQL query. It cleans database state deterministically before or after the test so it can be rerun against the same local volume.

## CI and scope

This issue adds the local Compose-based integration workflow only. CI continues to execute `make check`; container-backed integration tests will be added to CI in a later, dedicated delivery-quality issue.

Out of scope: gRPC/protobuf, product retrieval/listing use cases, categories, a Catalog executable, Kafka, Kubernetes, observability, `golangci-lint`, `govulncheck`, race tests, coverage, and merge-blocking rules.
