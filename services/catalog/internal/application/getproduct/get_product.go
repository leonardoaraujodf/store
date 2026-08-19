package getproduct

import (
	"context"
	"errors"

	"github.com/leonardoaraujodf/store/services/catalog/internal/application/port"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
)

var ErrProductNotFound = errors.New("product not found")

type Command struct {
	ID string
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
	if command.ID == "" {
		return product.Product{}, product.ErrEmptyID
	}

	p, found, err := u.repository.FindByID(ctx, command.ID)
	if err != nil {
		return product.Product{}, err
	}
	if !found {
		return product.Product{}, ErrProductNotFound
	}

	return p, nil
}
