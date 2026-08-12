package product_test

import (
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
