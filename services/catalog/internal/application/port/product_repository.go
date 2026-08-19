package port

import (
	"context"

	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type ProductRepository interface {
	Save(context.Context, product.Product) (product.Product, error)
	FindByID(ctx context.Context, id int64) (product.Product, bool, error)
}
