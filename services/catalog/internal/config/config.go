package config

import (
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	GRPCAddr    string
	DatabaseURL string
}

func Load() (Config, error) {
	grpcAddr := os.Getenv("CATALOG_GRPC_ADDR")
	if grpcAddr == "" {
		return Config{}, fmt.Errorf("CATALOG_GRPC_ADDR must be set")
	}
	databaseURL := os.Getenv("CATALOG_DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("CATALOG_DATABASE_URL must be set")
	}
	parsed, err := url.ParseRequestURI(databaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return Config{}, fmt.Errorf("CATALOG_DATABASE_URL must be a PostgreSQL connection URL")
	}
	return Config{GRPCAddr: grpcAddr, DatabaseURL: databaseURL}, nil
}
