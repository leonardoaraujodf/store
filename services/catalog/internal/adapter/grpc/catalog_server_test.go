package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	grpcadapter "github.com/leonardoaraujodf/store/services/catalog/internal/adapter/grpc"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/domain/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fakeRepository struct {
	saved   []product.Product
	saveErr error
}

func (r *fakeRepository) Save(_ context.Context, p product.Product) error {
	if r.saveErr != nil {
		return r.saveErr
	}

	r.saved = append(r.saved, p)
	return nil
}

func newCatalogClient(t *testing.T, useCase createproduct.UseCase) catalogv1.CatalogServiceClient {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(server, grpcadapter.NewCatalogServer(useCase))
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	connection, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return catalogv1.NewCatalogServiceClient(connection)
}

func TestCatalogServerCreateProductReturnsPersistedProduct(t *testing.T) {
	repository := &fakeRepository{}
	client := newCatalogClient(t, createproduct.New(repository))

	response, err := client.CreateProduct(context.Background(),
		&catalogv1.CreateProductRequest{
			Id:              "product-123",
			Name:            "Keyboard",
			Description:     "Mechanical Keyboard",
			PriceMinorUnits: 12_999,
			Currency:        "BRL",
		})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	if len(repository.saved) != 1 {
		t.Fatalf("saved = %#v, want one product", repository.saved)
	}
	if response.GetProduct().GetId() != repository.saved[0].ID {
		t.Errorf("product ID = %q, want %q", response.GetProduct().GetId(), repository.saved[0].ID)
	}
}

func TestCatalogServerCreateProductMapsDomainValidationToInvalidArgument(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{}))
	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Name:            "Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestCatalogServerCreateProductMapsRepositoryFailureToInternal(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{saveErr: errors.New("database unavailable")}))
	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Id:              "product-123",
		Name:            "Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.Internal, err)
	}
}
