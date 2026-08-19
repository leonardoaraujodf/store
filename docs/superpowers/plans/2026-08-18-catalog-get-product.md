# Catalog GetProduct Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `GetProduct` read path to Catalog's versioned gRPC API, backed by the same hexagonal architecture and PostgreSQL ownership model as `CreateProduct`.

**Architecture:** The protobuf contract gains an additive `GetProduct` RPC reusing the existing `Product` message. A new `getproduct` application use case owns read-path semantics (empty ID is invalid input; a missing row is a domain-level not-found result, not an error). The repository port gains a `FindByID` lookup that returns `(product.Product, bool, error)` so "not found" and "database failure" are never conflated; the PostgreSQL adapter translates `pgx.ErrNoRows` into `found=false` and lets every other error propagate. The gRPC handler maps use-case outcomes to `InvalidArgument`, `NotFound`, and `Internal`.

**Tech Stack:** Go 1.26.5, gRPC-Go, Protocol Buffers (Buf v2), pgx/pgxpool, PostgreSQL via Compose, `bufconn` for in-process gRPC tests.

**Spec:** `docs/superpowers/specs/2026-08-18-catalog-get-product-design.md`

## Global Constraints

- Domain/application packages must not import protobuf, gRPC, or PostgreSQL adapter packages.
- No product listing/pagination, categories, REST, Kafka, or new runtime infrastructure.
- Repository lookup must distinguish three outcomes: found; absent (not an error); database failure.
- `pgx.ErrNoRows` must be translated inside the PostgreSQL adapter; it must never escape into application or gRPC code.
- Empty ID → `codes.InvalidArgument`; missing product → `codes.NotFound`; repository/database failure → `codes.Internal` with a generic message that discloses no database implementation detail.
- `GetProduct` is additive to `catalog.v1`: no existing message or field number changes.
- `make check` remains database-free. Existing protobuf workflow (`make proto-format`, `make proto-lint`, `make proto-generate`, `make proto-check`) runs after any contract change. Existing Compose-backed `make test-integration` / `make test-grpc-integration` targets already target their whole package directories, so new tests in those packages are picked up automatically — no Makefile changes needed.
- Apply red, green, refactor to every behavior change.

---

## File structure

| Path | Responsibility |
| --- | --- |
| `proto/catalog/v1/catalog.proto` | Adds `GetProduct` RPC, `GetProductRequest`, `GetProductResponse`. |
| `gen/go/catalog/v1/*.pb.go` | Regenerated, committed Go types and service registration. |
| `services/catalog/internal/application/port/product_repository.go` | Repository port gains `FindByID`. |
| `services/catalog/internal/application/getproduct` | New package: `GetProduct` use case and its unit tests. |
| `services/catalog/internal/application/createproduct` | Existing test double updated to keep satisfying the extended port. |
| `services/catalog/internal/adapter/postgres` | `FindByID` lookup and its tagged integration tests. |
| `services/catalog/internal/adapter/grpc` | `GetProduct` gRPC handler, unit tests, and tagged gRPC-to-PostgreSQL integration test. |
| `services/catalog/cmd/catalog` | Wires the `GetProduct` use case into the runnable process. |
| `README.md` | Reflects that Catalog now supports product lookup by ID. |

---

### Task 1: Extend the Catalog protobuf contract with GetProduct

**Files:**
- Modify: `proto/catalog/v1/catalog.proto`
- Modify (generated, do not hand-edit): `gen/go/catalog/v1/catalog.pb.go`, `gen/go/catalog/v1/catalog_grpc.pb.go`

**Interfaces:**
- Produces `catalogv1.GetProductRequest{Id string}`, `catalogv1.GetProductResponse{Product *catalogv1.Product}`.
- Produces `catalogv1.CatalogServiceServer.GetProduct(context.Context, *GetProductRequest) (*GetProductResponse, error)` (and matching client method) for later tasks to implement/consume.

- [ ] **Step 1: Add the RPC and messages to the proto contract**

Edit `proto/catalog/v1/catalog.proto`:

