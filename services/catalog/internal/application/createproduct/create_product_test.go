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
