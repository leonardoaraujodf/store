# Catalog Product Creation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the transport-free Catalog product-creation capability required by GitHub Issue #1.

**Architecture:** The domain package owns product invariants and knows no infrastructure. The application use case converts a command into a domain product and depends on a repository interface (a port) to save it; tests substitute an in-memory fake for that port.

**Tech Stack:** Go 1.26.5, standard library (`context`, `errors`, `testing`).

## Global Constraints

- Module path: `github.com/leonardoaraujodf/store`.
- Keep all Catalog implementation under `services/catalog/internal`; no other service may import it.
- Do not add gRPC, PostgreSQL, Kafka, containers, Kubernetes, or external dependencies in this issue.
- Follow one focused red-green-refactor cycle for every production behavior.
- Run `gofmt`, `go vet ./...`, and `go test ./...` before considering the issue complete.

---

## File structure

| Path | Responsibility |
| --- | --- |
| `services/catalog/internal/domain/product/product.go` | Product value and its invariant validation. |
| `services/catalog/internal/domain/product/product_test.go` | Domain behavior tests. |
| `services/catalog/internal/application/port/product_repository.go` | Repository port required by the application. |
| `services/catalog/internal/application/createproduct/create_product.go` | Create-product command and use case. |
| `services/catalog/internal/application/createproduct/create_product_test.go` | Use-case tests and a test-only in-memory repository fake. |

## Task 1: Create a valid domain Product

**Files:**
- Create: `services/catalog/internal/domain/product/product.go`
- Create: `services/catalog/internal/domain/product/product_test.go`

**Interfaces:**
- Produces: `func New(id, name, description string, priceMinorUnits int64, currency string) (Product, error)`.
- Produces: `type Product struct { ID, Name, Description string; PriceMinorUnits int64; Currency string }`.

- [X] **Step 1: Write the failing test**

```go
package product_test

import (
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

func TestNewCreatesProductWithValidAttributes(t *testing.T) {
	t.Parallel()

	got, err := product.New("product-123", "Keyboard", "Mechanical keyboard", 12_999, "BRL")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := product.Product{
		ID:              "product-123",
		Name:            "Keyboard",
		Description:     "Mechanical keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	}
	if got != want {
		t.Errorf("New() = %#v, want %#v", got, want)
	}
}
```

- [X] **Step 2: Run the test and verify the expected failure**

Run: `go test ./services/catalog/internal/domain/product`

Expected: failure because the package or `product.New` does not exist.

- [X] **Step 3: Write the minimal implementation**

```go
package product

type Product struct {
	ID              string
	Name            string
	Description     string
	PriceMinorUnits int64
	Currency        string
}

func New(id, name, description string, priceMinorUnits int64, currency string) (Product, error) {
	return Product{
		ID:              id,
		Name:            name,
		Description:     description,
		PriceMinorUnits: priceMinorUnits,
		Currency:        currency,
	}, nil
}
```

- [X] **Step 4: Run the focused test and format the new files**

Run: `gofmt -w services/catalog/internal/domain/product/*.go && go test ./services/catalog/internal/domain/product`

Expected: PASS.

- [X] **Step 5: Commit the completed TDD slice**

Run: `git add services/catalog/internal/domain/product && git commit -m "feat(catalog): add product value"`

Expected: a commit containing only the valid-product model and its test. Configure Git author name and email first if Git requests them.

## Task 2: Enforce Product invariants in the domain

**Files:**
- Modify: `services/catalog/internal/domain/product/product.go`
- Modify: `services/catalog/internal/domain/product/product_test.go`

**Interfaces:**
- Consumes: `product.New` and `product.Product` from Task 1.
- Produces: `ErrEmptyID`, `ErrEmptyName`, `ErrNegativePrice`, and `ErrInvalidCurrency` sentinel errors.

- [ ] **Step 1: Add one failing test for an empty ID**

```go
func TestNewRejectsEmptyID(t *testing.T) {
	t.Parallel()

	_, err := product.New("", "Keyboard", "Mechanical keyboard", 12_999, "BRL")
	if !errors.Is(err, product.ErrEmptyID) {
		t.Fatalf("New() error = %v, want %v", err, product.ErrEmptyID)
	}
}
```

