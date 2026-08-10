# Roadmap

## 1. Foundation

Initialize the Go workspace and Catalog service skeleton. Add local PostgreSQL, test conventions, local quality commands, advisory CI, protobuf tooling, and the GitHub Issue convention.

## 2. Catalog

Implement product and category management, lookup, and listing through a gRPC API backed by PostgreSQL. Practice domain modeling, application ports, gRPC adapters, repository integration tests, and API validation.

## 3. Observability

Add structured logging, OpenTelemetry tracing with Jaeger, Prometheus metrics, and Grafana dashboards to the running Catalog service.

## 4. Inventory and Events

Introduce Kafka and an Inventory service. Model inventory ownership independently and publish/consume events with idempotent processing and contract tests.

## 5. Cart and Order

Build the customer cart and order workflow. Use gRPC for required synchronous collaboration and Kafka for completed business events. Add Compose-based end-to-end coverage for the main ordering path.

## 6. Payment and Reliability

Add a simulated Payment service. Learn idempotency, retries, timeouts, failure handling, and observability across a multi-service business workflow.

## 7. Kubernetes

Containerize services and deploy to a local Kubernetes cluster. Add configuration, secrets handling, probes, resource settings, and observability manifests. Choose a managed environment only when the local workflow is understood.
