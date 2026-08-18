# Catalog gRPC Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Catalog product creation available through a runnable, versioned gRPC service backed by its PostgreSQL repository.

**Architecture:** The protobuf contract is the public driving boundary. A gRPC adapter translates generated requests into `createproduct.Command`, invokes the existing application use case, and maps results/errors to gRPC. `cmd/catalog` is the composition root: configuration, PostgreSQL, server startup, and shutdown only.

**Tech Stack:** Go 1.26.5, gRPC-Go, Protocol Buffers, Buf v2, pgxpool, PostgreSQL Compose, and optional direnv.

## Global Constraints

- Domain/application packages must not import protobuf, gRPC, or PostgreSQL adapter packages.
- No REST, Kafka, lookup/listing, categories, Catalog application container, observability, Kubernetes, or CI expansion.
- `CATALOG_GRPC_ADDR` and `CATALOG_DATABASE_URL` are mandatory; neither has a default.
- Generated files below `gen/go` are committed; remote generators are pinned.
- Apply red, green, refactor to behavior. `make check` remains database-free.

---

## File structure

| Path | Responsibility |
| --- | --- |
| `buf.yaml`, `buf.gen.yaml` | Buf module, lint, and pinned generation setup. |
| `proto/catalog/v1/catalog.proto` | Versioned public contract. |
| `gen/go/catalog/v1/*.pb.go` | Generated, committed Go types and service registration. |
| `services/catalog/internal/config` | Mandatory process-environment configuration. |
| `services/catalog/internal/adapter/grpc` | gRPC driving adapter and transport tests. |
| `services/catalog/cmd/catalog` | Runtime composition root and graceful lifecycle. |
| `.envrc.example`, `Makefile`, `README.md` | Local developer workflow. |

### Task 1: Define and generate the versioned Catalog gRPC contract

**Files:**
- Create: `buf.yaml`, `buf.gen.yaml`, `proto/catalog/v1/catalog.proto`
- Create: `gen/go/catalog/v1/catalog.pb.go`, `gen/go/catalog/v1/catalog_grpc.pb.go`
- Modify: `Makefile`

**Interfaces:**
- Produces package `catalogv1` at `github.com/leonardoaraujodf/store/gen/go/catalog/v1`.
- Produces `CatalogService.CreateProduct(context.Context, *CreateProductRequest) (*CreateProductResponse, error)`.

- [X] **Step 1: Install Buf and check the executable**

```bash
go install github.com/bufbuild/buf/cmd/buf@v1.69.0
buf --version
```

Expected: a Buf version is printed. Ensure the Go bin directory is in `PATH`.

- [ ] **Step 2: Add Buf v2 configuration**

Create `buf.yaml`:

```yaml
version: v2
modules:
  - path: proto
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

Create `buf.gen.yaml`:

```yaml
version: v2
clean: true
plugins:
  - remote: buf.build/protocolbuffers/go:v1.35.2
    out: gen/go
    opt:
      - paths=source_relative
  - remote: buf.build/grpc/go:v1.5.1
    out: gen/go
    opt:
      - paths=source_relative
```

- [X] **Step 3: Add the source contract**

Create `proto/catalog/v1/catalog.proto`:

```proto
syntax = "proto3";

package catalog.v1;

option go_package = "github.com/leonardoaraujodf/store/gen/go/catalog/v1;catalogv1";

service CatalogService {
  rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse);
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

message Product {
  string id = 1;
  string name = 2;
  string description = 3;
  int64 price_minor_units = 4;
  string currency = 5;
}
```

Field numbers are public compatibility commitments: do not reorder or reuse them.

- [ ] **Step 4: Add Make targets**

Extend `.PHONY`, preserving all existing targets, and add:

```make
proto-format:
	buf format -w proto

proto-lint:
	buf lint

proto-generate:
	buf generate

proto-check:
	buf format --exit-code proto
	buf lint
	buf generate
	git diff --exit-code -- gen/go
