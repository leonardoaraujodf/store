package createproduct_test

import (
	"context"
	"errors"
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type fakeRepository struct {
	saved   []product.Product
	saveErr error
}

func (f *fakeRepository) Save(_ context.Context, p product.Product) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, p)
	return nil
}

func TestUseCaseExecuteSavesValidProduct(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	useCase := createproduct.New(repository)
	command := createproduct.Command{
		ID:              "product-123",
		Name:            "Keyboard",
		Description:     "Mechanical Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
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

func TestUseCaseExecuteDoesNotSaveInvalidProduct(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	useCase := createproduct.New(repository)
	_, err := useCase.Execute(context.Background(), createproduct.Command{
		Name:            "Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if !errors.Is(err, product.ErrEmptyID) {
		t.Fatalf("Execute() error = %v, want %v", err, product.ErrEmptyID)
	}
	if len(repository.saved) != 0 {
		t.Fatalf("saved = %#v, want no products", repository.saved)
	}
}

func TestUseCaseExecuteReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("repository unavailable")
	useCase := createproduct.New(&fakeRepository{saveErr: wantErr})
	_, err := useCase.Execute(context.Background(), createproduct.Command{
		ID:              "product-123",
		Name:            "Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
