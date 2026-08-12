# Catalog foundation design

## Goal

Prepare the repository for the first Catalog capability without adding transport, persistence, or other infrastructure. The work should make the hexagonal boundaries visible as the first product-creation behavior is developed through TDD.

## Module

The repository is one Go module:

```text
github.com/leonardoaraujodf/store
```

Go is installed locally by the learner before module initialization. The project should use a current stable Go release selected by the learner; its exact version is recorded in `go.mod`.

## Initial structure

Create directories only together with the first source or test file that needs their package. The intended Catalog layout is:

```text
services/catalog/internal/
  domain/product/
  application/createproduct/
  application/port/
```

`internal` prevents other services from importing Catalog implementation packages directly. Each future service will own an equivalent internal tree and its own domain model.

## Boundaries

- `domain/product` contains `Product`, validation, and domain errors. It has no knowledge of repositories, gRPC, PostgreSQL, Kafka, or process startup.
- `application/createproduct` contains the use case that accepts valid creation input and coordinates saving a product.
- `application/port` contains the repository interface required by the use case. It is an application dependency, not a database implementation.

No `cmd`, gRPC, PostgreSQL, Kafka, container, or Kubernetes package is introduced in this issue. A repository fake used by application unit tests may live in a test file because it exists only to exercise the port.

## Implementation sequence

1. Install Go and verify the toolchain.
2. Initialize the root Go module and run the empty test command.
3. Begin the product domain task with a focused failing unit test; create the domain package only as part of that test cycle.
4. Add the repository port and `CreateProduct` use case through further focused TDD cycles.

## Testing and errors

Every behavior follows red, green, refactor. Domain tests verify product invariants and errors. Application tests use an in-memory fake repository to prove valid products are saved, invalid input is not saved, and repository errors reach the caller.

## Acceptance boundary

This design fulfills only Issue #1: an in-process, transport-free product-creation capability. Persistence, network APIs, categories, listing, updates, events, and runtime infrastructure require later issues.