```

- [X] **Step 5: Generate and verify**

```bash
make proto-format
make proto-lint
make proto-generate
make proto-check
```

Expected: both generated Go files appear below `gen/go/catalog/v1`; all commands exit zero. Never edit those generated files.

- [X] **Step 6: Commit**

```bash
git add buf.yaml buf.gen.yaml proto/catalog/v1/catalog.proto gen/go Makefile
git commit -m "feat: add Catalog gRPC contract"
```

### Task 2: Add fail-fast Catalog runtime configuration and local direnv example

**Files:**
- Create: `services/catalog/internal/config/config.go`
- Create: `services/catalog/internal/config/config_test.go`
- Create: `.envrc.example`
- Modify: `.gitignore`

**Interfaces:**
- Produces `config.Config{GRPCAddr string, DatabaseURL string}`.
- Produces `config.Load() (config.Config, error)`.

- [X] **Step 1: Write the failing tests**

Create `services/catalog/internal/config/config_test.go`:

```go
package config_test

import (
	"strings"
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/config"
)

func TestLoadReturnsErrorWhenRequiredVariableIsMissing(t *testing.T) {
	t.Setenv("CATALOG_GRPC_ADDR", ":50051")
	t.Setenv("CATALOG_DATABASE_URL", "")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "CATALOG_DATABASE_URL") {
		t.Fatalf("Load() error = %v, want error naming CATALOG_DATABASE_URL", err)
	}
}

func TestLoadReturnsValidatedConfiguration(t *testing.T) {
	t.Setenv("CATALOG_GRPC_ADDR", ":50051")
	t.Setenv("CATALOG_DATABASE_URL", "postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable")

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.GRPCAddr != ":50051" || got.DatabaseURL != "postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable" {
		t.Fatalf("Load() = %#v, want configured values", got)
	}
}

func TestLoadReturnsErrorForInvalidDatabaseURL(t *testing.T) {
	t.Setenv("CATALOG_GRPC_ADDR", ":50051")
	t.Setenv("CATALOG_DATABASE_URL", "not a database url")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "CATALOG_DATABASE_URL") {
		t.Fatalf("Load() error = %v, want error naming CATALOG_DATABASE_URL", err)
	}
}
```

- [X] **Step 2: Confirm the red state**

```bash
go test ./services/catalog/internal/config
```

Expected: FAIL because the package and `config.Load` do not exist.

- [X] **Step 3: Implement the minimal loader**

Create `services/catalog/internal/config/config.go`:

```go
package config

import (
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	GRPCAddr    string
	DatabaseURL string
}

func Load() (Config, error) {
	grpcAddr := os.Getenv("CATALOG_GRPC_ADDR")
	if grpcAddr == "" {
		return Config{}, fmt.Errorf("CATALOG_GRPC_ADDR must be set")
	}
	databaseURL := os.Getenv("CATALOG_DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("CATALOG_DATABASE_URL must be set")
	}
	parsed, err := url.ParseRequestURI(databaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return Config{}, fmt.Errorf("CATALOG_DATABASE_URL must be a PostgreSQL connection URL")
	}
	return Config{GRPCAddr: grpcAddr, DatabaseURL: databaseURL}, nil
}
```

- [X] **Step 4: Verify green**

```bash
gofmt -w services/catalog/internal/config
go test ./services/catalog/internal/config
```

Expected: PASS. `t.Setenv` makes tests independent from shell environment.

- [X] **Step 5: Add optional direnv configuration**

Add `.envrc` to `.gitignore`. Create `.envrc.example`:

```bash
export CATALOG_GRPC_ADDR=:50051
export CATALOG_DATABASE_URL=postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable
```

Use it locally with:

```bash
cp .envrc.example .envrc
direnv allow
```

No Go package reads this file.

- [X] **Step 6: Commit**

```bash
git add services/catalog/internal/config .envrc.example .gitignore
git commit -m "feat: add Catalog runtime configuration"
```

### Task 3: Implement and unit-test the Catalog gRPC driving adapter

**Files:**
- Create: `services/catalog/internal/adapter/grpc/catalog_server.go`
- Create: `services/catalog/internal/adapter/grpc/catalog_server_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes `createproduct.UseCase.Execute(context.Context, createproduct.Command) (product.Product, error)`.
- Produces `grpcadapter.NewCatalogServer(createproduct.UseCase) *grpcadapter.CatalogServer`, implementing `catalogv1.CatalogServiceServer`.

