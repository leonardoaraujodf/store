package createproduct

import (
	"context"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/port"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type Command struct {
	Name, Description string
	PriceMinorUnits   int64
	Currency          string
}

type UseCase struct {
	repository port.ProductRepository
}

func New(repository port.ProductRepository) UseCase {
	return UseCase{
		repository: repository,
	}
}

func (u UseCase) Execute(ctx context.Context, command Command) (product.Product, error) {
	p, err := product.New(command.Name,
		command.Description,
		command.PriceMinorUnits,
		command.Currency)
	if err != nil {
		return product.Product{}, err
	}
	saved, err := u.repository.Save(ctx, p)
	if err != nil {
		return product.Product{}, err
	}
	return saved, nil
}
