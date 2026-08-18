# Online Store Backend

## Purpose

This is a learning-by-doing backend for an Amazon-like online store. It uses Go, hexagonal architecture, microservices, gRPC, Kubernetes, Kafka, and observability tooling. Work progresses in small milestones so concepts are learned before boilerplate is automated.

## Working Agreement

- The human implements concept-heavy work when they want to learn it. Provide explanations, review, and requested boilerplate.
- Plan documents must not embed full code implementations. State each task's goal, exact files/interfaces/signatures touched, constraints, and test/verification steps precisely enough to be actionable, but leave the code itself out. **This overrides any skill's own default of writing full code into every plan step** (e.g. `superpowers:writing-plans` defaults to embedding runnable code per step — do not follow that default in this repo).
- The agent executing a plan task proposes the actual code as a real change to the file(s), which is then reviewed — not written into the plan document itself. Prefer this even for tasks delegated for full implementation: delegation decides who writes the code (human vs. executing agent), not whether the plan document contains it.
- The human may explicitly delegate a task or issue for full implementation once they are comfortable with its concepts; then carry out the agreed work and present the resulting change for review. Give conceptual hints before showing a solution when the human is stuck.
- Plan a feature before implementation. Track each independently testable feature in a GitHub Issue in `leonardoaraujodf/store`.
- One issue may contain several small tasks. Keep issues focused on a single vertical capability.
- Preserve existing user changes. Do not add services, dependencies, infrastructure, or external integrations beyond the active issue without agreement.

## Architecture Rules

- Introduce services incrementally: Catalog, Inventory, Cart, Order, Payment, then Identity only if required.
- Each service owns its business logic and data. Do not share databases or domain models across services.
- Use hexagonal boundaries: domain and application code depend on ports; gRPC, PostgreSQL, Kafka, and observability are adapters.
- Use gRPC for synchronous service APIs and Kafka for asynchronous domain events. Keep protobuf contracts backward compatible.
- PostgreSQL is the initial datastore. Introduce Redis only for a measured, documented need.

## Quality Rules

- Apply TDD to every behavior change: write one failing test, verify the expected failure, write the minimum code to pass, then refactor with tests green.
- Add unit tests for domain/application behavior. Add integration tests whenever a real adapter, database, broker, or network boundary is introduced.
- Run formatting, static analysis, tests, and generated-code checks before considering a feature complete.
- Early CI reports checks without blocking merges while the tooling is being learned. Tighten enforcement only by an explicit project decision.

## Documentation

- Read `docs/architecture.md` for the system model and milestone boundaries.
- Read `docs/development-workflow.md` for TDD, testing, and issue conventions.
- Read `docs/quality-and-delivery.md` for CI/CD and quality-gate strategy.
- Read `docs/roadmap.md` for the delivery order.
