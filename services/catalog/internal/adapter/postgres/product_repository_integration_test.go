//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonardoaraujodf/store/services/catalog/internal/adapter/postgres"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

func TestProductRepositorySavePersistProduct(t *testing.T) {
	ctx := context.Background()

	databaseURL := os.Getenv("CATALOG_DATABASE_URL")
	if databaseURL == "" {
		t.Fatalf("CATALOG_DATABASE_URL must be set")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE products"); err != nil {
		t.Fatalf("TRUNCATE products error = %v", err)
	}
	want, err := product.New(
		"product-123",
		"Keyboard",
		"Mechanical keyboard",
		12_999,
		"BRL",
	)
	if err != nil {
		t.Fatalf("product.New() error = %v", err)
	}
	repository := postgres.NewProductRepository(pool)
	if err := repository.Save(ctx, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	var got product.Product
	err = pool.QueryRow(
		ctx,
		`SELECT id, name, description, price_minor_units, currency
		FROM products
		WHERE id = $1`,
		want.ID,
	).Scan(
		&got.ID,
		&got.Name,
		&got.Description,
		&got.PriceMinorUnits,
		&got.Currency,
	)
	if err != nil {
		t.Fatalf("QueryRow().Scan() error = %v", err)
	}

	if got != want {
		t.Errorf("persisted product = %#v, want %#v", got, want)
	}
}
