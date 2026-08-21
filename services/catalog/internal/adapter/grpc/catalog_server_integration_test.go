//go:build integration

package grpc_test

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	catalogv1 "github.com/leonardoaraujodf/store/gen/go/catalog/v1"
	grpcadapter "github.com/leonardoaraujodf/store/services/catalog/internal/adapter/grpc"
	"github.com/leonardoaraujodf/store/services/catalog/internal/adapter/postgres"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/createproduct"
	"github.com/leonardoaraujodf/store/services/catalog/internal/application/getproduct"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestCatalogServerCreateProductPersistsThroughPostgreSQL(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("CATALOG_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("CATALOG_DATABASE_URL must be set")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "TRUNCATE products"); err != nil {
		t.Fatalf("TRUNCATE products error = %v", err)
	}

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	repository := postgres.NewProductRepository(pool)
	catalogv1.RegisterCatalogServiceServer(server,
		grpcadapter.NewCatalogServer(createproduct.New(repository), getproduct.New(repository)))
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)
	connection, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	client := catalogv1.NewCatalogServiceClient(connection)

	response, err := client.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Name:            "Keyboard",
		Description:     "Mechanical keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	createdID := response.GetProduct().GetId()
	if createdID == 0 {
		t.Fatalf("response product ID = %d, want non-zero", createdID)
	}

	var persistedName string
	if err := pool.QueryRow(ctx, "SELECT name FROM products where id = $1", createdID).Scan(&persistedName); err != nil {
		t.Fatalf("QueryRow().Scan() error = %v", err)
	}
	if persistedName != "Keyboard" {
		t.Errorf("persisted name = %q, want %q", persistedName, "Keyboard")
	}

	_, err = client.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
	var totalRows int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM products").Scan(&totalRows); err != nil {
		t.Fatalf("count products error = %v", err)
	}
	if totalRows != 1 {
		t.Errorf("total persisted rows = %d, want 1 (only the valid create should persist)", totalRows)
	}
}

func TestCatalogServerGetProductPersistsThroughPostgreSQL(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("CATALOG_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("CATALOG_DATABASE_URL must be set")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "TRUNCATE products"); err != nil {
		t.Fatalf("TRUNCATE products error = %v", err)
	}

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	repository := postgres.NewProductRepository(pool)
	catalogv1.RegisterCatalogServiceServer(server,
		grpcadapter.NewCatalogServer(createproduct.New(repository), getproduct.New(repository)))
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)
	connection, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	client := catalogv1.NewCatalogServiceClient(connection)

	createResponse, err := client.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Name:            "Keyboard",
		Description:     "Mechanical keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	createdID := createResponse.GetProduct().GetId()

	response, err := client.GetProduct(ctx, &catalogv1.GetProductRequest{Id: createdID})
	if err != nil {
		t.Fatalf("GetProduct() error = %v", err)
	}
	if response.GetProduct().GetName() != "Keyboard" {
		t.Errorf("product name = %q, want %q", response.GetProduct().GetName(), "Keyboard")
	}

	_, err = client.GetProduct(ctx, &catalogv1.GetProductRequest{Id: 999_999_999})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.NotFound, err)
	}
}
