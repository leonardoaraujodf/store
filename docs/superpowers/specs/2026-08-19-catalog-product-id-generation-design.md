# Catalog product ID generation design

## Goal

Replace client-supplied product IDs with PostgreSQL-generated, auto-incrementing IDs, so product identity is guaranteed unique by the database instead of trusted from the caller.

## Problem

`CreateProductRequest.id` is currently a caller-supplied string written straight to a `TEXT PRIMARY KEY` column. Nothing generates or reserves it server-side: two callers can pick the same ID, and the only protection is the database's uniqueness constraint rejecting the second insert. As product volume grows, the chance of accidental collision grows with it. The database should own identity assignment.

## Scope

Change how a product's ID is produced and represented, end to end: domain, application ports/use cases, PostgreSQL adapter and schema, and the `catalog.v1` gRPC contract. This does not add product listing, pagination, categories, REST, Kafka, or new runtime infrastructure — it only changes how an existing product gets its ID.

## ID strategy

PostgreSQL assigns the ID via `BIGINT GENERATED ALWAYS AS IDENTITY` on insert (`RETURNING id`). The ID type is `int64` at every layer — domain, repository port, PostgreSQL column, and the gRPC contract — with no string conversion at any boundary.

Rejected alternatives:

- **App-generated UUID (v4):** globally unique and DB-independent, but random UUIDs don't sort by creation order, so `ListProducts` pagination would need a separate `created_at`/sequence column to order by. Deferred as unnecessary complexity for a single-database, single-service scope.
- **App-generated UUID v7:** solves the ordering problem UUIDv4 has, but adds a newer/less common library dependency for no benefit over letting Postgres assign a sequential integer directly.
- **int64 internally, decimal string on the wire:** future-proofs the public contract for JSON/JS clients (which lose precision above 2^53) at the cost of a parse/format step at the gRPC boundary. Not worth it while Catalog has no REST/JS client.

## Domain (`internal/domain/product`)

- `Product.ID` changes from `string` to `int64`.
- `New(name, description string, priceMinorUnits int64, currency string) (Product, error)` drops the `id` parameter. A freshly constructed `Product` has `ID: 0`, meaning "not yet persisted."
- `ErrEmptyID` is removed. No code path accepts a client-supplied ID anymore, so the invariant it protected no longer applies to construction.

## Application layer

- `createproduct.Command` drops its `ID` field.
- `port.ProductRepository.Save` changes signature from `Save(context.Context, product.Product) error` to `Save(context.Context, product.Product) (product.Product, error)`. It returns the persisted product with the database-assigned ID filled in. `createproduct.UseCase.Execute` returns whatever `Save` returns instead of the pre-save value.
- `port.ProductRepository.FindByID` takes `id int64` instead of `id string`.
- `getproduct.Command.ID` changes from `string` to `int64`. The existing "invalid input" guard moves from reusing `product.ErrEmptyID` to a new `getproduct.ErrInvalidID` (`"product id must be positive"`), returned when `command.ID <= 0`. This lives in `getproduct`, not `product`, because it is a query-key precondition, not an entity-construction invariant.

## PostgreSQL adapter and migration

- `Save` becomes:
  ```sql
  INSERT INTO products(name, description, price_minor_units, currency)
  VALUES ($1, $2, $3, $4)
  RETURNING id
  ```
  scanning the returned `id` into a copy of the input product before returning it.
- `FindByID(ctx, id int64)` uses the same query shape as today with an `int64` parameter.
- New migration `000002_product_id_autoincrement.{up,down}.sql` (the existing `000001` migration is not edited, per normal migration hygiene):
  - **up:** `ALTER TABLE products DROP COLUMN id;` then `ALTER TABLE products ADD COLUMN id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY;`
  - **down:** reverse to the original shape — drop the identity `id` column and re-add `id TEXT PRIMARY KEY`.
  - This is destructive to any existing rows' IDs (a dropped column can't repopulate meaningfully). There is no production data; locally this means running `make db-reset` after pulling the migration, which is already the documented "wipe and reapply" workflow.

## Proto contract (`proto/catalog/v1/catalog.proto`)

- `CreateProductRequest` loses its `id` field. The field number is not reused: `reserved 1; reserved "id";`.
- `GetProductRequest.id` and `Product.id` change type from `string` to `int64`.

**Deliberate exception to the "keep protobuf contracts backward compatible" project rule:** removing a request field and changing a field's wire type (length-delimited string → varint int64) are both wire-breaking changes, not additive ones. This is accepted as a one-time, agreed exception because `catalog.v1` has no external or production consumers yet — Catalog is the only service, and the project is pre-release. There is nothing deployed to break. Future contract changes still follow the additive-preferred rule; this is not a precedent for skipping compatibility planning once real consumers exist.

## Testing and verification

Apply TDD per changed behavior, not a silent patch of existing tests:

- Domain: `product.New` tests drop ID-related cases; `ErrEmptyID` assertions are removed.
- `createproduct` use case: tests assert the returned product carries the ID `Save` provides, using a fake repository.
- `getproduct` use case: tests cover `ID <= 0` → `ErrInvalidID`, in addition to existing found/not-found/repository-failure cases, with `int64` command IDs.
- PostgreSQL adapter integration tests (`integration` build tag): assert `Save` returns a product with a nonzero, database-assigned ID, and that sequential inserts get distinct IDs; `FindByID` takes an `int64`.
- gRPC handler tests (`bufconn`) and the gRPC-to-PostgreSQL integration test: `CreateProductRequest` without an `id` field; `GetProductRequest.id` as `int64`; `getproduct.ErrInvalidID` maps to `codes.InvalidArgument` in place of the old `product.ErrEmptyID` mapping.

`make check` remains database-free. After the contract change, run `make proto-format`, `make proto-lint`, `make proto-generate`, and `make proto-check`. Existing Compose-backed integration commands (`make test-integration`, `make test-grpc-integration`) continue to cover PostgreSQL against the new schema.

## Compatibility

Additive within `catalog.v1` otherwise: no other messages or RPCs change shape. See the Proto contract section above for the one deliberate, agreed exception to the backward-compatibility rule, scoped to the `id` field/type change.
