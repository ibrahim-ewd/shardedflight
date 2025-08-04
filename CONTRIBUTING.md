# Contributing to ShardedFlight

First off, thank you for taking the time to contribute!  
ShardedFlight is released under the **GNU LGPL-3.0-or-later**; by submitting
code you agree that your contribution may be distributed under the same terms.

---

## Table of Contents

1. [Getting Started](#getting-started)  
2. [Reporting Issues](#reporting-issues)  
3. [Pull-Request Workflow](#pull-request-workflow)  
4. [Coding Guidelines](#coding-guidelines)  
5. [Commit Message Conventions](#commit-message-conventions)  
6. [Running the Test-Suite](#running-the-test-suite)  
7. [Code of Conduct](#code-of-conduct)  

---

## Getting Started

```bash
git clone https://github.com/voluminor/shardedflight.git
cd shardedflight
go 1.24              # minimum supported Go version
go test ./...
````

> **Tip:** Enable `GOFLAGS="-trimpath"` to ensure reproducible builds.

---

## Reporting Issues

* Search existing issues first — duplicates will be closed.
* Provide **Go version**, **OS/arch**, and a **minimal code snippet** that
  reproduces the problem.
* For performance regressions include benchmark results (`go test -bench`).

---

## Pull-Request Workflow

1. **Fork** the repo and create a topic branch
   `git checkout -b feature/my-awesome-idea`

2. Add tests that cover the change.

3. Ensure `go vet ./...`, `go test -race ./...`
   all pass.

4. Format code with

   ```bash
   go fmt ./...
   goimports -w .
   ```

5. Squash-and-rebase onto `main` so the branch history is clean.

6. Open the PR against `main`, fill in the template, and sign off your commit:

   ```text
   Signed-off-by: Jane Dev <jane@example.com>
   ```

   The **DCO** (Developer Certificate of Origin) is required for all commits.

7. A maintainer will review; please address review comments promptly.

8. Once approved, the PR is merged with **“Squash & Merge”**.

---

## Coding Guidelines

| Area           | Rule                                                                                  |
| -------------- | ------------------------------------------------------------------------------------- |
| **Style**      | Follow the standard Go style. Run `go fmt` and `goimports`.                           |
| **Docs**       | All exported identifiers **must** have a full-sentence doc-comment.                   |
| **Errors**     | Prefer sentinel errors in `errors.go`. Wrap external errors with `%w`.                |
| **Tests**      | Aim for ≥ 95 % coverage for new code. Use table-driven tests where sensible.          |
| **Benchmarks** | Place in `*_test.go`, name them `BenchmarkXxx`, and avoid I/O inside benchmark loops. |

---

## Commit Message Conventions

We use a lightweight variant of **Conventional Commits**:

```
<type>(scope): <subject>

<body>  # optional – wrapped at 72 chars
```

* **type**: `fix`, `feat`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`
* **scope**: package or file name (`singleflight`, `conf`)
* Use the imperative mood: “fix race condition”, **not** “fixed” or “fixes”.

---

## Running the Test-Suite

```bash
go test -race ./...          # run unit tests with the race detector
go test -bench . ./...       # run all benchmarks
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out
```

CI will fail if coverage drops below the current threshold.

---

## Code of Conduct

Be respectful and constructive. Harassment or discrimination of any kind will
not be tolerated. We follow the \[Contributor Covenant v2.1].

---

Happy hacking!

> У файлі вже згадано, що проєкт поширюється під **LGPL-3.0-or-later**, як ви й зазначили.
```
