# Development Workflow

## Learning Contract

Work is collaborative. Before implementing a substantial feature, agree on its goal, boundaries, acceptance criteria, and the parts the human wants to implement personally. Boilerplate may be generated only when requested.

## GitHub Issues

Create a GitHub Issue in `leonardoaraujodf/store` for every independently testable feature. GitHub Issues are the single source of truth for planning and tracking new work.

An issue contains:

- a stable number and clear feature title;
- the problem and acceptance criteria;
- small implementation tasks in dependency order;
- test scenarios, including unit and integration coverage where relevant;
- status: `planned`, `in-progress`, `blocked`, or `done`.

A task is smaller than an issue: it is a reviewable step toward the feature, not a separate feature. Split an issue when its parts could be accepted and released independently.

## TDD Is Required

Every production behavior follows red, green, refactor:

1. Write one focused test for the next behavior.
2. Run it and confirm it fails for the expected missing behavior.
3. Write the smallest production change that makes it pass.
4. Run the focused test and relevant suite.
5. Refactor only while the suite remains green.

Do not write production behavior first and add tests afterward. A test that passes before its behavior exists is corrected before implementation continues.

## Test Levels

- **Unit tests:** domain and application use cases run quickly with fakes for ports. They assert observable business behavior, not mock implementation details.
- **Integration tests:** use real adapters at a real boundary, such as PostgreSQL repositories, Kafka consumers/producers, gRPC transport, or inter-service flows.
- **Contract tests:** add them when services exchange protobuf APIs or events that require compatibility protection.
- **End-to-end tests:** add focused Compose-based flows once multiple services form a user-visible workflow.

Prefer deterministic tests. Isolate test data, avoid time-sensitive behavior where possible, and make retries/idempotency explicit when asynchronous processing is introduced.

## Definition of Done

A feature is complete when its acceptance criteria are met, its TDD cycles have been performed, relevant unit and integration tests pass, documentation and issue status are current, and the available quality checks have been run.
