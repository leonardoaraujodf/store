# Catalog Product ID Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Repo-specific override:** Do not embed full code implementations in this plan (this deviates from writing-plans' usual per-step code blocks — see `AGENTS.md`). Steps state exact files, signatures, SQL, test names, and scenarios precisely enough to implement from, but the executor writes the actual code as a real, reviewable change to each file.

**Goal:** Replace client-supplied product IDs with PostgreSQL-generated, auto-incrementing `int64` IDs, so product identity is guaranteed unique by the database instead of trusted from the caller.

**Architecture:** `Product.ID` becomes `int64` end to end (domain, port, PostgreSQL column, `catalog.v1` contract). `product.New` no longer takes an `id` argument — a freshly constructed `Product` has `ID: 0` ("not yet persisted"). `port.ProductRepository.Save` returns the persisted product with its database-assigned ID, instead of returning only an error. PostgreSQL assigns the ID via `INSERT ... RETURNING id` against a `BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY` column. `CreateProductRequest` drops its `id` field; `GetProductRequest.id` and `Product.id` change type from `string` to `int64` — a deliberate, one-time exception to the contract-compatibility rule, since `catalog.v1` has no external consumers yet.

**Tech Stack:** Go 1.26.5, gRPC-Go, Protocol Buffers (Buf v2), pgx/pgxpool, PostgreSQL via Compose, `bufconn` for in-process gRPC tests.

**Spec:** `docs/superpowers/specs/2026-08-19-catalog-product-id-generation-design.md`

## Global Constraints

- ID type is `int64` at every layer — domain, repository port, PostgreSQL column, gRPC contract — no string conversion at any boundary.
- Domain/application packages must not import protobuf, gRPC, or PostgreSQL adapter packages.
- No product listing/pagination, categories, REST, Kafka, or new runtime infrastructure.
- `pgx.ErrNoRows` must be translated inside the PostgreSQL adapter; it must never escape into application or gRPC code (pre-existing `FindByID` contract, unchanged).
- `GetProduct`: a non-positive ID → `codes.InvalidArgument`; missing product → `codes.NotFound`; repository/database failure → `codes.Internal` with a generic message that discloses no database implementation detail.
- The existing `000001` migration is not edited; the ID change lands in a new `000002` migration.
- Deliberate, one-time exception to "keep protobuf contracts backward compatible," scoped to the `id` field/type change on `CreateProductRequest`, `GetProductRequest`, and `Product`.
- `make check` remains database-free. Run `make proto-format`, `make proto-lint`, `make proto-generate`, `make proto-check` after the contract change.
- Apply red, green, refactor to every behavior change.

---

## File structure

| Path | Responsibility |
| --- | --- |
| `proto/catalog/v1/catalog.proto` | Removes `CreateProductRequest.id` (reserved); changes `GetProductRequest.id` and `Product.id` to `int64`. |
| `gen/go/catalog/v1/*.pb.go` | Regenerated, committed Go types matching the contract change. |
| `services/catalog/internal/domain/product/product.go` | `Product.ID` becomes `int64`; `New` drops the `id` parameter; `ErrEmptyID` removed. |
| `services/catalog/internal/domain/product/product_test.go` | Construction tests updated for the new signature. |
| `services/catalog/internal/application/port/product_repository.go` | `Save` returns `(product.Product, error)`; `FindByID` takes `id int64`. |
| `services/catalog/internal/application/createproduct/create_product.go` | `Command` drops `ID`; `Execute` returns what `Save` returns. |
| `services/catalog/internal/application/createproduct/create_product_test.go` | Fake repository and test cases updated for the new port and command shape. |
| `services/catalog/internal/application/getproduct/get_product.go` | `Command.ID` becomes `int64`; new `ErrInvalidID` guard replaces the reused `product.ErrEmptyID` check. |
| `services/catalog/internal/application/getproduct/get_product_test.go` | Fake repository and test cases updated for the new port, command shape, and guard. |
| `services/catalog/migrations/000002_product_id_autoincrement.up.sql` / `.down.sql` | New migration converting `products.id` to `BIGINT GENERATED ALWAYS AS IDENTITY`. |
| `services/catalog/internal/adapter/postgres/product_repository.go` | `Save` uses `RETURNING id`; `FindByID` takes `int64`. |
| `services/catalog/internal/adapter/postgres/product_repository_integration_test.go` | Integration tests updated for DB-assigned IDs; new distinct-sequential-IDs test. |
| `services/catalog/internal/adapter/grpc/catalog_server.go` | `CreateProduct` mapping drops `ID`; `GetProduct` maps `int64`; error mapping uses `getproduct.ErrInvalidID`. |
| `services/catalog/internal/adapter/grpc/catalog_server_test.go` | Fake repository and test cases updated for the new request/response shapes. |
| `services/catalog/internal/adapter/grpc/catalog_server_integration_test.go` | End-to-end test flow updated to use DB-assigned IDs. |

