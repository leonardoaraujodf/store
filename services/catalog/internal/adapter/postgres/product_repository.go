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

func (r *Repository) Save(ctx context.Context, p product.Product) (product.Product, error) {
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO products(name, description, price_minor_units, currency)
		 VALUES($1, $2, $3, $4) RETURNING id`,
		p.Name,
		p.Description,
		p.PriceMinorUnits,
		p.Currency,
	).Scan(&p.ID)
	if err != nil {
		return product.Product{}, err
	}
	return p, nil
}

func (r *Repository) FindByID(ctx context.Context, id int64) (product.Product, bool, error) {
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
