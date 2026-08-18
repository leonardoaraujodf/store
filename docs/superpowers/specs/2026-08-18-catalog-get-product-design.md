# Catalog GetProduct design

## Goal

Expose retrieval of one existing Catalog product through the versioned gRPC API, using the same hexagonal path and PostgreSQL ownership model as product creation.

## Scope

Add `GetProduct` to the existing `catalog.v1.CatalogService`. Its request contains only a product ID, and its response reuses the existing `Product` protobuf message with ID, name, description, price minor units, and currency.

This issue does not add product listing, pagination, categories or product-category relationships, REST, Kafka, or new runtime infrastructure.

## Driving flow

```text
gRPC client → Catalog gRPC handler → GetProduct use case → ProductRepository port → PostgreSQL adapter
```

The gRPC handler remains a driving adapter. It maps the request ID to the GetProduct use case, maps the returned domain product to the existing protobuf `Product`, and maps application or infrastructure outcomes to gRPC statuses. It does not query PostgreSQL directly.

## Repository contract

The repository lookup distinguishes three explicit outcomes:

| Repository outcome | Meaning | Application behavior |
| --- | --- | --- |
| product, `found=true`, `err=nil` | A row exists. | Return the product. |
| empty product, `found=false`, `err=nil` | No row exists; this is an expected lookup result. | Return an application-level product-not-found error. |
| empty product, `err!=nil` | PostgreSQL or another repository operation failed. | Propagate the error unchanged. |

The port therefore exposes lookup as a product value, a found flag, and an error. PostgreSQL-specific `pgx.ErrNoRows` must be translated inside the PostgreSQL adapter; it must not escape into application or gRPC code.

## Application and gRPC behavior

The GetProduct use case owns the read-path semantics. An empty ID is invalid input. A syntactically valid ID that has no matching stored product is a not-found result, not an invalid request and not an infrastructure failure.

| Situation | gRPC result |
| --- | --- |
| Request ID is empty | `codes.InvalidArgument` |
| Product exists | Successful `GetProductResponse` containing `Product` |
| Product does not exist | `codes.NotFound` |
| Repository/database failure | `codes.Internal` |

The error text returned for internal failures must remain generic and must not disclose database implementation details.

## Testing and verification

Use TDD for each behavior change. Cover:

- GetProduct use-case behavior for an existing product, a missing product, an empty ID, and a repository failure.
- PostgreSQL adapter lookup behavior for found and absent products using the existing `integration` build tag.
- gRPC handler behavior using the generated client and `bufconn` for success, `InvalidArgument`, `NotFound`, and `Internal` outcomes.
- A tagged gRPC-to-PostgreSQL integration test that creates/uses a persisted product, retrieves it through gRPC, and confirms a missing ID produces `NotFound`.

`make check` remains database-free. Existing protobuf workflow remains unchanged: run `make proto-format`, `make proto-lint`, `make proto-generate`, and `make proto-check` after contract changes. The existing Compose-backed integration commands continue to cover PostgreSQL; extend them only if their package scope no longer includes the relevant test.

## Compatibility

`GetProduct` is additive to `catalog.v1` and does not alter existing messages or field numbers. The existing `Product` protobuf message is reused rather than duplicated, preventing two diverging product representations.

## Prerequisite and branch handling

This work depends on Issue #7 / PR #8, which introduces the Catalog gRPC service and `Product` message. The Issue #9 branch begins from that branch and must be rebased onto `main` after PR #8 merges, before implementation or PR creation.