`services/catalog/cmd/catalog/main.go` is not modified — `NewCatalogServer(createUseCase, getUseCase)`'s signature is unchanged, and `postgres.Repository` keeps satisfying `port.ProductRepository` once Task 6 lands.

---

### Task 1: Change the Catalog protobuf contract to server-owned int64 IDs

**Files:**
- Modify: `proto/catalog/v1/catalog.proto`
- Modify (generated, do not hand-edit): `gen/go/catalog/v1/catalog.pb.go`, `gen/go/catalog/v1/catalog_grpc.pb.go`

**Interfaces:**
- Produces: `catalogv1.CreateProductRequest` without an `Id` field (field number 1 reserved, name `"id"` reserved).
- Produces: `catalogv1.GetProductRequest{Id int64}`, `catalogv1.Product{Id int64, ...}` (field numbers unchanged from today; only the `id` field's wire type changes).

- [ ] **Step 1: Edit the proto contract**

In `proto/catalog/v1/catalog.proto`:
- In `message CreateProductRequest`, delete the `string id = 1;` line and add `reserved 1;` and `reserved "id";` inside the message body.
- In `message GetProductRequest`, change `string id = 1;` to `int64 id = 1;`.
- In `message Product`, change `string id = 1;` to `int64 id = 1;`.
- Leave every other field, field number, RPC, and message untouched.

- [ ] **Step 2: Format, lint, and generate**

Run:
```bash
make proto-format
make proto-lint
make proto-generate
```
Expected: all three exit zero. `gen/go/catalog/v1/catalog.pb.go` now declares `CreateProductRequest` without an `Id` field/getter, and `GetProductRequest.Id`/`Product.Id` as `int64` with matching `GetId() int64` getters.

- [ ] **Step 3: Verify no drift**

Run:
```bash
make proto-check
```
Expected: exits zero.

- [ ] **Step 4: Confirm the expected build break**

Run:
```bash
go build ./...
```
Expected: **fails.** `services/catalog/internal/adapter/grpc/catalog_server.go` still reads `request.GetId()` on a `CreateProductRequest` (method no longer exists) and assigns a `string` `p.ID` into the now-`int64` `catalogv1.Product.Id` field. This is expected and is resolved in Task 7 — it is not a regression to fix here. Confirm the failure is limited to that one package (and its importers) by also running:
```bash
go build ./gen/... ./services/catalog/internal/domain/... ./services/catalog/internal/application/... ./services/catalog/internal/adapter/postgres/... ./services/catalog/internal/config/...
```
Expected: succeeds. Do **not** include `./services/catalog/cmd/...` in that check — `main.go` imports `internal/adapter/grpc` directly, so it stays broken until Task 7 even though none of the packages above are affected.

- [ ] **Step 5: Commit**

```bash
git add proto/catalog/v1/catalog.proto gen/go
git commit -m "feat: change Catalog product IDs to server-owned int64"
```

---

### Task 2: Change the Product domain to server-assigned IDs

**Files:**
- Modify: `services/catalog/internal/domain/product/product.go`
- Modify: `services/catalog/internal/domain/product/product_test.go`

**Interfaces:**
- Produces: `product.Product{ID int64, Name, Description string, PriceMinorUnits int64, Currency string}`.
- Produces: `product.New(name, description string, priceMinorUnits int64, currency string) (Product, error)` — no `id` parameter. A returned `Product` has `ID: 0`.
- Removes: `product.ErrEmptyID`.

- [ ] **Step 1: Update the failing domain tests**

Edit `services/catalog/internal/domain/product/product_test.go`:
- `TestNewCreatesProductWithValidAttributes`: call `product.New("Keyboard", "Mechanical keyboard", 12_999, "BRL")` (drop the id argument); the expected `product.Product` has `ID: 0` and the same `Name`/`Description`/`PriceMinorUnits`/`Currency` as today.
- `TestNewRejectsInvalidAttributes`: drop the `id` field from the test-case struct and from every table row and from the `product.New(...)` call in the loop body; delete the `{"empty id", ...}` row entirely (there is no longer an id argument to be empty).

- [ ] **Step 2: Run the domain tests to verify they fail**

Run:
```bash
go test ./services/catalog/internal/domain/product/...
```
Expected: FAIL — `product.New` still takes 5 arguments; `ID: 0` doesn't match the current struct literal shape yet.

- [ ] **Step 3: Update the domain implementation**

Edit `services/catalog/internal/domain/product/product.go`:
- Change `Product.ID` from `string` to `int64`.
- Delete `var ErrEmptyID = errors.New("product id cannot be empty")`.
- Change `New` to `func New(name string, description string, priceMinorUnits int64, currency string) (Product, error)`, removing the `id`/`ErrEmptyID` parameter and check entirely, and removing `id` from the returned `Product` literal (so `ID` keeps its zero value, `0`).

- [ ] **Step 4: Run the domain tests to verify they pass**

Run:
```bash
go test -v ./services/catalog/internal/domain/product/...
```
Expected: PASS for both tests.

- [ ] **Step 5: Confirm the expected build breaks**

Run:
```bash
go build ./services/catalog/internal/domain/...
```
Expected: succeeds. Do **not** run `go build ./...` yet — `internal/adapter/grpc` is still broken from Task 1, and `internal/application/createproduct` and `internal/application/getproduct` now break too: `create_product.go` still calls `product.New` with 5 arguments (including `command.ID`), and `get_product.go` still references the now-removed `product.ErrEmptyID`. Both are expected and are resolved in Task 3 and Task 4 respectively.

- [ ] **Step 6: Commit**

```bash
git add services/catalog/internal/domain/product
git commit -m "feat: make Product IDs server-assigned in the domain"
```

---

### Task 3: Update the repository port and CreateProduct for server-assigned IDs

**Files:**
- Modify: `services/catalog/internal/application/port/product_repository.go`
- Modify: `services/catalog/internal/application/createproduct/create_product.go`
- Modify: `services/catalog/internal/application/createproduct/create_product_test.go`

**Interfaces:**
- Consumes: `product.New(name, description string, priceMinorUnits int64, currency string) (product.Product, error)`, `product.ErrEmptyName` (Task 2).
- Produces: `port.ProductRepository` requiring `Save(ctx context.Context, p product.Product) (product.Product, error)` and `FindByID(ctx context.Context, id int64) (product.Product, bool, error)`.
- Produces: `createproduct.Command{Name, Description string, PriceMinorUnits int64, Currency string}` (no `ID` field); `createproduct.UseCase.Execute` unchanged in shape (`ctx, Command) (product.Product, error)`), now returning whatever `Save` returns.

- [ ] **Step 1: Update the failing createproduct tests**

Edit `services/catalog/internal/application/createproduct/create_product_test.go`:
- Change `fakeRepository.Save` to signature `func (f *fakeRepository) Save(_ context.Context, p product.Product) (product.Product, error)`. It must assign `p.ID` sequentially starting at 1 (the next ID being `int64(len(f.saved)) + 1`) before appending to `f.saved` and returning the stored copy, `nil` — mirroring the identity semantics PostgreSQL will provide.
- Change `fakeRepository.FindByID` signature to `func (f *fakeRepository) FindByID(_ context.Context, id int64) (product.Product, bool, error)` (body unchanged: still always returns `product.Product{}, false, nil`).
- `TestUseCaseExecuteSavesValidProduct`: build `createproduct.Command{Name: "Keyboard", Description: "Mechanical Keyboard", PriceMinorUnits: 12_999, Currency: "BRL"}` (no `ID` field). Keep the existing assertions that `Execute`'s result equals `repository.saved[0]`, and add an assertion that `got.ID != 0`.
- `TestUseCaseExecuteDoesNotSaveInvalidProduct`: this test's `Command` already omits `Name`, so it is already an invalid-name case; change only its error assertion from `product.ErrEmptyID` to `product.ErrEmptyName` (there is no longer an ID-based invalidity — an empty name is now the invalid case this test exercises).
- `TestUseCaseExecuteReturnsRepositoryError`: change the `Command` literal to drop `ID`, and change `fakeRepository{saveErr: wantErr}`'s `Save` (from the shared fake) to return `product.Product{}, wantErr` when `saveErr` is set.

- [ ] **Step 2: Run the createproduct tests to verify they fail**

Run:
```bash
go test ./services/catalog/internal/application/createproduct/...
```
Expected: FAIL to compile — `fakeRepository` no longer satisfies the still-unchanged `port.ProductRepository` interface (signature mismatch), and `createproduct.Command` still has an `ID` field the test no longer sets consistently with production code.

- [ ] **Step 3: Update the repository port**

Edit `services/catalog/internal/application/port/product_repository.go`: change `Save(context.Context, product.Product) error` to `Save(context.Context, product.Product) (product.Product, error)`, and change `FindByID(ctx context.Context, id string) (product.Product, bool, error)` to `FindByID(ctx context.Context, id int64) (product.Product, bool, error)`.

- [ ] **Step 4: Update the createproduct use case**

Edit `services/catalog/internal/application/createproduct/create_product.go`:
- Remove `ID` from the `Command` struct.
- In `Execute`, call `product.New(command.Name, command.Description, command.PriceMinorUnits, command.Currency)` (drop `command.ID`).
- Replace the `if err := u.repository.Save(ctx, p); err != nil { ... }; return p, nil` block with capturing and returning `Save`'s own result: call `saved, err := u.repository.Save(ctx, p)`, return `product.Product{}, err` on error, else return `saved, nil`.

- [ ] **Step 5: Run the createproduct tests to verify they pass**

Run:
```bash
go test -v ./services/catalog/internal/application/createproduct/...
```
Expected: PASS for all three tests.

- [ ] **Step 6: Confirm scope of remaining breaks**

Run:
```bash
go build ./services/catalog/internal/application/createproduct/... ./services/catalog/internal/application/port/...
```
Expected: succeeds. Do **not** run `go build ./...` yet: `internal/application/getproduct` still implements the old `FindByID(ctx, string)` shape and references `product.ErrEmptyID` (broken since Task 2, resolved in Task 4); `internal/adapter/grpc` is still broken from Task 1; `internal/adapter/postgres` no longer satisfies the extended port (`Save`/`FindByID` signatures), resolved in Task 6.

- [ ] **Step 7: Commit**

```bash
git add services/catalog/internal/application/port \
        services/catalog/internal/application/createproduct
git commit -m "feat: return server-assigned IDs from CreateProduct"
```

---

### Task 4: Update GetProduct for int64 IDs and a dedicated invalid-ID guard

**Files:**
- Modify: `services/catalog/internal/application/getproduct/get_product.go`
- Modify: `services/catalog/internal/application/getproduct/get_product_test.go`

**Interfaces:**
- Consumes: `port.ProductRepository` (Task 3), `product.New` (Task 2).
- Produces: `getproduct.Command{ID int64}`; `getproduct.ErrInvalidID` (new, message `"product id must be positive"`); `getproduct.ErrProductNotFound` (unchanged); `getproduct.UseCase.Execute(ctx, Command) (product.Product, error)` returns `ErrInvalidID` when `command.ID <= 0`.

- [ ] **Step 1: Update the failing getproduct tests**

Edit `services/catalog/internal/application/getproduct/get_product_test.go`:
- Change `fakeRepository.products` from `map[string]product.Product` to `map[int64]product.Product`.
- Change `fakeRepository.Save` to signature `func (f *fakeRepository) Save(_ context.Context, p product.Product) (product.Product, error)`. It must assign `p.ID` sequentially starting at 1 (the next ID being `int64(len(f.products)) + 1`) before storing it in the map and returning the stored copy, `nil`.
- Change `fakeRepository.FindByID` signature to `func (f *fakeRepository) FindByID(_ context.Context, id int64) (product.Product, bool, error)` (body otherwise unchanged: still checks `findErr` first, then looks the id up in the map).
- `TestUseCaseExecuteReturnsExistingProduct`: build the product via `product.New("Keyboard", "Mechanical Keyboard", 12_999, "BRL")` (no id argument); save it via `repository.Save`, capture the returned, ID-assigned product as `want`; call `useCase.Execute` with `getproduct.Command{ID: want.ID}`; keep the `got != want` assertion.
- Rename `TestUseCaseExecuteReturnsErrEmptyIDForEmptyID` to `TestUseCaseExecuteReturnsErrInvalidIDForNonPositiveID`, convert it to a table test over two cases — `ID: 0` and `ID: -1` — each asserting `errors.Is(err, getproduct.ErrInvalidID)`.
- `TestUseCaseExecuteReturnsErrProductNotFoundForMissingProduct`: change `getproduct.Command{ID: "missing-product"}` to `getproduct.Command{ID: 999_999_999}` (a positive ID no test in this file ever assigns).
- `TestUseCaseExecuteReturnsRepositoryError`: change `getproduct.Command{ID: "product-123"}` to `getproduct.Command{ID: 1}`.

- [ ] **Step 2: Run the getproduct tests to verify they fail**

Run:
```bash
go test ./services/catalog/internal/application/getproduct/...
```
Expected: FAIL to compile — `fakeRepository` doesn't yet satisfy the extended port; `getproduct.ErrInvalidID` doesn't exist yet.

- [ ] **Step 3: Update the getproduct use case**

Edit `services/catalog/internal/application/getproduct/get_product.go`:
- Add `var ErrInvalidID = errors.New("product id must be positive")`.
- Change `Command.ID` from `string` to `int64`.
- In `Execute`, replace `if command.ID == "" { return product.Product{}, product.ErrEmptyID }` with `if command.ID <= 0 { return product.Product{}, ErrInvalidID }`.

- [ ] **Step 4: Run the getproduct tests to verify they pass**

Run:
```bash
go test -v ./services/catalog/internal/application/getproduct/...
```
Expected: PASS for all five tests (the two non-positive-ID cases plus the three unchanged-in-count tests).

- [ ] **Step 5: Confirm the application layer is fully green**

Run:
```bash
go build ./services/catalog/internal/application/...
go test ./services/catalog/internal/application/... ./services/catalog/internal/domain/... ./services/catalog/internal/config/...
```
Expected: all succeed. Do **not** run `go build ./...` yet: `internal/adapter/grpc` (Task 1) and `internal/adapter/postgres` (Task 3's port change) are still expected to be broken, resolved in Task 6 and Task 7.

- [ ] **Step 6: Commit**

```bash
git add services/catalog/internal/application/getproduct
git commit -m "feat: use int64 IDs and ErrInvalidID in GetProduct"
```

---

### Task 5: Add the migration converting products.id to an identity column

**Files:**
- Create: `services/catalog/migrations/000002_product_id_autoincrement.up.sql`
- Create: `services/catalog/migrations/000002_product_id_autoincrement.down.sql`

**Interfaces:**
- Produces: `products.id` as `BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY` after `up`; reverts to `TEXT PRIMARY KEY` (the `000001` shape) after `down`.

- [ ] **Step 1: Write the up migration**

Create `services/catalog/migrations/000002_product_id_autoincrement.up.sql` containing exactly two statements, in this order: `ALTER TABLE products DROP COLUMN id;` then `ALTER TABLE products ADD COLUMN id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY;`.

- [ ] **Step 2: Write the down migration**

Create `services/catalog/migrations/000002_product_id_autoincrement.down.sql` containing exactly two statements, in this order: `ALTER TABLE products DROP COLUMN id;` then `ALTER TABLE products ADD COLUMN id TEXT PRIMARY KEY;`.

- [ ] **Step 3: Apply and verify the migration locally**

This is destructive to any existing local rows' IDs — reset the local database rather than migrating it in place:
```bash
make db-reset
```
Expected: exits zero — this recreates the database and reapplies `000001` then `000002` from scratch.

Then confirm the resulting column shape, e.g.:
```bash
podman compose exec postgres psql -U catalog -d catalog -c '\d products'
```
(substitute `docker compose` if that's your local override) Expected: `id` is listed as `bigint`, `not null`, with `generated always as identity`, and remains the table's primary key.

- [ ] **Step 4: Commit**

```bash
git add services/catalog/migrations/000002_product_id_autoincrement.up.sql \
        services/catalog/migrations/000002_product_id_autoincrement.down.sql
git commit -m "feat: migrate products.id to a generated identity column"
```

---

### Task 6: Update the PostgreSQL adapter for server-assigned IDs

**Files:**
- Modify: `services/catalog/internal/adapter/postgres/product_repository.go`
- Modify: `services/catalog/internal/adapter/postgres/product_repository_integration_test.go`

**Interfaces:**
- Consumes: `port.ProductRepository` shape (Task 3), the `000002` migration (Task 5), `product.New` (Task 2).
- Produces: `(*postgres.Repository).Save(ctx, product.Product) (product.Product, error)`, `(*postgres.Repository).FindByID(ctx, id int64) (product.Product, bool, error)` — making `*postgres.Repository` satisfy `port.ProductRepository` again.

- [ ] **Step 1: Update the failing integration tests**

Edit `services/catalog/internal/adapter/postgres/product_repository_integration_test.go` (same file, same `//go:build integration` tag and `postgres_test` package):
- `TestProductRepositorySavePersistProduct`: build `want` via `product.New("Keyboard", "Mechanical keyboard", 12_999, "BRL")` (no id argument). Call `saved, err := repository.Save(ctx, want)`; assert `err == nil` and `saved.ID != 0`. Query the row by `saved.ID` (not `want.ID` — `want.ID` is `0`) and assert the scanned `got` equals `saved`.
- `TestProductRepositoryFindByIDReturnsPersistedProduct`: same construction/save change as above; call `repository.FindByID(ctx, saved.ID)`; assert `found` is `true` and the returned product equals `saved`.
- `TestProductRepositoryFindByIDReturnsNotFoundForMissingProduct`: change `repository.FindByID(ctx, "missing-product")` to `repository.FindByID(ctx, int64(999_999_999))`; keep the same found/error/zero-value assertions.
- Add a new test, `TestProductRepositorySaveAssignsDistinctSequentialIDs`: after truncating, build and save two products via `product.New(...)` + `repository.Save` back to back (any valid, distinct attributes); assert both saves return `err == nil`, both `ID`s are nonzero, the two IDs are not equal to each other, and the second ID is greater than the first. This directly proves the collision concern the project decided to fix is actually fixed.

- [ ] **Step 2: Run the integration tests to verify they fail**

Run:
```bash
make db-up
make migrate-up
CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' \
  go test -tags=integration -v ./services/catalog/internal/adapter/postgres
```
Expected: FAIL to compile — `Save`/`FindByID` on `*postgres.Repository` don't match the port yet, and `product.New` is still called with the old 5-argument shape in this file until this step's edits land; once the test file compiles, it fails at runtime because `Save` still returns only `error`.

- [ ] **Step 3: Update the PostgreSQL adapter**

Edit `services/catalog/internal/adapter/postgres/product_repository.go`:
- Change `Save(ctx context.Context, product product.Product) error` to `Save(ctx context.Context, product product.Product) (product.Product, error)`. The `INSERT` statement becomes `INSERT INTO products(name, description, price_minor_units, currency) VALUES($1, $2, $3, $4) RETURNING id`, executed with `r.pool.QueryRow(...)` (not `Exec`) so the returned `id` can be scanned into `product.ID` before returning `product, nil`; return `product.Product{}, err` on failure.
- Change `FindByID(ctx context.Context, id string) (product.Product, bool, error)` to take `id int64`; the query and scan logic are otherwise unchanged.

- [ ] **Step 4: Run the integration tests to verify they pass**

Run:
```bash
CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' \
  go test -tags=integration -v ./services/catalog/internal/adapter/postgres
```
Expected: PASS for all four tests (three updated plus the new distinct-IDs test).

- [ ] **Step 5: Run the full tagged suite for this package**

```bash
make test-integration
```
Expected: passes.

- [ ] **Step 6: Confirm scope of remaining breaks**

Run:
```bash
go build ./services/catalog/internal/adapter/postgres/...
```
Expected: succeeds — `*postgres.Repository` now satisfies `port.ProductRepository` again. Do **not** run `go build ./...` or `go build ./services/catalog/cmd/...` yet: `internal/adapter/grpc` is still broken from Task 1, and `cmd/catalog/main.go` imports that package directly, so the break still propagates to `cmd` even though `postgres` itself is fine. Task 7 resolves both.

- [ ] **Step 7: Commit**

```bash
git add services/catalog/internal/adapter/postgres
git commit -m "feat: assign product IDs via PostgreSQL identity column"
```

---

### Task 7: Update the gRPC handler for server-assigned int64 IDs

**Files:**
- Modify: `services/catalog/internal/adapter/grpc/catalog_server.go`
- Modify: `services/catalog/internal/adapter/grpc/catalog_server_test.go`

**Interfaces:**
- Consumes: `catalogv1.CreateProductRequest`/`GetProductRequest`/`Product` (Task 1), `getproduct.ErrInvalidID` (Task 4), `createproduct.Command`/`getproduct.Command` (Tasks 3–4).
- Produces: `(*grpcadapter.CatalogServer).CreateProduct`/`GetProduct` unchanged in method shape; `toProto` assigns `Id: p.ID` directly (`int64` to `int64`, no conversion).

- [ ] **Step 1: Update the failing gRPC handler tests**

Edit `services/catalog/internal/adapter/grpc/catalog_server_test.go`:
- Change `fakeRepository.Save` to `func (r *fakeRepository) Save(_ context.Context, p product.Product) (product.Product, error)`. On `saveErr != nil`, return `product.Product{}, r.saveErr`; otherwise assign `p.ID = int64(len(r.saved)) + 1`, append it to `r.saved`, and return it, `nil`.
- Change `fakeRepository.FindByID` to take `id int64` (body otherwise unchanged: still checks `findErr` first, then scans `r.saved` for a matching `ID`).
- `TestCatalogServerCreateProductReturnsPersistedProduct`: drop `Id: "product-123"` from the request; change the assertion from comparing `response.GetProduct().GetId()` to `repository.saved[0].ID` (equality still makes sense — the fake stores exactly what it returns) to also assert `response.GetProduct().GetId() != 0`.
- `TestCatalogServerCreateProductMapsDomainValidationToInvalidArgument`: no `Id` field to drop (it never set one); unchanged otherwise.
- `TestCatalogServerCreateProductMapsRepositoryFailureToInternal`: drop `Id: "product-123"` from the request.
- `TestCatalogServerGetProductReturnsPersistedProduct`: drop `Id: "product-123"` from the `CreateProduct` request; capture the created product's ID from `response.GetProduct().GetId()` after `CreateProduct`; call `GetProduct` with `&catalogv1.GetProductRequest{Id: createdID}`; change the ID assertion to `response.GetProduct().GetId() != createdID` (keep the existing name/description/price/currency assertions).
- `TestCatalogServerGetProductMapsEmptyIDToInvalidArgument`: rename to `TestCatalogServerGetProductMapsNonPositiveIDToInvalidArgument`; change `&catalogv1.GetProductRequest{Id: ""}` to `&catalogv1.GetProductRequest{Id: 0}`.
- `TestCatalogServerGetProductMapsMissingProductToNotFound`: change `&catalogv1.GetProductRequest{Id: "missing-product"}` to `&catalogv1.GetProductRequest{Id: 999_999_999}`.
- `TestCatalogServerGetProductMapsRepositoryFailureToInternal`: change `&catalogv1.GetProductRequest{Id: "product-123"}` to `&catalogv1.GetProductRequest{Id: 1}`.

- [ ] **Step 2: Run the gRPC handler tests to verify they fail**

Run:
```bash
go test ./services/catalog/internal/adapter/grpc/...
```
Expected: FAIL to compile — the package still has the Task 1 break (`request.GetId()` doesn't exist on `CreateProductRequest`; `p.ID` doesn't fit `catalogv1.Product.Id`), plus the fake repository no longer matches the port.

- [ ] **Step 3: Update the gRPC handler**

Edit `services/catalog/internal/adapter/grpc/catalog_server.go`:
- In `CreateProduct`, remove the `ID: request.GetId(),` line from the `createproduct.Command{...}` literal (the field no longer exists on `Command` as of Task 3, and the request no longer carries an `id`).
- In `isProductValidationError`, remove the `errors.Is(err, product.ErrEmptyID) ||` clause (the error no longer exists as of Task 2); keep the `ErrEmptyName`/`ErrNegativePrice`/`ErrInvalidCurrency` checks.
- In `GetProduct`, change `getproduct.Command{ID: request.GetId()}` — this already works once `request.GetId()` returns `int64` (Task 1) and `Command.ID` is `int64` (Task 4), no further change needed there. Change the error mapping from `errors.Is(err, product.ErrEmptyID)` to `errors.Is(err, getproduct.ErrInvalidID)`.
- In `toProto`, no code change is needed: `Id: p.ID` now assigns `int64` to `int64` directly.

- [ ] **Step 4: Run the gRPC handler tests to verify they pass**

Run:
```bash
go test -v ./services/catalog/internal/adapter/grpc/...
```
Expected: PASS for all seven tests.

- [ ] **Step 5: Confirm the whole module builds and the full database-free suite passes**

Run:
```bash
go build ./...
make check
```
Expected: both succeed — every package-level break introduced since Task 1 is now resolved.

- [ ] **Step 6: Commit**

```bash
git add services/catalog/internal/adapter/grpc/catalog_server.go \
        services/catalog/internal/adapter/grpc/catalog_server_test.go
git commit -m "feat: handle server-assigned int64 product IDs over gRPC"
```

---

### Task 8: Prove the complete create/get flow against PostgreSQL with server-assigned IDs

**Files:**
- Modify: `services/catalog/internal/adapter/grpc/catalog_server_integration_test.go`

**Interfaces:**
- Consumes: `postgres.NewProductRepository` (Task 6), `grpcadapter.NewCatalogServer` (Task 7, signature unchanged).

- [ ] **Step 1: Update the create-and-persist test**

Edit `TestCatalogServerCreateProductPersistsThroughPostgreSQL` in `services/catalog/internal/adapter/grpc/catalog_server_integration_test.go`:
- Drop `Id: "product-123"` from the first `CreateProduct` request.
- Capture the response's assigned ID: `createdID := response.GetProduct().GetId()`; assert `createdID != 0` (replacing the old `!= "product-123"` check).
- Change the `SELECT name FROM products where id = $1` query parameter from `"product-123"` to `createdID`.
- The second `CreateProduct` call (the invalid-product case expecting `codes.InvalidArgument`) needs no ID-related change — it never set one.

- [ ] **Step 2: Update the get-after-create test**

Edit `TestCatalogServerGetProductPersistsThroughPostgreSQL` in the same file:
- Drop `Id: "product-123"` from the `CreateProduct` request; capture the created product's ID from its response (`createResponse.GetProduct().GetId()`).
- Change the `GetProduct` call to use `&catalogv1.GetProductRequest{Id: createdID}`.
- Change the missing-product `GetProduct` call from `&catalogv1.GetProductRequest{Id: "missing-product"}` to `&catalogv1.GetProductRequest{Id: 999_999_999}`, keeping the `codes.NotFound` assertion.

- [ ] **Step 3: Run the tests**

Run:
```bash
make db-reset
make migrate-up
CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' \
  go test -tags=integration -v ./services/catalog/internal/adapter/grpc
```
Expected: both `TestCatalogServerCreateProductPersistsThroughPostgreSQL` and `TestCatalogServerGetProductPersistsThroughPostgreSQL` pass.

- [ ] **Step 4: Run the full tagged gRPC integration suite**

```bash
make test-grpc-integration
```
Expected: both tests pass.

- [ ] **Step 5: Run the full database-free quality contract once more**

```bash
make check
```
Expected: passes.

- [ ] **Step 6: Commit**

```bash
git add services/catalog/internal/adapter/grpc/catalog_server_integration_test.go
git commit -m "test: cover Catalog create/get flow with server-assigned IDs"
```

---

### Task 9: Final verification and issue closeout

**Files:** none (verification only, plus pushing the branch).

- [ ] **Step 1: Confirm README needs no change**

Read `README.md`'s "Current State" section. It currently reads: "Catalog exposes versioned gRPC product creation and retrieval by ID, wired through its `CreateProduct`/`GetProduct` use cases and PostgreSQL repository adapter." This remains accurate (it never described the ID as client-supplied) — no edit needed. If it has drifted from this text by the time this task runs, update it to match reality before proceeding.

- [ ] **Step 2: Run the full local quality contract one last time**

```bash
make check
```
Expected: passes (fmt, vet, all unit tests).

- [ ] **Step 3: Run both tagged integration suites one last time**

```bash
make db-reset
make test-integration
make test-grpc-integration
```
Expected: both pass.

- [ ] **Step 4: Verify against Issue #13's acceptance criteria**

Confirm each holds (all exercised by tests above):
- Creating a product does not require or accept a client-supplied ID; PostgreSQL assigns a unique, sequential `int64` ID on insert (`TestProductRepositorySaveAssignsDistinctSequentialIDs`, `TestCatalogServerCreateProductReturnsPersistedProduct`).
- Sequential/concurrent creates never collide on ID (`TestProductRepositorySaveAssignsDistinctSequentialIDs`).
- `GetProduct` looks up by `int64` ID; a non-positive ID returns `codes.InvalidArgument` (`TestCatalogServerGetProductMapsNonPositiveIDToInvalidArgument`).
- `make check` remains database-free; tagged integration targets cover PostgreSQL and the end-to-end gRPC path against the new schema (Steps 2–3 above).

- [ ] **Step 5: Push the branch**

```bash
git push -u origin issue_13
```
