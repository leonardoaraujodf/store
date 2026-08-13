package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (r *Repository) Save(ctx context.Context, product product.Product) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO products(id, name, description, price_minor_units, currency)
		 VALUES($1, $2, $3, $4, $5)`,
		product.ID,
		product.Name,
		product.Description,
		product.PriceMinorUnits,
		product.Currency,
	)
	if err != nil {
		return err
	}
	return nil
}
