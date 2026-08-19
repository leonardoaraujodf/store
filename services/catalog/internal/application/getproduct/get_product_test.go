package getproduct_test

import (
	"context"
	"errors"
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type fakeRepository struct {
	products map[int64]product.Product
	findErr  error
}

func (f *fakeRepository) Save(_ context.Context, p product.Product) (product.Product, error) {
	if f.products == nil {
		f.products = map[int64]product.Product{}
	}
	p.ID = int64(len(f.products)) + 1
	f.products[p.ID] = p
	return p, nil
}

func (f *fakeRepository) FindByID(_ context.Context, id int64) (product.Product, bool, error) {
	if f.findErr != nil {
		return product.Product{}, false, f.findErr
	}
	p, found := f.products[id]
	return p, found, nil
}

func TestUseCaseExecuteReturnsExistingProduct(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	p, err := product.New("Keyboard", "Mechanical Keyboard", 12_999, "BRL")
	if err != nil {
		t.Fatalf("product.New() error = %v", err)
	}
	want, err := repository.Save(context.Background(), p)
	if err != nil {
		t.Fatalf("repository.Save() error = %v", err)
	}

	useCase := getproduct.New(repository)
	got, err := useCase.Execute(context.Background(), getproduct.Command{ID: want.ID})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != want {
		t.Fatalf("result = %#v, want %#v", got, want)
	}
}

func TestUseCaseExecuteReturnsErrInvalidIDForNonPositiveID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   int64
	}{
		{name: "zero", id: 0},
		{name: "negative", id: -1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := getproduct.New(&fakeRepository{})
			_, err := useCase.Execute(context.Background(), getproduct.Command{ID: tt.id})
			if !errors.Is(err, getproduct.ErrInvalidID) {
				t.Fatalf("Execute() error = %v, want %v", err, getproduct.ErrInvalidID)
			}
		})
	}
}

func TestUseCaseExecuteReturnsErrProductNotFoundForMissingProduct(t *testing.T) {
	t.Parallel()

	useCase := getproduct.New(&fakeRepository{})
	_, err := useCase.Execute(context.Background(), getproduct.Command{ID: 999_999_999})
	if !errors.Is(err, getproduct.ErrProductNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, getproduct.ErrProductNotFound)
	}
}

func TestUseCaseExecuteReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("repository unavailable")
	useCase := getproduct.New(&fakeRepository{findErr: wantErr})
	_, err := useCase.Execute(context.Background(), getproduct.Command{ID: 1})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
