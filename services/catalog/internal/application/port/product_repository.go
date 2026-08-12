package port

import (
	"context"

	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type ProductRepository interface {
	Save(context.Context, product.Product) error
}
