# Foundation Quality Checks and Advisory CI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provide one repeatable local Go quality command and an advisory GitHub Actions workflow that executes it.

**Architecture:** A root Makefile defines the developer-facing quality contract. `check` composes the named Go checks, and the workflow delegates to `make check` instead of repeating those commands in YAML.

**Tech Stack:** GNU Make, Go 1.26.5, GitHub Actions.

## Global Constraints

- Work only in the `issue_3` worktree: `/home/leonardo/Development/store/.worktrees/issue_3`.
- The Makefile uses `go` from `PATH`; Go 1.26.5 is declared by `go.mod`.
- CI must remain advisory: do not configure branch protection or required checks.
- Do not add Docker Compose, PostgreSQL, Buf/protobuf, gRPC, `golangci-lint`, `govulncheck`, race tests, coverage, Kafka, or Kubernetes.
- Verify behavior with commands; this tooling change does not require application-code TDD.

---

## File structure

| Path | Responsibility |
| --- | --- |
| `Makefile` | Local formatting, vet, test, and aggregate quality commands. |
| `.github/workflows/ci.yml` | Push/PR workflow that runs the local aggregate command. |
| `README.md` | Current project state, Go prerequisite, and local command reference. |

## Task 1: Define local Makefile commands

**Files:**
- Create: `Makefile`

**Interfaces:**
- Produces: `make fmt-check`, `make vet`, `make test`, and `make check`.
- Produces: `check` that runs `fmt-check`, then `vet`, then `test`.

- [X] **Step 1: Confirm Make is available and capture the empty-target baseline**

Run:

```bash
make --version
make test
```

Expected: `make --version` reports an installed Make implementation; `make test` fails because no Makefile or `test` target exists.

- [X] **Step 2: Create the Makefile**

```make
.PHONY: fmt-check vet test check

fmt-check:
	@test -z "$$(gofmt -l $$(git ls-files '*.go'))" || \
		(echo "Run gofmt on the files listed above."; exit 1)

vet:
	go vet ./...

test:
	go test ./...

check: fmt-check vet test
```

The command substitution is escaped as `$$` because Make must pass a literal `$` to the shell. `git ls-files '*.go'` limits the check to tracked Go source, while `gofmt -l` prints every unformatted file and produces no output when formatting is clean.

- [X] **Step 3: Verify each target individually**

Run:

```bash
make fmt-check
make vet
make test
```

Expected: each exits 0. `fmt-check` produces no output on a formatted working tree; `vet` and `test` delegate to Go.

- [X] **Step 4: Verify aggregate order and success**

Run:

```bash
make check
```

Expected: output shows `go vet ./...` followed by `go test ./...`; the command exits 0. `fmt-check` runs first and remains quiet when successful.

- [X] **Step 5: Prove fmt-check fails for unformatted tracked source, then restore the tree**

Run:

```bash
printf 'package temporary\nfunc x(){}\n' > /tmp/store_fmt_check_probe.go
cp /tmp/store_fmt_check_probe.go services/catalog/internal/domain/product/fmt_check_probe.go
git add -N services/catalog/internal/domain/product/fmt_check_probe.go
make fmt-check
git restore --staged services/catalog/internal/domain/product/fmt_check_probe.go
rm services/catalog/internal/domain/product/fmt_check_probe.go /tmp/store_fmt_check_probe.go
make fmt-check
```

Expected: the first `make fmt-check` fails with its formatting-remediation message; after removing the intent-to-add index entry and file, the final command exits 0. Do not commit the temporary probe file.

- [X] **Step 6: Commit the local quality contract**

Run:

```bash
git add Makefile
git commit -m "build: add local Go quality checks"
```

Expected: a commit containing only the Makefile.

## Task 2: Run the same local contract in advisory CI

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: the `make check` target from Task 1.
- Produces: a GitHub Actions workflow triggered by pushes and pull requests.

- [X] **Step 1: Create the workflow**

```yaml
name: CI

on:
  push:
  pull_request:

jobs:
  quality:
    name: Go quality checks
    runs-on: ubuntu-latest
    steps:
      - name: Check out repository
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Run local quality contract
        run: make check
```

- [X] **Step 2: Inspect that CI does not duplicate quality commands**

Run:

```bash
rg -n 'make check|go vet|go test|gofmt' .github/workflows/ci.yml
```

Expected: the output contains `make check` and no standalone `go vet`, `go test`, or `gofmt` commands.

- [X] **Step 3: Run the same command CI will run**

Run:

```bash
make check
```

Expected: exit 0. This validates the workflow's command contract locally; GitHub Actions executes it after the branch is pushed.

- [X] **Step 4: Commit the advisory workflow**

Run:

```bash
git add .github/workflows/ci.yml
git commit -m "ci: run Go quality checks"
```

Expected: a commit containing only the workflow.

## Task 3: Document current state and local checks

**Files:**
- Modify: `README.md`

**Interfaces:**
- Documents: Go 1.26.5 and the Makefile commands from Task 1.

- [X] **Step 1: Replace the outdated Current State paragraph**

Replace the paragraph below `## Current State` with:

```markdown
The project is in **Milestone 1: Foundation**. Catalog has a transport-free product domain and `CreateProduct` application use case, both covered by focused unit tests. PostgreSQL, gRPC, and other infrastructure are not yet introduced.
```

- [X] **Step 2: Add a Local quality checks section after Current State**

```markdown
## Local quality checks

Go **1.26.5** is required. Run the complete local quality contract with:

```bash
make check
```

Individual commands are also available:

```bash
make fmt-check
make vet
make test
```
```

- [X] **Step 3: Verify documentation and the final quality contract**

Run:

```bash
rg -n 'Milestone 1|CreateProduct|Local quality checks|make check|Go \*\*1\.26\.5\*\*' README.md
make check
git diff --check
```

Expected: README search finds every documented concept; `make check` and whitespace validation exit 0.

- [X] **Step 4: Commit documentation and verify history**

Run:

```bash
git add README.md
git commit -m "docs: describe local quality checks"
git status --short --branch
```

Expected: the worktree is clean and shows `issue_3` ahead of `origin/main` by the Issue #3 commits.
