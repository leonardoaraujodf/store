.PHONY: test vet fmt-check check db-up migrate-up db-down db-reset test-integration

test:
	@go test -v ./...

vet:
	@go vet ./...

fmt-check:
	@test -z "$$(gofmt -l $$(git ls-files '*.go'))" || \
	(echo "Run gofmt on the files listed above."; exit 1)

check: fmt-check vet test

db-up:
	docker compose up -d --wait postgres

migrate-up:
	docker compose run --rm migrate

db-down:
	docker compose down

db-reset:
	docker compose down -v
	$(MAKE) db-up
	$(MAKE) migrate-up

test-integration: db-up migrate-up
	CATALOG_DATABASE_URL='postgres://catalog:catalog@localhost:5432/catalog?sslmode=disable' go test -tags=integration ./services/catalog/internal/adapter/postgres
