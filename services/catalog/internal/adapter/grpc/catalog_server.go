package grpc

import (
	"context"
	"errors"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CatalogServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	createUseCase createproduct.UseCase
	getUseCase    getproduct.UseCase
}

func NewCatalogServer(createUseCase createproduct.UseCase, getUseCase getproduct.UseCase) *CatalogServer {
	return &CatalogServer{createUseCase: createUseCase, getUseCase: getUseCase}
}

func (s *CatalogServer) CreateProduct(ctx context.Context, request *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	p, err := s.createUseCase.Execute(ctx, createproduct.Command{
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
		Product: toProto(p),
	}, nil
}

func (s *CatalogServer) GetProduct(ctx context.Context, request *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	p, err := s.getUseCase.Execute(ctx, getproduct.Command{ID: request.GetId()})
	if err != nil {
		if errors.Is(err, getproduct.ErrInvalidID) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, getproduct.ErrProductNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "could not get product")
	}
	return &catalogv1.GetProductResponse{
		Product: toProto(p),
	}, nil
}

func toProto(p product.Product) *catalogv1.Product {
	return &catalogv1.Product{
		Id:              p.ID,
		Name:            p.Name,
		Description:     p.Description,
		PriceMinorUnits: p.PriceMinorUnits,
		Currency:        p.Currency,
	}
}

func isProductValidationError(err error) bool {
	return errors.Is(err, product.ErrEmptyName) ||
		errors.Is(err, product.ErrNegativePrice) ||
		errors.Is(err, product.ErrInvalidCurrency)
}