- [X] **Step 1: Add the gRPC runtime dependency**

```bash
go get google.golang.org/grpc@v1.76.0
go mod tidy
```

Expected: `go.mod` requires `google.golang.org/grpc`, and Go records generated-code/runtime transitive dependencies in `go.sum`.

- [X] **Step 2: Write a failing gRPC transport test**

Create `services/catalog/internal/adapter/grpc/catalog_server_test.go`. It must use a generated client against an in-memory `bufconn` listener—not call the server method directly—so serialization and gRPC status mapping are exercised.

```go
package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	grpcadapter "github.com/leonardoaraujodf/store/services/catalog/internal/adapter/grpc"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type fakeRepository struct {
	saved   []product.Product
	saveErr error
}

func (r *fakeRepository) Save(_ context.Context, p product.Product) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, p)
	return nil
}

func newCatalogClient(t *testing.T, useCase createproduct.UseCase) catalogv1.CatalogServiceClient {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(server, grpcadapter.NewCatalogServer(useCase))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	connection, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
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
	client := newCatalogClient(t, createproduct.New(repository))

	response, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Id: "product-123", Name: "Keyboard", Description: "Mechanical keyboard",
		PriceMinorUnits: 12_999, Currency: "BRL",
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
	client := newCatalogClient(t, createproduct.New(&fakeRepository{}))
	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Name: "Keyboard", PriceMinorUnits: 12_999, Currency: "BRL",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestCatalogServerCreateProductMapsRepositoryFailureToInternal(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{saveErr: errors.New("database unavailable")}))
	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Id: "product-123", Name: "Keyboard", PriceMinorUnits: 12_999, Currency: "BRL",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.Internal, err)
	}
}
```

- [X] **Step 3: Confirm the red state**

```bash
go test ./services/catalog/internal/adapter/grpc
```

Expected: FAIL because the `grpc` adapter package and `NewCatalogServer` do not exist.

- [X] **Step 4: Implement the adapter**

Create `services/catalog/internal/adapter/grpc/catalog_server.go`:

```go
package grpcadapter

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type CatalogServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	useCase createproduct.UseCase
}

func NewCatalogServer(useCase createproduct.UseCase) *CatalogServer {
	return &CatalogServer{useCase: useCase}
}

func (s *CatalogServer) CreateProduct(ctx context.Context, request *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	p, err := s.useCase.Execute(ctx, createproduct.Command{
		ID: request.GetId(), Name: request.GetName(), Description: request.GetDescription(),
		PriceMinorUnits: request.GetPriceMinorUnits(), Currency: request.GetCurrency(),
	})
	if err != nil {
		if isProductValidationError(err) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "could not create product")
	}
	return &catalogv1.CreateProductResponse{Product: &catalogv1.Product{
		Id: p.ID, Name: p.Name, Description: p.Description,
		PriceMinorUnits: p.PriceMinorUnits, Currency: p.Currency,
	}}, nil
}

func isProductValidationError(err error) bool {
	return errors.Is(err, product.ErrEmptyID) ||
		errors.Is(err, product.ErrEmptyName) ||
		errors.Is(err, product.ErrNegativePrice) ||
		errors.Is(err, product.ErrInvalidCurrency)
}
```

- [X] **Step 5: Verify green and commit**

```bash
gofmt -w services/catalog/internal/adapter/grpc
go test ./services/catalog/internal/adapter/grpc
go test ./services/catalog/internal/application/createproduct ./services/catalog/internal/domain/product
git add go.mod go.sum services/catalog/internal/adapter/grpc
git commit -m "feat: add Catalog gRPC adapter"
```

Expected: all tests pass. The adapter must contain no SQL and no duplicated product validation.

### Task 4: Wire a runnable Catalog process with graceful shutdown

**Files:**
- Create: `services/catalog/cmd/catalog/main.go`
- Modify: `Makefile`

