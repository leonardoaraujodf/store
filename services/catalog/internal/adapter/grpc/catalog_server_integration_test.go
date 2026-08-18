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
		grpcadapter.NewCatalogServer(createproduct.New(repository)))
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
		Id:              "product-123",
		Name:            "Keyboard",
		Description:     "Mechanical keyboard",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	if response.GetProduct().GetId() != "product-123" {
		t.Fatalf("response product ID = %q, want %q", response.GetProduct().GetId(), "product-123")
	}

	var persistedName string
	if err := pool.QueryRow(ctx, "SELECT name FROM products where id = $1", "product-123").Scan(&persistedName); err != nil {
		t.Fatalf("QueryRow().Scan() error = %v", err)
	}
	if persistedName != "Keyboard" {
		t.Errorf("persisted name = %q, want %q", persistedName, "Keyboard")
	}

	_, err = client.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Name:            "Invalid product",
		PriceMinorUnits: 12_999,
		Currency:        "BRL",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(error) = %s, want %s; error = %v", status.Code(err), codes.InvalidArgument, err)
	}
	var invalidRows int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM products WHERE name = $1", "Invalid product").Scan(&invalidRows); err != nil {
		t.Fatalf("count invalid products error = %v", err)
	}
	if invalidRows != 0 {
		t.Errorf("invalid persisted rows = %d, want 0", invalidRows)
	}
}
