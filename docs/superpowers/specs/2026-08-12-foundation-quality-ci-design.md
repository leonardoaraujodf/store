# Foundation quality checks and advisory CI design

## Goal

Give the project one small, repeatable quality loop that developers run locally and GitHub Actions runs unchanged. The scope is limited to Go formatting verification, static vetting, and tests.

## Local command interface

Add a root `Makefile` with these targets:

```text
fmt-check  Verify that tracked Go files are formatted; do not rewrite files.
vet        Run `go vet ./...`.
test       Run `go test ./...`.
check      Run fmt-check, vet, and test in that order.
```

`check` is the one complete local quality command. It composes the named commands rather than duplicating their shell instructions.

The Makefile invokes `go` from the developer's `PATH`. Go 1.26.5 is required, as declared in `go.mod`.

## CI design

Add a GitHub Actions workflow triggered by pushes and pull requests. It checks out the repository, installs Go 1.26.5, and executes exactly `make check`.

The workflow reports failures normally. The repository does not add branch protection or required-check rules in this issue, so the checks remain advisory rather than merge-blocking.

## Documentation

Update the README to state that the Catalog product domain and create-product application use case now exist. Add a concise local-development section with the Go requirement and commands:

```text
make fmt-check
make vet
make test
make check
```

## Verification

This work uses command-based verification instead of unit-test TDD:

1. Confirm each target runs its expected Go command.
2. Confirm `make check` runs all three checks and succeeds on the clean repository.
3. Inspect the workflow so it calls `make check`, with no duplicate Go quality commands.
4. Run `make check` after all documentation and workflow changes.

## Out of scope

This issue does not add Docker Compose, PostgreSQL, protobuf/Buf, gRPC, `golangci-lint`, `govulncheck`, race tests, coverage reporting, Kafka, Kubernetes, or merge-blocking rules.
