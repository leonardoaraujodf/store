package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"

	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	grpcadapter "github.com/leonardoaraujodf/store/services/catalog/internal/adapter/grpc"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
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
	findErr error
}

func (r *fakeRepository) Save(_ context.Context, p product.Product) (product.Product, error) {
	if r.saveErr != nil {
		return product.Product{}, r.saveErr
	}

	p.ID = int64(len(r.saved)) + 1
	r.saved = append(r.saved, p)
	return p, nil
}

func (r *fakeRepository) FindByID(_ context.Context, id int64) (product.Product, bool, error) {
	if r.findErr != nil {
		return product.Product{}, false, r.findErr
	}
	for _, p := range r.saved {
		if p.ID == id {
			return p, true, nil
		}
	}
	return product.Product{}, false, nil
}

func newCatalogClient(t *testing.T, createUseCase createproduct.UseCase, getUseCase getproduct.UseCase) catalogv1.CatalogServiceClient {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(server, grpcadapter.NewCatalogServer(createUseCase, getUseCase))
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
	client := newCatalogClient(t, createproduct.New(repository), getproduct.New(repository))

	response, err := client.CreateProduct(context.Background(),
		&catalogv1.CreateProductRequest{
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
		t.Errorf("product ID = %d, want %d", response.GetProduct().GetId(), repository.saved[0].ID)
	}
	if response.GetProduct().GetId() == 0 {
		t.Errorf("product ID = %d, want non-zero", response.GetProduct().GetId())
	}
}

func TestCatalogServerCreateProductMapsDomainValidationToInvalidArgument(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{}), getproduct.New(&fakeRepository{}))
	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestCatalogServerCreateProductMapsRepositoryFailureToInternal(t *testing.T) {
	repository := &fakeRepository{saveErr: errors.New("database unavailable")}
	client := newCatalogClient(t, createproduct.New(repository), getproduct.New(repository))
	_, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Name:            "Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.Internal, err)
	}
}

func TestCatalogServerGetProductReturnsPersistedProduct(t *testing.T) {
	repository := &fakeRepository{}
	client := newCatalogClient(t, createproduct.New(repository), getproduct.New(repository))

	createResponse, err := client.CreateProduct(context.Background(), &catalogv1.CreateProductRequest{
		Name:            "Keyboard",
		Description:     "Mechanical Keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	createdID := createResponse.GetProduct().GetId()

	response, err := client.GetProduct(context.Background(), &catalogv1.GetProductRequest{Id: createdID})
	if err != nil {
		t.Fatalf("GetProduct() error = %v", err)
	}
	if response.GetProduct().GetId() != createdID {
		t.Errorf("product ID = %d, want %d", response.GetProduct().GetId(), createdID)
	}
	if response.GetProduct().GetName() != "Keyboard" {
		t.Errorf("product name = %q, want %q", response.GetProduct().GetName(), "Keyboard")
	}
	if response.GetProduct().GetDescription() != "Mechanical Keyboard" {
		t.Errorf("product description = %q, want %q", response.GetProduct().GetDescription(), "Mechanical Keyboard")
	}
	if response.GetProduct().GetPriceMinorUnits() != 12_999 {
		t.Errorf("product price minor units = %d, want %d", response.GetProduct().GetPriceMinorUnits(), 12_999)
	}
	if response.GetProduct().GetCurrency() != "BRL" {
		t.Errorf("product currency = %q, want %q", response.GetProduct().GetCurrency(), "BRL")
	}
}

func TestCatalogServerGetProductMapsNonPositiveIDToInvalidArgument(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{}), getproduct.New(&fakeRepository{}))
	_, err := client.GetProduct(context.Background(), &catalogv1.GetProductRequest{Id: 0})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestCatalogServerGetProductMapsMissingProductToNotFound(t *testing.T) {
	client := newCatalogClient(t, createproduct.New(&fakeRepository{}), getproduct.New(&fakeRepository{}))
	_, err := client.GetProduct(context.Background(), &catalogv1.GetProductRequest{Id: 999_999_999})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.NotFound, err)
	}
}

func TestCatalogServerGetProductMapsRepositoryFailureToInternal(t *testing.T) {
	repository := &fakeRepository{findErr: errors.New("database unavailable")}
	client := newCatalogClient(t, createproduct.New(repository), getproduct.New(repository))
	_, err := client.GetProduct(context.Background(), &catalogv1.GetProductRequest{Id: 1})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.Internal, err)
	}
}
