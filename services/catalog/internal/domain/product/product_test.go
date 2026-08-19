package product_test

import (
	"errors"
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

func TestNewCreatesProductWithValidAttributes(t *testing.T) {
	got, err := product.New("Keyboard", "Mechanical keyboard", 12_999, "BRL")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := product.Product{
		ID:              0,
		Name:            "Keyboard",
		Description:     "Mechanical keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	}
	if got != want {
		t.Errorf("New() = %#v, want %#v", got, want)
	}
}

func TestNewRejectsInvalidAttributes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		testname    string
		name        string
		description string
		price       int64
		currency    string
		want        error
	}{
		{"empty name", "", "Mechanical keyboard", 12_999, "BRL", product.ErrEmptyName},
		{"negative price", "keyboard", "Mechanical keyboard", -1, "BRL", product.ErrNegativePrice},
		{"lowercase currency", "keyboard", "Mechanical keyboard", 12_999, "brl", product.ErrInvalidCurrency},
		{"short currency", "keyboard", "Mechanical keyboard", 12_999, "BR", product.ErrInvalidCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			_, err := product.New(tt.name, tt.description, tt.price, tt.currency)
			if !errors.Is(err, tt.want) {
				t.Fatalf("New() error = %v, want %v", err, tt.want)
			}
		})
	}
}
