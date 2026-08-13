# Catalog PostgreSQL Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist valid Catalog products through the existing repository port using PostgreSQL, migrations, and real-database integration tests.

**Architecture:** Compose provides a long-running PostgreSQL database and a one-shot migration job. The PostgreSQL adapter depends outward on `pgxpool` while the existing application/domain packages remain database-free; tagged integration tests prove the adapter writes the expected row.

**Tech Stack:** Go 1.26.5, PostgreSQL, Docker Compose, `migrate/migrate` v4 image, `github.com/jackc/pgx/v5`.

## Global Constraints

- Work only in `/home/leonardo/Development/store/.worktrees/issue_5` on `issue_5`.
- Catalog owns its migrations under `services/catalog/migrations`; no other service may use this schema.
- Keep `make check` database-free. Only `make test-integration` starts Docker Compose and runs tagged tests.
- Do not add gRPC, a Catalog executable, retrieval/listing use cases, categories, Kafka, Kubernetes, or CI container tests.

---

## File structure

| Path | Responsibility |
| --- | --- |
| `compose.yaml` | PostgreSQL and one-shot migration services. |
| `services/catalog/migrations/000001_create_products.{up,down}.sql` | Catalog-owned schema evolution. |
| `services/catalog/internal/adapter/postgres/product_repository.go` | `ProductRepository.Save` implementation. |
| `services/catalog/internal/adapter/postgres/product_repository_integration_test.go` | Tagged real-PostgreSQL adapter test. |
| `Makefile` | Database, migration, and integration-test commands. |
| `README.md` | Local database workflow. |

## Task 1: Local PostgreSQL and schema migrations

**Files:**
- Create: `compose.yaml`
- Create: `services/catalog/migrations/000001_create_products.up.sql`
- Create: `services/catalog/migrations/000001_create_products.down.sql`
- Modify: `Makefile`

- [X] **Step 1: Confirm Docker and Compose are available**

Run: `docker version && docker compose version`

Expected: both commands exit 0. Do not continue until the Docker daemon is running.

- [X] **Step 2: Add the first migration pair**

`000001_create_products.up.sql`:

```sql
CREATE TABLE products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    price_minor_units BIGINT NOT NULL,
    currency CHAR(3) NOT NULL
);
```

`000001_create_products.down.sql`:

```sql
DROP TABLE products;
```

- [X] **Step 3: Add Compose services**

Create `postgres` with `postgres:17-alpine`, database/user/password `catalog`, exposed port `5432`, named volume `catalog-postgres-data`, and a `pg_isready -U catalog -d catalog` health check. Create a `migrate` service using `migrate/migrate:v4.18.3`, mounting `./services/catalog/migrations:/migrations:ro`, depending on healthy `postgres`, and running:

```text
-path=/migrations -database=postgres://catalog:catalog@postgres:5432/catalog?sslmode=disable up
```

- [X] **Step 4: Add lifecycle commands**

Append these targets to `.PHONY` and implement them:

```make
db-up:
	docker compose up -d --wait postgres

migrate-up:
	docker compose run --rm migrate

db-down:
	docker compose down

db-reset:
	docker compose down -v
	$(MAKE) db-up
	$(MAKE) migrate-up
```

- [X] **Step 5: Verify database and migration lifecycle**

Run: `make db-reset && docker compose exec -T postgres psql -U catalog -d catalog -c '\\d products'`

Expected: PostgreSQL is healthy and the table has exactly the five planned columns.

- [X] **Step 6: Commit schema and local database workflow**

Run: `git add compose.yaml Makefile services/catalog/migrations && git commit -m "build: add Catalog PostgreSQL migrations"`

## Task 2: PostgreSQL repository adapter

**Files:**
- Create: `services/catalog/internal/adapter/postgres/product_repository.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Implements: `port.ProductRepository` with `Save(context.Context, product.Product) error`.
- Produces: `func NewProductRepository(pool *pgxpool.Pool) ProductRepository`.

- [X] **Step 1: Add the pgx dependency**

Run: `go get github.com/jackc/pgx/v5/pgxpool && go mod tidy`

Expected: `go.mod` and `go.sum` record the exact resolved versions.

- [X] **Step 2: Write the failing integration test**

Create a build-tagged test file (`//go:build integration`) that opens `CATALOG_DATABASE_URL`, constructs the adapter, saves a `product.New(...)` result, then queries `products` directly and compares `id`, `name`, `description`, `price_minor_units`, and `currency`.

Run: `CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' go test -tags=integration ./services/catalog/internal/adapter/postgres`

Expected: compile failure because the adapter does not exist.

- [X] **Step 3: Implement the smallest adapter**

```go
type ProductRepository struct{ pool *pgxpool.Pool }

func NewProductRepository(pool *pgxpool.Pool) ProductRepository { return ProductRepository{pool: pool} }

func (r ProductRepository) Save(ctx context.Context, p product.Product) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO products (id, name, description, price_minor_units, currency) VALUES ($1, $2, $3, $4, $5)`, p.ID, p.Name, p.Description, p.PriceMinorUnits, p.Currency)
	return err
}
```

- [X] **Step 4: Make test state deterministic and verify green**

At the beginning of the test, execute `TRUNCATE products`; use `t.Cleanup` to close the pool. Run the focused tagged test again and confirm it passes.

- [X] **Step 5: Commit the adapter and integration test**

Run: `git add go.mod go.sum services/catalog/internal/adapter/postgres && git commit -m "feat(catalog): persist products with PostgreSQL"`

## Task 3: One integration-test command and documentation

**Files:**
- Modify: `Makefile`
- Modify: `README.md`

- [X] **Step 1: Add the integration command**

```make
test-integration: db-up migrate-up
	CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' go test -tags=integration ./services/catalog/internal/adapter/postgres
```

- [X] **Step 2: Document the workflow**

Add a README section documenting `make db-up`, `make migrate-up`, `make test-integration`, `make db-down`, and `make db-reset`. State that `make check` does not start Docker or run integration tests.

- [X] **Step 3: Verify both quality paths**

Run: `make check && make test-integration && git diff --check`

Expected: unit quality checks remain database-free; tagged adapter integration test passes against Compose PostgreSQL.

- [X] **Step 4: Commit, then stop the local database if desired**

Run: `git add Makefile README.md && git commit -m "docs: add Catalog database workflow" && make db-down`

Expected: branch has three functional commits and the database volume remains for future local work.