```proto
syntax = "proto3";

package catalog.v1;

option go_package = "github.com/leonardoaraujodf/store/gen/go/catalog/v1;catalogv1";

service CatalogService {
  rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse);
  rpc GetProduct(GetProductRequest) returns (GetProductResponse);
}

message CreateProductRequest {
  string id = 1;
  string name = 2;
  string description = 3;
  int64 price_minor_units = 4;
  string currency = 5;
}

message CreateProductResponse {
  Product product = 1;
}

message GetProductRequest {
  string id = 1;
}

message GetProductResponse {
  Product product = 1;
}

message Product {
  string id = 1;
  string name = 2;
  string description = 3;
  int64 price_minor_units = 4;
  string currency = 5;
}
```

Field numbers on existing messages are unchanged. `GetProductRequest`/`GetProductResponse` are new messages; do not reuse field numbers from other messages.

- [ ] **Step 2: Format, lint, and generate**

```bash
make proto-format
make proto-lint
make proto-generate
```

Expected: all three commands exit zero; `gen/go/catalog/v1/catalog.pb.go` and `gen/go/catalog/v1/catalog_grpc.pb.go` are updated with `GetProductRequest`, `GetProductResponse`, `GetProduct` client/server methods, and `UnimplementedCatalogServiceServer.GetProduct`.

- [ ] **Step 3: Verify no drift**

```bash
make proto-check
```

Expected: exits zero (format, lint, generate, and `git diff --exit-code -- gen/go` all pass — generation is reproducible and already committed-shape).

- [ ] **Step 4: Build the module**

```bash
go build ./...
```

Expected: succeeds. `services/catalog/internal/adapter/grpc.CatalogServer` does not yet implement `GetProduct`, but it embeds `catalogv1.UnimplementedCatalogServiceServer`, so the interface is still satisfied.

- [ ] **Step 5: Commit**

```bash
git add proto/catalog/v1/catalog.proto gen/go
git commit -m "feat: add Catalog GetProduct gRPC contract"
```

---

### Task 2: Extend the repository port and implement the GetProduct use case

**Files:**
- Modify: `services/catalog/internal/application/port/product_repository.go`
- Modify: `services/catalog/internal/application/createproduct/create_product_test.go`
- Create: `services/catalog/internal/application/getproduct/get_product.go`
- Create: `services/catalog/internal/application/getproduct/get_product_test.go`

**Interfaces:**
- Consumes: `product.Product` (`services/catalog/internal/domain/product`), `product.ErrEmptyID`.
- Produces: `port.ProductRepository` interface now requires `FindByID(ctx context.Context, id string) (product.Product, bool, error)` in addition to the existing `Save`.
- Produces: `getproduct.Command{ID string}`, `getproduct.UseCase`, `getproduct.New(repository port.ProductRepository) UseCase`, `getproduct.UseCase.Execute(ctx context.Context, command Command) (product.Product, error)`, `getproduct.ErrProductNotFound`.

- [ ] **Step 1: Write the failing use-case tests**

Create `services/catalog/internal/application/getproduct/get_product_test.go`:

```go
package getproduct_test

import (
	"context"
	"errors"
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type fakeRepository struct {
	products map[string]product.Product
	findErr  error
}

func (f *fakeRepository) Save(_ context.Context, p product.Product) error {
	if f.products == nil {
		f.products = map[string]product.Product{}
	}
	f.products[p.ID] = p
	return nil
}

func (f *fakeRepository) FindByID(_ context.Context, id string) (product.Product, bool, error) {
	if f.findErr != nil {
		return product.Product{}, false, f.findErr
	}
	p, found := f.products[id]
	return p, found, nil
}

func TestUseCaseExecuteReturnsExistingProduct(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	want, err := product.New("product-123", "Keyboard", "Mechanical Keyboard", 12_999, "BRL")
	if err != nil {
		t.Fatalf("product.New() error = %v", err)
	}
	if err := repository.Save(context.Background(), want); err != nil {
		t.Fatalf("repository.Save() error = %v", err)
	}

	useCase := getproduct.New(repository)
	got, err := useCase.Execute(context.Background(), getproduct.Command{ID: "product-123"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != want {
		t.Fatalf("result = %#v, want %#v", got, want)
	}
}

func TestUseCaseExecuteReturnsErrEmptyIDForEmptyID(t *testing.T) {
	t.Parallel()

	useCase := getproduct.New(&fakeRepository{})
	_, err := useCase.Execute(context.Background(), getproduct.Command{ID: ""})
	if !errors.Is(err, product.ErrEmptyID) {
		t.Fatalf("Execute() error = %v, want %v", err, product.ErrEmptyID)
	}
}

func TestUseCaseExecuteReturnsErrProductNotFoundForMissingProduct(t *testing.T) {
	t.Parallel()

	useCase := getproduct.New(&fakeRepository{})
	_, err := useCase.Execute(context.Background(), getproduct.Command{ID: "missing-product"})
	if !errors.Is(err, getproduct.ErrProductNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, getproduct.ErrProductNotFound)
	}
}

func TestUseCaseExecuteReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("repository unavailable")
	useCase := getproduct.New(&fakeRepository{findErr: wantErr})
	_, err := useCase.Execute(context.Background(), getproduct.Command{ID: "product-123"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

```bash
go test ./services/catalog/internal/application/getproduct/...
```

Expected: FAIL — package `getproduct` does not exist yet.

- [ ] **Step 3: Extend the repository port**

Edit `services/catalog/internal/application/port/product_repository.go`:

```go
package port

