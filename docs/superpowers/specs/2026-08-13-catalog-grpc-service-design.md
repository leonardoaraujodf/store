# Catalog gRPC service design

## Goal

Expose Catalog product creation through a runnable gRPC service. The service accepts a versioned protobuf request, delegates to the existing `CreateProduct` use case, and persists through the existing PostgreSQL repository adapter.

## Driving flow

```text
gRPC client → Catalog gRPC handler → CreateProduct use case → ProductRepository port → PostgreSQL adapter
```

The handler is a driving adapter: it maps protobuf fields to `createproduct.Command`, invokes the use case, and maps the result or error back to gRPC. It contains no product validation or SQL.

## Protobuf and generation

The contract lives at `proto/catalog/v1/catalog.proto` with package `catalog.v1` and Go package `github.com/leonardoaraujodf/store/gen/go/catalog/v1;catalogv1`.

`CatalogService` exposes exactly one RPC:

```text
rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse)
```

The request contains `id`, `name`, `description`, `price_minor_units` (`int64`), and `currency`. The response contains the created product with the same fields.

Buf v2 configuration supplies protobuf formatting, linting, and generation. Pinned remote Go and gRPC generator plugins write committed generated code below `gen/go`. Make targets run `buf format`, `buf lint`, and `buf generate`; generated-code freshness is checked by generate-then-diff.

## Configuration and direnv

Catalog configuration is an internal package that reads ordinary environment variables:

| Variable | Meaning |
| --- | --- |
| `CATALOG_GRPC_ADDR` | gRPC listener address, such as `:50051`. |
| `CATALOG_DATABASE_URL` | PostgreSQL connection URL. |

Both are mandatory. Missing values and an invalid database URL cause clear startup errors; there are no defaults.

`.envrc.example` documents local values. `.envrc` is ignored and may be loaded by direnv after `direnv allow`. Direnv is optional local shell convenience; no production Go code parses `.envrc` or depends on direnv.

## Composition root and lifecycle

`services/catalog/cmd/catalog/main.go` is the composition root. It loads configuration, connects and pings a `pgxpool.Pool`, constructs the PostgreSQL repository, creates the `CreateProduct` use case, registers the gRPC handler, and starts the listener.

On SIGINT or SIGTERM, it calls `GracefulStop` on the gRPC server and closes the database pool. If startup cannot load configuration, listen, or connect to PostgreSQL, it writes a clear error to stderr and exits nonzero.

## Error mapping

Domain errors from invalid product creation map to gRPC `codes.InvalidArgument`. Repository/database failures map to `codes.Internal`. The gRPC response returns a created product only after persistence succeeds.

## Tests and local commands

A build-tagged gRPC integration test starts PostgreSQL and migrations through Compose, starts an in-process gRPC server wired to the real PostgreSQL adapter, calls the generated client, and directly queries PostgreSQL to verify persistence. A second call with invalid input must receive `codes.InvalidArgument` and leave no row persisted.

Commands:

```text
proto-format              Format protobuf source.
proto-lint                Lint protobuf source.
proto-generate            Generate committed Go protobuf and gRPC files.
proto-check               Format, lint, generate, and verify no generated-code diff.
run-catalog               Run the Catalog gRPC service using exported environment variables.
test-grpc-integration     Start/migrate PostgreSQL and run tagged gRPC integration tests.
```

`make check` remains database-free. Container-backed integration tests and generated-code freshness remain explicit local commands in this issue; CI expansion is deferred.

## Out of scope

Product lookup/listing, categories, REST, Kafka, Dockerizing the Catalog application, observability, Kubernetes, static-analysis expansion, race tests, coverage, and merge-blocking quality rules.
