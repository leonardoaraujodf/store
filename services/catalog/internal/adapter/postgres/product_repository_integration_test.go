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
		"Keyboard",
		"Mechanical keyboard",
		12_999,
		"BRL",
	)
	if err != nil {
		t.Fatalf("product.New() error = %v", err)
	}
	repository := postgres.NewProductRepository(pool)
	saved, err := repository.Save(ctx, want)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if saved.ID == 0 {
		t.Fatalf("Save() saved.ID = 0, want nonzero")
	}
	var got product.Product
	err = pool.QueryRow(
		ctx,
		`SELECT id, name, description, price_minor_units, currency
		FROM products
		WHERE id = $1`,
		saved.ID,
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

	if got != saved {
		t.Errorf("persisted product = %#v, want %#v", got, saved)
	}
}

func TestProductRepositoryFindByIDReturnsPersistedProduct(t *testing.T) {
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
		"Keyboard",
		"Mechanical keyboard",
		12_999,
		"BRL",
	)
	if err != nil {
		t.Fatalf("product.New() error = %v", err)
	}
	repository := postgres.NewProductRepository(pool)
	saved, err := repository.Save(ctx, want)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, found, err := repository.FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if !found {
		t.Fatalf("FindByID() found = false, want true")
	}
	if got != saved {
		t.Errorf("FindByID() = %#v, want %#v", got, saved)
	}
}

func TestProductRepositoryFindByIDReturnsNotFoundForMissingProduct(t *testing.T) {
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

	repository := postgres.NewProductRepository(pool)
	got, found, err := repository.FindByID(ctx, int64(999_999_999))
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found {
		t.Fatalf("FindByID() found = true, want false")
	}
	if got != (product.Product{}) {
		t.Errorf("FindByID() = %#v, want zero value", got)
	}
}

func TestProductRepositorySaveAssignsDistinctSequentialIDs(t *testing.T) {
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

	first, err := product.New(
		"Keyboard",
		"Mechanical keyboard",
		12_999,
		"BRL",
	)
	if err != nil {
		t.Fatalf("product.New() error = %v", err)
	}
	second, err := product.New(
		"Mouse",
		"Wireless mouse",
		4_999,
		"BRL",
	)
	if err != nil {
		t.Fatalf("product.New() error = %v", err)
	}

	repository := postgres.NewProductRepository(pool)

	savedFirst, err := repository.Save(ctx, first)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	savedSecond, err := repository.Save(ctx, second)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if savedFirst.ID == 0 {
		t.Fatalf("Save() savedFirst.ID = 0, want nonzero")
	}
	if savedSecond.ID == 0 {
		t.Fatalf("Save() savedSecond.ID = 0, want nonzero")
	}
	if savedFirst.ID == savedSecond.ID {
		t.Fatalf("Save() assigned duplicate IDs: %d", savedFirst.ID)
	}
	if savedSecond.ID <= savedFirst.ID {
		t.Errorf("savedSecond.ID = %d, want greater than savedFirst.ID = %d", savedSecond.ID, savedFirst.ID)
	}
}