Add `"errors"` to the test imports.

- [X] **Step 2: Run the test and verify the expected failure**

Run: `go test ./services/catalog/internal/domain/product -run '^TestNewRejectsEmptyID$'`

Expected: compilation failure because `product.ErrEmptyID` does not exist, or a test failure because an empty ID is accepted.

- [X] **Step 3: Implement only the empty-ID rule**

```go
var ErrEmptyID = errors.New("product ID must not be empty")

func New(id, name, description string, priceMinorUnits int64, currency string) (Product, error) {
	if id == "" {
		return Product{}, ErrEmptyID
	}
	return Product{
		ID:              id,
		Name:            name,
		Description:     description,
		PriceMinorUnits: priceMinorUnits,
		Currency:        currency,
	}, nil
}
```

Add `"errors"` to the production imports.

- [X] **Step 4: Repeat the same red-green cycle for the remaining three rules**

After the empty-ID cycle is green, add this complete test and introduce each row one at a time. Each newly introduced row must fail before adding its corresponding sentinel error and validation branch.

```go
func TestNewRejectsInvalidAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		id    string
		nameV string
		price int64
		curr  string
		want  error
	}{
		{"empty name", "product-123", "", 12_999, "BRL", product.ErrEmptyName},
		{"negative price", "product-123", "Keyboard", -1, "BRL", product.ErrNegativePrice},
		{"lowercase currency", "product-123", "Keyboard", 12_999, "brl", product.ErrInvalidCurrency},
		{"short currency", "product-123", "Keyboard", 12_999, "BR", product.ErrInvalidCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := product.New(tt.id, tt.nameV, "description", tt.price, tt.curr)
			if !errors.Is(err, tt.want) {
				t.Fatalf("New() error = %v, want %v", err, tt.want)
			}
		})
	}
}
```

For `ErrInvalidCurrency`, accept exactly three ASCII uppercase letters (`A` through `Z`). Keep zero as a valid price; Issue #1 rejects negative prices, not free products.

- [X] **Step 5: Run the complete domain package and commit**

Run: `gofmt -w services/catalog/internal/domain/product/*.go && go test ./services/catalog/internal/domain/product && git add services/catalog/internal/domain/product && git commit -m "feat(catalog): validate product attributes"`

Expected: all domain tests PASS.

## Task 3: Define the repository port and save valid products

**Files:**
- Create: `services/catalog/internal/application/port/product_repository.go`
- Create: `services/catalog/internal/application/createproduct/create_product.go`
- Create: `services/catalog/internal/application/createproduct/create_product_test.go`

**Interfaces:**
- Consumes: `product.New` from Task 2.
- Produces: `type ProductRepository interface { Save(context.Context, product.Product) error }`.
- Produces: `type Command` and `func (UseCase) Execute(context.Context, Command) (product.Product, error)`.

- [X] **Step 1: Write the failing application test with a test-only fake**

```go
package createproduct_test

import (
	"context"
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type fakeRepository struct {
	saved []product.Product
}

func (f *fakeRepository) Save(_ context.Context, p product.Product) error {
	f.saved = append(f.saved, p)
	return nil
}

func TestUseCaseExecuteSavesValidProduct(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	useCase := createproduct.New(repository)
	command := createproduct.Command{
		ID: "product-123", Name: "Keyboard", Description: "Mechanical keyboard",
		PriceMinorUnits: 12_999, Currency: "BRL",
	}

	got, err := useCase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(repository.saved) != 1 {
		t.Fatalf("saved = %#v, want one product", repository.saved)
	}
	if got != repository.saved[0] {
		t.Fatalf("result = %#v, saved = %#v", got, repository.saved[0])
	}
}
```

- [X] **Step 2: Run the test and verify the expected failure**

Run: `go test ./services/catalog/internal/application/createproduct -run '^TestUseCaseExecuteSavesValidProduct$'`

Expected: failure because the application packages and use case do not exist.

- [X] **Step 3: Implement the port and minimal use case**

```go
// application/port/product_repository.go
package port

import (
	"context"

	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type ProductRepository interface {
	Save(context.Context, product.Product) error
}
```