**Interfaces:**
- Consumes `config.Load`, `pgxpool.New`, `postgres.NewProductRepository`, `createproduct.New`, and `grpcadapter.NewCatalogServer`.
- Produces executable package `./services/catalog/cmd/catalog` and `make run-catalog`.

- [X] **Step 1: Establish the runtime acceptance check**

```bash
make run-catalog
```

Expected before implementation: FAIL because the command package does not exist. After implementation, with PostgreSQL migrated and the two variables exported, it prints a listening message, remains running, and returns cleanly after Ctrl-C.

- [X] **Step 2: Implement the composition root**

Create `services/catalog/cmd/catalog/main.go`:

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
	"google.golang.org/grpc"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	grpcadapter "github.com/leonardoaraujodf/store/services/catalog/internal/adapter/grpc"
	"github.com/leonardoaraujodf/store/services/catalog/internal/adapter/postgres"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/config"
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
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", cfg.GRPCAddr, err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	repository := postgres.NewProductRepository(pool)
	catalogv1.RegisterCatalogServiceServer(server, grpcadapter.NewCatalogServer(createproduct.New(repository)))
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.Serve(listener) }()
	fmt.Printf("Catalog gRPC service listening on %s\n", cfg.GRPCAddr)

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-signalContext.Done():
		server.GracefulStop()
		return nil
	case err := <-serveErrors:
		return fmt.Errorf("serve gRPC: %w", err)
	}
}
```

- [X] **Step 3: Add and run the Make target**

Add `run-catalog` to `.PHONY` and add:

```make
run-catalog:
	go run ./services/catalog/cmd/catalog
```

Then verify fail-fast configuration:

```bash
env -u CATALOG_GRPC_ADDR -u CATALOG_DATABASE_URL go run ./services/catalog/cmd/catalog
```

Expected: nonzero exit and stderr naming `CATALOG_GRPC_ADDR`. Then load the example variables, run `make db-up migrate-up`, run `make run-catalog`, and interrupt it with Ctrl-C.

- [X] **Step 4: Verify and commit**

```bash
make check
git add services/catalog/cmd/catalog/main.go Makefile
git commit -m "feat: run Catalog gRPC service"
```

Expected: `make check` passes without starting Docker.

### Task 5: Prove the complete gRPC-to-PostgreSQL path with a tagged integration test

**Files:**
- Create: `services/catalog/internal/adapter/grpc/catalog_server_integration_test.go`
- Modify: `Makefile`

**Interfaces:**
- Consumes the generated `catalogv1` client/server, the real PostgreSQL repository, and `CATALOG_DATABASE_URL`.
- Produces `make test-grpc-integration`.

- [ ] **Step 1: Write the tagged end-to-end test**

Create `services/catalog/internal/adapter/grpc/catalog_server_integration_test.go`:

```go
//go:build integration

package grpc_test

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	grpcadapter "github.com/leonardoaraujodf/store/services/catalog/internal/adapter/grpc"
	"github.com/leonardoaraujodf/store/services/catalog/internal/adapter/postgres"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
)

func TestCatalogServerCreateProductPersistsThroughPostgreSQL(t *testing.T) {
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
	catalogv1.RegisterCatalogServiceServer(server, grpcadapter.NewCatalogServer(createproduct.New(repository)))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	connection, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	client := catalogv1.NewCatalogServiceClient(connection)

	response, err := client.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Id: "product-123", Name: "Keyboard", Description: "Mechanical keyboard",
		PriceMinorUnits: 12_999, Currency: "BRL",
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	if response.GetProduct().GetId() != "product-123" {
		t.Fatalf("response product ID = %q, want %q", response.GetProduct().GetId(), "product-123")
	}

	var persistedName string
	if err := pool.QueryRow(ctx, "SELECT name FROM products WHERE id = $1", "product-123").Scan(&persistedName); err != nil {
		t.Fatalf("QueryRow().Scan() error = %v", err)
	}
	if persistedName != "Keyboard" {
		t.Errorf("persisted name = %q, want %q", persistedName, "Keyboard")
	}

	_, err = client.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Name: "Invalid product", PriceMinorUnits: 12_999, Currency: "BRL",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
	var invalidRows int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM products WHERE name = $1", "Invalid product").Scan(&invalidRows); err != nil {
		t.Fatalf("count invalid products error = %v", err)
	}
	if invalidRows != 0 {
		t.Errorf("invalid persisted rows = %d, want 0", invalidRows)
	}
}
```

- [ ] **Step 2: Confirm the red state**

```bash
CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' go test -tags=integration ./services/catalog/internal/adapter/grpc
```

Expected before Task 3 exists: compile failure naming `grpcadapter.NewCatalogServer`. After Task 3 but before PostgreSQL is prepared: a clear connection/table failure. Neither outcome is green.

- [ ] **Step 3: Add the integration target**

Add it to `.PHONY` and add:

```make
test-grpc-integration: db-up migrate-up
	CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' go test -tags=integration ./services/catalog/internal/adapter/grpc
