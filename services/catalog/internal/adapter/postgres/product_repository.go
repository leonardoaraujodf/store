package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) FindByID(ctx context.Context, id string) (product.Product, bool, error) {
	var p product.Product
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, name, description, price_minor_units, currency
		 FROM products
		 WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.PriceMinorUnits, &p.Currency)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return product.Product{}, false, nil
		}
		return product.Product{}, false, err
	}
	return p, true, nil
}
