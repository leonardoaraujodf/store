package product_test

import (
	"errors"
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

func TestNewCreatesProductWithValidAttributes(t *testing.T) {
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

func TestNewRejectsInvalidAttributes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		testname    string
		id          string
		name        string
		description string
		price       int64
		currency    string
		want        error
	}{
		{"empty id", "", "keyboard", "Mechanical keyboard", 12_999, "BRL", product.ErrEmptyID},
		{"empty name", "product-123", "", "Mechanical keyboard", 12_999, "BRL", product.ErrEmptyName},
		{"negative price", "product-123", "keyboard", "Mechanical keyboard", -1, "BRL", product.ErrNegativePrice},
		{"lowercase currency", "product-123", "keyboard", "Mechanical keyboard", 12_999, "brl", product.ErrInvalidCurrency},
		{"short currency", "product-123", "keyboard", "Mechanical keyboard", 12_999, "BR", product.ErrInvalidCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			_, err := product.New(tt.id, tt.name, tt.description, tt.price, tt.currency)
			if !errors.Is(err, tt.want) {
				t.Fatalf("New() error = %v, want %v", err, tt.want)
			}
		})
	}
}