```

- [ ] **Step 4: Verify green**

```bash
make test-grpc-integration
```

Expected: Compose starts PostgreSQL and migrations; the generated client receives the product; direct SQL finds it; invalid input returns `codes.InvalidArgument` and persists no row.

- [ ] **Step 5: Run all checks and commit**

```bash
make check
make proto-check
make test-integration
make test-grpc-integration
git add services/catalog/internal/adapter/grpc/catalog_server_integration_test.go Makefile
git commit -m "test: cover Catalog gRPC PostgreSQL flow"
```

### Task 6: Document local gRPC development and complete the issue checklist

**Files:**
- Modify: `README.md`
- Modify: `docs/superpowers/plans/2026-08-14-catalog-grpc-service.md`

**Interfaces:**
- Produces documentation for contract generation, optional direnv setup, service startup, and database-backed gRPC validation.

- [ ] **Step 1: Update the current-state statement**

Replace the claim that gRPC is not introduced. State that Catalog exposes only versioned gRPC product creation, wired through the use case and PostgreSQL repository. State that REST, Kafka, lookup/listing, and categories remain future work.

- [ ] **Step 2: Add a Catalog gRPC contract section**

Add this content to `README.md`:

```markdown
## Catalog gRPC contract

The versioned Catalog contract is at `proto/catalog/v1/catalog.proto`. Generated Go files in `gen/go` are committed: edit protobuf source and regenerate; never edit generated files.

```bash
make proto-format
make proto-lint
make proto-generate
make proto-check
```
```

- [ ] **Step 3: Add a local startup section**

Add this content:

```markdown
## Run Catalog gRPC locally

Catalog requires `CATALOG_GRPC_ADDR` and `CATALOG_DATABASE_URL`; it exits clearly if either is absent or the database URL is invalid. For optional local direnv setup:

```bash
cp .envrc.example .envrc
direnv allow
make db-up
make migrate-up
make run-catalog
```

`.envrc` is local and ignored. Production and other environments export the same variables; the Go program does not load `.envrc`.
```

- [ ] **Step 4: Add the gRPC integration command**

Document `make test-grpc-integration` next to the existing PostgreSQL integration command. Explain it runs an in-process gRPC server backed by real PostgreSQL, and keep the explicit statement that `make check` remains database-free.

- [ ] **Step 5: Mark completed work and verify**

Change only finished checklist items in this plan to `- [x]`, then run:

```bash
make check
make proto-check
git diff --check
git status --short
```

Expected: quality and contract checks pass, whitespace checks are clean, and only intended Issue #7 work appears in status.

- [ ] **Step 6: Commit documentation**

```bash
git add README.md docs/superpowers/plans/2026-08-14-catalog-grpc-service.md
git commit -m "docs: document Catalog gRPC workflow"
```

## Final issue verification

- [ ] Confirm package `catalog.v1` and all Task 1 protobuf field numbers are unchanged.
- [ ] Confirm domain/application packages import neither `catalogv1`, `grpc`, nor `pgx`.
- [ ] Confirm missing configuration exits nonzero and prints the missing variable name.
- [ ] Run `make check`, `make proto-check`, `make test-integration`, and `make test-grpc-integration`.
- [ ] Review `git diff origin/main...HEAD`, push `issue_7`, open a PR linked to GitHub issue #7, and request review.
