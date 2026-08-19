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
