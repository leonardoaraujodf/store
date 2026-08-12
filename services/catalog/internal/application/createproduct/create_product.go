package createproduct

import (
	"context"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/port"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

type Command struct {
	ID, Name, Description string
	PriceMinorUnits       int
	Currency              string
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
	p, err := product.New(command.ID,
		command.Name,
		command.Description,
		command.PriceMinorUnits,
		command.Currency)
	if err != nil {
		return product.Product{}, err
	}
	if err := u.repository.Save(ctx, p); err != nil {
		return product.Product{}, err
	}
	return p, nil
}
