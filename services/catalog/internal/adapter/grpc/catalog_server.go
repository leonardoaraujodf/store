package grpc

import (
	"context"
	"errors"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CatalogServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	useCase createproduct.UseCase
}

func NewCatalogServer(useCase createproduct.UseCase) *CatalogServer {
	return &CatalogServer{useCase: useCase}
}

func (s *CatalogServer) CreateProduct(ctx context.Context, request *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	p, err := s.useCase.Execute(ctx, createproduct.Command{
		ID:              request.GetId(),
		Name:            request.GetName(),
		Description:     request.GetDescription(),
		PriceMinorUnits: request.GetPriceMinorUnits(),
		Currency:        request.GetCurrency(),
	})
	if err != nil {
		if isProductValidationError(err) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "could not create product")
	}
	return &catalogv1.CreateProductResponse{
		Product: &catalogv1.Product{
			Id:              p.ID,
			Name:            p.Name,
			Description:     p.Description,
			PriceMinorUnits: p.PriceMinorUnits,
			Currency:        p.Currency,
		},
	}, nil
}

func isProductValidationError(err error) bool {
	return errors.Is(err, product.ErrEmptyID) ||
		errors.Is(err, product.ErrEmptyName) ||
		errors.Is(err, product.ErrNegativePrice) ||
		errors.Is(err, product.ErrInvalidCurrency)
}
