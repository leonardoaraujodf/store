# Quality and Delivery

## Initial Quality Gate

Set up the following checks in the foundation milestone and expose the same checks through local developer commands and CI:

| Check | Purpose | Initial CI policy |
| --- | --- | --- |
| `gofmt` verification | Enforce standard Go formatting | Report only |
| `go vet` | Find common correctness problems | Report only |
| `golangci-lint` | Run configurable Go linters, including Staticcheck | Report only |
| `go test ./...` | Run the full automated test suite | Report only |
| Race-enabled tests | Detect data races in CI | Report only |
| Coverage report | Show test coverage trends | Report only; no threshold initially |
| Buf format/lint | Keep protobuf source and API style consistent | Report only |
| Generated-code diff | Ensure protobuf-generated output is current | Report only |
| `govulncheck` | Identify known vulnerable Go dependencies | Report only |

“Report only” means a failure is visible in CI and must be discussed, but it does not initially prevent merging. The project will later decide which checks become required after the team understands their signal and maintenance cost.

## CI Stages

1. **Foundation:** run Go, protobuf, and dependency quality checks on every pull request and main-branch push.
2. **Container stage:** build service images and run PostgreSQL-backed integration tests after Docker Compose exists.
3. **Kubernetes stage:** validate manifests and chart/template rendering after Kubernetes deployment assets exist.
4. **Delivery stage:** build and publish images, then deploy only to an explicitly chosen development environment. Production deployment automation is deferred until a registry, cluster, secrets model, and rollback policy are selected.

## Observability Quality

The observability milestone adds structured logs, Prometheus metrics, and OpenTelemetry traces exported to Jaeger. New networked flows should make it possible to correlate an inbound request, downstream gRPC calls, and emitted Kafka events where applicable.

## Tooling Principles

- Keep CI configuration versioned with the application.
- Use the same commands locally and in CI to avoid environment-specific surprises.
- Pin tool versions once tooling is introduced; document upgrades as deliberate maintenance work.
- Do not introduce a coverage percentage target until the Catalog service establishes a meaningful baseline. Coverage informs review; behavior-focused tests provide confidence.
