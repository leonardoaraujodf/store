package config_test

import (
	"strings"
	"testing"

	"github.com/leonardoaraujodf/store/services/catalog/internal/config"
)

func TestLoadReturnsErrorWhenRequiredVariableIsMissing(t *testing.T) {
	t.Setenv("CATALOG_GRPC_ADDR", ":50051")
	t.Setenv("CATALOG_DATABASE_URL", "")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "CATALOG_DATABASE_URL") {
		t.Fatalf("Load() error = %v, want error naming CATALOG_DATABASE_URL", err)
	}
}

func TestLoadReturnsValidatedConfiguration(t *testing.T) {
	t.Setenv("CATALOG_GRPC_ADDR", ":50051")
	t.Setenv("CATALOG_DATABASE_URL", "postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable")

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.GRPCAddr != ":50051" || got.DatabaseURL != "postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable" {
		t.Fatalf("Load() = %#v, want configured values", got)
	}
}

func TestLoadReturnsErrorForInvalidDatabaseURL(t *testing.T) {
	t.Setenv("CATALOG_GRPC_ADDR", ":50051")
	t.Setenv("CATALOG_DATABASE_URL", "not a database url")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "CATALOG_DATABASE_URL") {
		t.Fatalf("Load() error = %v, want error naming CATALOG_DATABASE_URL", err)
	}
}
