# Catalog ListProducts design

## Goal

Expose paginated listing of existing Catalog products through the versioned gRPC API, using the same hexagonal path and PostgreSQL ownership model as product creation and retrieval.

## Scope

Add `ListProducts` to the existing `catalog.v1.CatalogService`. Its request carries a page size and a row offset; its response returns a page of the existing `Product` protobuf message plus a flag signaling whether further pages exist.

This issue does not add filtering, sorting, a total-count field, categories or product-category relationships, REST, Kafka, or new runtime infrastructure.

## Driving flow

```text
gRPC client → Catalog gRPC handler → ListProducts use case → ProductRepository port → PostgreSQL adapter
```

The gRPC handler remains a driving adapter. It maps the request's page size and offset to the ListProducts use case, maps the returned domain products to the existing protobuf `Product` message, and maps application or infrastructure outcomes to gRPC statuses. It does not query PostgreSQL directly.

## API shape

```protobuf
rpc ListProducts(ListProductsRequest) returns (ListProductsResponse);

message ListProductsRequest {
  int32 page_size = 1;
  int32 offset = 2;
}

message ListProductsResponse {
  repeated Product products = 1;
  bool has_more = 2;
}
```

`page_size = 0` means "use the default page size." `offset` is a plain row offset (not a page number), matching the offset/limit pagination style. There is no total-count field; `has_more` is the only paging signal, avoiding an extra `COUNT(*)` query per call.

## Application behavior

The ListProducts use case owns page-size defaulting, validation, and the has-more determination — none of this is PostgreSQL-specific:

1. `page_size == 0` → substitute the default page size (20).
2. `page_size < 0` or `page_size > 100` (the max) → invalid page size.
3. `offset < 0` → invalid offset.
4. Ask the repository for `page_size + 1` rows starting at `offset`. If the repository returns `page_size + 1` rows, set `has_more = true` and drop the extra row before returning the page; otherwise `has_more = false`.

Invalid input is rejected outright (no silent clamping), consistent with how `CreateProduct`/`GetProduct` already reject invalid input rather than correcting it.

| Situation | gRPC result |
| --- | --- |
| `page_size` negative or greater than 100 | `codes.InvalidArgument` |
| `offset` negative | `codes.InvalidArgument` |
| Valid request, zero or more products | Successful `ListProductsResponse` containing the page and `has_more` |
| Repository/database failure | `codes.Internal` |

The error text returned for internal failures must remain generic and must not disclose database implementation details.

## Repository contract

The `ProductRepository` port gains a listing method that returns a page of products ordered by ID, with no distinction between "empty result" and "not found" (an empty catalog is a valid, successful result, unlike a single missing ID in `GetProduct`). The PostgreSQL adapter fetches rows with `ORDER BY id LIMIT $1 OFFSET $2`; the "fetch one extra row" trick used to compute `has_more` is application logic, not part of this SQL, so it stays testable with a fake repository independent of PostgreSQL.

## Testing and verification

Use TDD for each behavior change. Cover:

- ListProducts use-case behavior: default page size substitution, invalid page size (negative and over-max), invalid offset (negative), `has_more` true/false with correct trimming, and repository failure propagation — using a fake repository.
- PostgreSQL adapter listing behavior for an empty table, a single page, and a page boundary (using the existing `integration` build tag).
- gRPC handler behavior using the generated client and `bufconn` for a success case (asserting product mapping and `has_more`), `InvalidArgument`, and `Internal` outcomes.
- A tagged gRPC-to-PostgreSQL integration test that creates several products and lists them across two pages through gRPC, confirming ordering, `has_more`, and pagination boundaries end to end.

`make check` remains database-free. Existing protobuf workflow remains unchanged: run `make proto-format`, `make proto-lint`, `make proto-generate`, and `make proto-check` after contract changes. The existing Compose-backed integration commands continue to cover PostgreSQL; extend them only if their package scope no longer includes the relevant test.

## Compatibility

`ListProducts` is additive to `catalog.v1` and does not alter existing messages or field numbers. The existing `Product` protobuf message is reused rather than duplicated, preventing two diverging product representations.