```go
// application/createproduct/create_product.go
package createproduct

import (
	"context"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/port"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type Command struct {
	ID, Name, Description string
	PriceMinorUnits       int64
	Currency              string
}

type UseCase struct{ repository port.ProductRepository }

func New(repository port.ProductRepository) UseCase { return UseCase{repository: repository} }

func (u UseCase) Execute(ctx context.Context, command Command) (product.Product, error) {
	p, err := product.New(command.ID, command.Name, command.Description, command.PriceMinorUnits, command.Currency)
	if err != nil { return product.Product{}, err }
	if err := u.repository.Save(ctx, p); err != nil { return product.Product{}, err }
	return p, nil
}
```

- [X] **Step 4: Format and run the focused test**

Run: `gofmt -w services/catalog/internal/application/{port,createproduct}/*.go && go test ./services/catalog/internal/application/createproduct -run '^TestUseCaseExecuteSavesValidProduct$'`

Expected: PASS.

- [X] **Step 5: Commit the port and successful application flow**

Run: `git add services/catalog/internal/application && git commit -m "feat(catalog): save products through repository port"`

Expected: a commit with the port, use case, and fake-backed test.

## Task 4: Prevent invalid persistence and return repository failures

**Files:**
- Modify: `services/catalog/internal/application/createproduct/create_product_test.go`

**Interfaces:**
- Consumes: the `UseCase`, `Command`, and `ProductRepository` from Task 3.
- Verifies: invalid commands return domain errors without calling `Save`; errors from `Save` are returned unchanged.

- [ ] **Step 1: Write an invalid-command test that proves the use case composes the domain and port correctly**

```go
func TestUseCaseExecuteDoesNotSaveInvalidProduct(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	useCase := createproduct.New(repository)
	_, err := useCase.Execute(context.Background(), createproduct.Command{
		Name: "Keyboard", PriceMinorUnits: 12_999, Currency: "BRL",
	})
	if !errors.Is(err, product.ErrEmptyID) {
		t.Fatalf("Execute() error = %v, want %v", err, product.ErrEmptyID)
	}
	if len(repository.saved) != 0 {
		t.Fatalf("saved = %#v, want no products", repository.saved)
	}
}
```

Add `"errors"` to imports.

- [ ] **Step 2: Run the test and verify the expected result**

Run: `go test ./services/catalog/internal/application/createproduct -run '^TestUseCaseExecuteDoesNotSaveInvalidProduct$'`

Expected: PASS without production changes: the use case asks the domain to validate before it calls the port. If it fails, fix the ordering in `Execute` before proceeding.

- [ ] **Step 3: Extend the fake and write a repository-error test**

```go
type fakeRepository struct {
	saved   []product.Product
	saveErr error
}

func (f *fakeRepository) Save(_ context.Context, p product.Product) error {
	if f.saveErr != nil { return f.saveErr }
	f.saved = append(f.saved, p)
	return nil
}

func TestUseCaseExecuteReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("repository unavailable")
	useCase := createproduct.New(&fakeRepository{saveErr: wantErr})
	_, err := useCase.Execute(context.Background(), createproduct.Command{
		ID: "product-123", Name: "Keyboard", PriceMinorUnits: 12_999, Currency: "BRL",
	})
	if !errors.Is(err, wantErr) { t.Fatalf("Execute() error = %v, want %v", err, wantErr) }
}
```

- [X] **Step 4: Run the repository-error test, then make the smallest repair if needed**

Run: `go test ./services/catalog/internal/application/createproduct -run '^TestUseCaseExecuteReturnsRepositoryError$'`

Expected: PASS if `Execute` returns the `Save` error. Otherwise, return that error directly from `Execute` and rerun the test.

- [X] **Step 5: Run all quality checks and commit**

Run: `gofmt -w services/catalog/internal/application/createproduct/*.go && go vet ./... && go test ./... && git add services/catalog/internal/application/createproduct && git commit -m "test(catalog): cover create product failures"`

Expected: formatter produces no remaining changes, `go vet ./...` exits 0, and `go test ./...` reports PASS.