import (
	"context"

	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type ProductRepository interface {
	Save(context.Context, product.Product) error
	FindByID(ctx context.Context, id string) (product.Product, bool, error)
}
```

- [ ] **Step 4: Update the createproduct test double to satisfy the extended port**

The `port.ProductRepository` interface now requires `FindByID`. `services/catalog/internal/application/createproduct/create_product_test.go`'s `fakeRepository` is passed into `createproduct.New`, whose parameter is typed `port.ProductRepository`, so it must implement the new method too — even though no `createproduct` test exercises it. Add a minimal, always-empty stub so the package keeps compiling and its existing behavior is unchanged:

```go
func (f *fakeRepository) FindByID(_ context.Context, id string) (product.Product, bool, error) {
	return product.Product{}, false, nil
}
```

Add this method next to `fakeRepository`'s existing `Save` method in that file.

- [ ] **Step 5: Implement the GetProduct use case**

Create `services/catalog/internal/application/getproduct/get_product.go`:

```go
package getproduct

import (
	"context"
	"errors"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/port"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

var ErrProductNotFound = errors.New("product not found")

type Command struct {
	ID string
}

type UseCase struct {
	repository port.ProductRepository
}

func New(repository port.ProductRepository) UseCase {
	return UseCase{
		repository: repository,
	}
}

func (u UseCase) Execute(ctx context.Context, command Command) (product.Product, error) {
	if command.ID == "" {
		return product.Product{}, product.ErrEmptyID
	}

	p, found, err := u.repository.FindByID(ctx, command.ID)
	if err != nil {
		return product.Product{}, err
	}
	if !found {
		return product.Product{}, ErrProductNotFound
	}

	return p, nil
}
```

- [ ] **Step 6: Run the new tests to verify they pass**

```bash
go test -v ./services/catalog/internal/application/getproduct/...
```

Expected: PASS for all four tests.

- [ ] **Step 7: Confirm the affected packages still compile and pass**

```bash
go build ./services/catalog/internal/...
go test ./services/catalog/internal/application/createproduct/...
```

Expected: both succeed — `createproduct`'s existing three tests still pass unchanged. Do **not** run `go build ./...` yet: `services/catalog/cmd/catalog/main.go` still passes `*postgres.Repository` into `createproduct.New`, and that concrete type won't satisfy the newly-extended `port.ProductRepository` (missing `FindByID`) until Task 3 implements it there. That whole-module build gap is expected and closes in Task 3; it is not a regression to fix here.

- [ ] **Step 8: Commit**

```bash
git add services/catalog/internal/application/port/product_repository.go \
        services/catalog/internal/application/createproduct/create_product_test.go \
        services/catalog/internal/application/getproduct
git commit -m "feat: implement GetProduct use case"
```

---

### Task 3: Implement and integration-test the PostgreSQL FindByID lookup

**Files:**
- Modify: `services/catalog/internal/adapter/postgres/product_repository.go`
- Modify: `services/catalog/internal/adapter/postgres/product_repository_integration_test.go`

**Interfaces:**
- Consumes: `port.ProductRepository.FindByID` signature from Task 2; `product.Product`.
- Produces: `(*postgres.Repository).FindByID(ctx context.Context, id string) (product.Product, bool, error)`, making `*postgres.Repository` satisfy the extended `port.ProductRepository`.

- [ ] **Step 1: Write the failing integration tests**

Add to `services/catalog/internal/adapter/postgres/product_repository_integration_test.go` (same file, same `//go:build integration` tag and `postgres_test` package as the existing `TestProductRepositorySavePersistProduct`):

```go
func TestProductRepositoryFindByIDReturnsPersistedProduct(t *testing.T) {
	ctx := context.Background()

	databaseURL := os.Getenv("CATALOG_DATABASE_URL")
	if databaseURL == "" {
		t.Fatalf("CATALOG_DATABASE_URL must be set")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE products"); err != nil {
		t.Fatalf("TRUNCATE products error = %v", err)
	}
	want, err := product.New(
		"product-123",
		"Keyboard",
		"Mechanical keyboard",
		12_999,
		"BRL",
	)
	if err != nil {
		t.Fatalf("product.New() error = %v", err)
	}
	repository := postgres.NewProductRepository(pool)
	if err := repository.Save(ctx, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, found, err := repository.FindByID(ctx, "product-123")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if !found {
		t.Fatalf("FindByID() found = false, want true")
	}
	if got != want {
		t.Errorf("FindByID() = %#v, want %#v", got, want)
	}
}

func TestProductRepositoryFindByIDReturnsNotFoundForMissingProduct(t *testing.T) {
	ctx := context.Background()

	databaseURL := os.Getenv("CATALOG_DATABASE_URL")
	if databaseURL == "" {
		t.Fatalf("CATALOG_DATABASE_URL must be set")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE products"); err != nil {
		t.Fatalf("TRUNCATE products error = %v", err)
	}

	repository := postgres.NewProductRepository(pool)
	got, found, err := repository.FindByID(ctx, "missing-product")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found {
		t.Fatalf("FindByID() found = true, want false")
	}
	if got != (product.Product{}) {
		t.Errorf("FindByID() = %#v, want zero value", got)
	}
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

```bash
make db-up
make migrate-up
CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' \
  go test -tags=integration -run TestProductRepositoryFindByID -v ./services/catalog/internal/adapter/postgres
```

Expected: FAIL — `repository.FindByID undefined (type *postgres.Repository has no field or method FindByID)`.

- [ ] **Step 3: Implement FindByID**

Edit `services/catalog/internal/adapter/postgres/product_repository.go`:

```go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (r *Repository) Save(ctx context.Context, product product.Product) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO products(id, name, description, price_minor_units, currency)
		 VALUES($1, $2, $3, $4, $5)`,
		product.ID,
		product.Name,
		product.Description,
		product.PriceMinorUnits,
		product.Currency,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (product.Product, bool, error) {
	var p product.Product
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, name, description, price_minor_units, currency
		 FROM products
		 WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.PriceMinorUnits, &p.Currency)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return product.Product{}, false, nil
		}
		return product.Product{}, false, err
	}
	return p, true, nil
}
```

`pgx.ErrNoRows` is translated to `found=false, err=nil` here so it never escapes into application or gRPC code, matching the design spec.

- [ ] **Step 4: Run the new tests to verify they pass**

```bash
CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' \
  go test -tags=integration -run TestProductRepositoryFindByID -v ./services/catalog/internal/adapter/postgres
```

Expected: PASS for both new tests.

- [ ] **Step 5: Run the full tagged suite for this package**

```bash
make test-integration
```

Expected: all tests in `services/catalog/internal/adapter/postgres` pass, including the pre-existing `TestProductRepositorySavePersistProduct`.

- [ ] **Step 6: Confirm the database-free build still compiles**

```bash
go build ./...
go vet ./services/catalog/internal/adapter/postgres/... ./services/catalog/internal/application/...
go test ./services/catalog/internal/application/... ./services/catalog/internal/domain/... ./services/catalog/internal/config/...
```

Expected: all succeed. Do **not** run `make check` (or bare `go vet ./...`/`go test ./...`) yet: `services/catalog/internal/adapter/grpc/catalog_server_test.go`'s `fakeRepository` test double still only implements `Save`, so it fails to satisfy the newly-extended `port.ProductRepository` (missing `FindByID`) when the `grpc` package's tests are type-checked. That gap is expected — Task 4 replaces `catalog_server_test.go` wholesale, including that test double — and is not a regression to fix here. `go build ./...` succeeds regardless because plain `build` never type-checks `_test.go` files.

- [ ] **Step 7: Commit**

```bash
git add services/catalog/internal/adapter/postgres
git commit -m "feat: implement PostgreSQL FindByID adapter"
```

---

### Task 4: Implement and unit-test the GetProduct gRPC handler

**Files:**
- Modify: `services/catalog/internal/adapter/grpc/catalog_server.go`
- Modify: `services/catalog/internal/adapter/grpc/catalog_server_test.go`

**Interfaces:**
- Consumes: `catalogv1.GetProductRequest`/`GetProductResponse` (Task 1), `getproduct.New`/`Command`/`UseCase`/`ErrProductNotFound` (Task 2), `product.ErrEmptyID`.
- Produces: `grpcadapter.NewCatalogServer(createUseCase createproduct.UseCase, getUseCase getproduct.UseCase) *CatalogServer` (signature change — both existing call sites in this file and `cmd/catalog/main.go` must be updated). Produces `(*CatalogServer).GetProduct(context.Context, *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error)`.

- [ ] **Step 1: Update the test double and helper, and write the failing GetProduct tests**

Replace the full contents of `services/catalog/internal/adapter/grpc/catalog_server_test.go`:

```go
package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	grpcadapter "github.com/leonardoaraujodf/store/services/catalog/internal/adapter/grpc"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fakeRepository struct {
	saved   []product.Product
	saveErr error
	findErr error
}

func (r *fakeRepository) Save(_ context.Context, p product.Product) error {
	if r.saveErr != nil {
		return r.saveErr
	}

	r.saved = append(r.saved, p)
	return nil
}

func (r *fakeRepository) FindByID(_ context.Context, id string) (product.Product, bool, error) {
	if r.findErr != nil {
		return product.Product{}, false, r.findErr
	}
	for _, p := range r.saved {
		if p.ID == id {
			return p, true, nil
		}
	}
	return product.Product{}, false, nil
}

func newCatalogClient(t *testing.T, createUseCase createproduct.UseCase, getUseCase getproduct.UseCase) catalogv1.CatalogServiceClient {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(server, grpcadapter.NewCatalogServer(createUseCase, getUseCase))
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	connection, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return catalogv1.NewCatalogServiceClient(connection)
}

func TestCatalogServerCreateProductReturnsPersistedProduct(t *testing.T) {
	repository := &fakeRepository{}
	client := newCatalogClient(t, createproduct.New(repository), getproduct.New(repository))

	response, err := client.CreateProduct(context.Background(),
		&catalogv1.CreateProductRequest{
			Id:              "product-123",
			Name:            "Keyboard",
			Description:     "Mechanical Keyboard",
			PriceMinorUnits: 12_999,
			Currency:        "BRL",
		})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	if len(repository.saved) != 1 {
		t.Fatalf("saved = %#v, want one product", repository.saved)
	}
	if response.GetProduct().GetId() != repository.saved[0].ID {
		t.Errorf("product ID = %q, want %q", response.GetProduct().GetId(), repository.saved[0].ID)
	}
}

func TestCatalogServerCreateProductMapsDomainValidationToInvalidArgument(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{}), getproduct.New(&fakeRepository{}))
	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Name:            "Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestCatalogServerCreateProductMapsRepositoryFailureToInternal(t *testing.T) {
	repository := &fakeRepository{saveErr: errors.New("database unavailable")}
	client := newCatalogClient(t, createproduct.New(repository), getproduct.New(repository))
	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Id:              "product-123",
		Name:            "Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.Internal, err)
	}
}

func TestCatalogServerGetProductReturnsPersistedProduct(t *testing.T) {
	repository := &fakeRepository{}
	client := newCatalogClient(t, createproduct.New(repository), getproduct.New(repository))

	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Id:              "product-123",
		Name:            "Keyboard",
		Description:     "Mechanical Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	response, err := client.GetProduct(context.Background(), &catalogv1.GetProductRequest{Id: "product-123"})
	if err != nil {
		t.Fatalf("GetProduct() error = %v", err)
	}
	if response.GetProduct().GetId() != "product-123" {
		t.Errorf("product ID = %q, want %q", response.GetProduct().GetId(), "product-123")
	}
	if response.GetProduct().GetName() != "Keyboard" {
		t.Errorf("product name = %q, want %q", response.GetProduct().GetName(), "Keyboard")
	}
}

func TestCatalogServerGetProductMapsEmptyIDToInvalidArgument(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{}), getproduct.New(&fakeRepository{}))
	_, err := client.GetProduct(context.Background(), &catalogv1.GetProductRequest{Id: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestCatalogServerGetProductMapsMissingProductToNotFound(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{}), getproduct.New(&fakeRepository{}))
	_, err := client.GetProduct(context.Background(), &catalogv1.GetProductRequest{Id: "missing-product"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.NotFound, err)
	}
}

func TestCatalogServerGetProductMapsRepositoryFailureToInternal(t *testing.T) {
	repository := &fakeRepository{findErr: errors.New("database unavailable")}
	client := newCatalogClient(t, createproduct.New(repository), getproduct.New(repository))
	_, err := client.GetProduct(context.Background(), &catalogv1.GetProductRequest{Id: "product-123"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.Internal, err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./services/catalog/internal/adapter/grpc/...
```

Expected: FAIL — `grpcadapter.NewCatalogServer` still takes one argument, and `client.GetProduct` is undefined.

- [ ] **Step 3: Implement the GetProduct handler and update the constructor**

Replace the full contents of `services/catalog/internal/adapter/grpc/catalog_server.go`:

```go
package grpc

import (
	"context"
	"errors"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CatalogServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	createUseCase createproduct.UseCase
	getUseCase    getproduct.UseCase
}

func NewCatalogServer(createUseCase createproduct.UseCase, getUseCase getproduct.UseCase) *CatalogServer {
	return &CatalogServer{createUseCase: createUseCase, getUseCase: getUseCase}
}

func (s *CatalogServer) CreateProduct(ctx context.Context, request *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	p, err := s.createUseCase.Execute(ctx, createproduct.Command{
		ID:              request.GetId(),
		Name:            request.GetName(),
		Description:     request.GetDescription(),
		PriceMinorUnits: request.GetPriceMinorUnits(),
		Currency:        request.GetCurrency(),
	})
	if err != nil {
		if isProductValidationError(err) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "could not create product")
	}
	return &catalogv1.CreateProductResponse{
		Product: &catalogv1.Product{
			Id:              p.ID,
			Name:            p.Name,
			Description:     p.Description,
			PriceMinorUnits: p.PriceMinorUnits,
			Currency:        p.Currency,
		},
	}, nil
}

func (s *CatalogServer) GetProduct(ctx context.Context, request *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	p, err := s.getUseCase.Execute(ctx, getproduct.Command{ID: request.GetId()})
	if err != nil {
		if errors.Is(err, product.ErrEmptyID) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, getproduct.ErrProductNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "could not get product")
	}
	return &catalogv1.GetProductResponse{
		Product: &catalogv1.Product{
			Id:              p.ID,
			Name:            p.Name,
			Description:     p.Description,
			PriceMinorUnits: p.PriceMinorUnits,
			Currency:        p.Currency,
		},
	}, nil
}

func isProductValidationError(err error) bool {
	return errors.Is(err, product.ErrEmptyID) ||
		errors.Is(err, product.ErrEmptyName) ||
		errors.Is(err, product.ErrNegativePrice) ||
		errors.Is(err, product.ErrInvalidCurrency)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test -v ./services/catalog/internal/adapter/grpc/...
```

Expected: PASS for all six tests (three existing `CreateProduct` tests plus three new `GetProduct` tests).

- [ ] **Step 5: Note the expected transient build break**

```bash
go build ./...
```

Expected: FAILS — `services/catalog/cmd/catalog/main.go` still calls the old one-argument `NewCatalogServer`. This is expected; do not attempt to fix `main.go` here. Task 5 fixes it next. Do not run `make check` until Task 5 is complete, since it also builds `cmd/catalog`.

- [ ] **Step 6: Commit**

```bash
git add services/catalog/internal/adapter/grpc/catalog_server.go \
        services/catalog/internal/adapter/grpc/catalog_server_test.go
git commit -m "feat: implement GetProduct gRPC handler"
```

---

### Task 5: Wire GetProduct into the runnable Catalog process

**Files:**
- Modify: `services/catalog/cmd/catalog/main.go`

**Interfaces:**
- Consumes: `getproduct.New(repository port.ProductRepository) getproduct.UseCase` (Task 2), `grpcadapter.NewCatalogServer(createUseCase, getUseCase)` (Task 4).

- [ ] **Step 1: Update the composition root**

Edit `services/catalog/cmd/catalog/main.go`: add the `getproduct` import and pass a second use case into `NewCatalogServer`.

```go
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	grpcadapter "github.com/leonardoaraujodf/store/services/catalog/internal/adapter/grpc"
	"github.com/leonardoaraujodf/store/services/catalog/internal/adapter/postgres"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/config"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "catalog service error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", cfg.GRPCAddr, err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	repository := postgres.NewProductRepository(pool)
	catalogv1.RegisterCatalogServiceServer(server,
		grpcadapter.NewCatalogServer(createproduct.New(repository), getproduct.New(repository)))
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()
	fmt.Printf("Catalog gRPC service listening on %s\n", cfg.GRPCAddr)

	signalContext, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-signalContext.Done():
		server.GracefulStop()
		return nil
	case err := <-serveErrors:
		return fmt.Errorf("serve GRPC: %w", err)
	}

}
```

- [ ] **Step 2: Confirm the module builds**

```bash
go build ./...
```

Expected: succeeds — `postgres.Repository` satisfies the extended `port.ProductRepository` (Task 3), so `createproduct.New(repository)` and `getproduct.New(repository)` both compile with the same `repository` value.

- [ ] **Step 3: Run the full database-free quality contract**

```bash
make check
```

Expected: passes — `fmt-check`, `vet`, and every unit test across the module, including the `getproduct` and `grpc` package tests from Tasks 2 and 4.

- [ ] **Step 4: Commit**

```bash
git add services/catalog/cmd/catalog/main.go
git commit -m "feat: wire GetProduct into Catalog runtime"
```

---

### Task 6: Prove the complete GetProduct gRPC-to-PostgreSQL path

**Files:**
- Modify: `services/catalog/internal/adapter/grpc/catalog_server_integration_test.go`

**Interfaces:**
- Consumes: `postgres.NewProductRepository` (Task 3), `getproduct.New` (Task 2), `grpcadapter.NewCatalogServer(createUseCase, getUseCase)` (Task 4).

- [ ] **Step 1: Update the existing test's wiring and add the failing GetProduct test**

Edit `services/catalog/internal/adapter/grpc/catalog_server_integration_test.go`. First, update the existing `RegisterCatalogServiceServer` call in `TestCatalogServerCreateProductPersistsThroughPostgreSQL` to pass both use cases:

```go
	repository := postgres.NewProductRepository(pool)
	catalogv1.RegisterCatalogServiceServer(server,
		grpcadapter.NewCatalogServer(createproduct.New(repository), getproduct.New(repository)))
```

Add the `getproduct` import:

```go
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
```

Then append a new test function to the same file:

```go
func TestCatalogServerGetProductPersistsThroughPostgreSQL(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("CATALOG_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("CATALOG_DATABASE_URL must be set")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "TRUNCATE products"); err != nil {
		t.Fatalf("TRUNCATE products error = %v", err)
	}

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	repository := postgres.NewProductRepository(pool)
	catalogv1.RegisterCatalogServiceServer(server,
		grpcadapter.NewCatalogServer(createproduct.New(repository), getproduct.New(repository)))
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)
	connection, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	client := catalogv1.NewCatalogServiceClient(connection)

	_, err = client.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Id:              "product-123",
		Name:            "Keyboard",
		Description:     "Mechanical keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	response, err := client.GetProduct(ctx, &catalogv1.GetProductRequest{Id: "product-123"})
	if err != nil {
		t.Fatalf("GetProduct() error = %v", err)
	}
	if response.GetProduct().GetName() != "Keyboard" {
		t.Errorf("product name = %q, want %q", response.GetProduct().GetName(), "Keyboard")
	}

	_, err = client.GetProduct(ctx, &catalogv1.GetProductRequest{Id: "missing-product"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.NotFound, err)
	}
}
```

- [ ] **Step 2: Run the tests**

This task adds proof-level end-to-end coverage over behavior Tasks 2–5 already implemented and unit-tested, mirroring how this repository already separates unit-level TDD from a dedicated tagged integration-test commit (see `3486a34 test: cover Catalog gRPC PostgreSQL flow` and `2e68ef8 test: tag Catalog gRPC integration test` in `git log`). There is no red phase to chase here beyond the test not existing yet — just confirm it passes once written:

```bash
make db-up
make migrate-up
CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' \
  go test -tags=integration -v ./services/catalog/internal/adapter/grpc
```

Expected: both `TestCatalogServerCreateProductPersistsThroughPostgreSQL` and `TestCatalogServerGetProductPersistsThroughPostgreSQL` pass.

- [ ] **Step 3: Run the full tagged gRPC integration suite**

```bash
make test-grpc-integration
```

Expected: both `TestCatalogServerCreateProductPersistsThroughPostgreSQL` and `TestCatalogServerGetProductPersistsThroughPostgreSQL` pass.

- [ ] **Step 4: Run the full database-free quality contract once more**

```bash
make check
```

Expected: passes.

- [ ] **Step 5: Commit**

```bash
git add services/catalog/internal/adapter/grpc/catalog_server_integration_test.go
git commit -m "test: cover Catalog GetProduct gRPC-to-PostgreSQL flow"
```

---

### Task 7: Document GetProduct and finalize issue verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the Current State section**

In `README.md`, replace:

```markdown
The project is in **Milestone 1: Foundation**. Catalog exposes versioned gRPC product creation, wired through its `CreateProduct` use case and PostgreSQL repository adapter. REST, Kafka, product lookup/listing, and category management remain future work.
```

with:

```markdown
The project is in **Milestone 1: Foundation**. Catalog exposes versioned gRPC product creation and retrieval by ID, wired through its `CreateProduct`/`GetProduct` use cases and PostgreSQL repository adapter. REST, Kafka, product listing, and category management remain future work.
```

- [ ] **Step 2: Commit the documentation change**

```bash
git add README.md
git commit -m "docs: document Catalog GetProduct workflow"
```

- [ ] **Step 3: Run the full local quality contract one last time**

```bash
make check
```

Expected: passes (fmt, vet, all unit tests).

- [ ] **Step 4: Run both tagged integration suites one last time**

```bash
make test-integration
make test-grpc-integration
```

Expected: both pass.

- [ ] **Step 5: Verify against the issue's acceptance criteria**

Confirm each of the following holds (all were exercised by tests above):

- A valid request for an existing product returns the persisted product (`TestCatalogServerGetProductReturnsPersistedProduct`, `TestCatalogServerGetProductPersistsThroughPostgreSQL`).
- A valid request for a missing product returns gRPC `NotFound` (`TestCatalogServerGetProductMapsMissingProductToNotFound`, `TestCatalogServerGetProductPersistsThroughPostgreSQL`).
- An empty ID returns gRPC `InvalidArgument` (`TestCatalogServerGetProductMapsEmptyIDToInvalidArgument`).
- A database failure returns gRPC `Internal` (`TestCatalogServerGetProductMapsRepositoryFailureToInternal`).
- `make check` remains database-free; tagged integration targets cover PostgreSQL and the end-to-end gRPC path (Steps 3–4 above).

- [ ] **Step 6: Push the branch**

```bash
git push -u origin issue_9
```
