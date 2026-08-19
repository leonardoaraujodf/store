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

func (f *fakeRepository) Save(_ context.Context, p product.Product) (product.Product, error) {
	if f.saveErr != nil {
		return product.Product{}, f.saveErr
	}
	p.ID = int64(len(f.saved)) + 1
	f.saved = append(f.saved, p)
	return p, nil
}

func (f *fakeRepository) FindByID(_ context.Context, id int64) (product.Product, bool, error) {
	return product.Product{}, false, nil
}

func TestUseCaseExecuteSavesValidProduct(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{}
	useCase := createproduct.New(repository)
	command := createproduct.Command{
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
	if got.ID == 0 {
		t.Fatalf("got.ID = %v, want non-zero", got.ID)
	}
}

func TestUseCaseExecuteDoesNotSaveInvalidProduct(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	useCase := createproduct.New(repository)
	_, err := useCase.Execute(context.Background(), createproduct.Command{
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if !errors.Is(err, product.ErrEmptyName) {
		t.Fatalf("Execute() error = %v, want %v", err, product.ErrEmptyName)
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
		Name:            "Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
