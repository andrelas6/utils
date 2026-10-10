# ae-plan: ci-pr-checks — every pull request runs the checks

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-init `ci-pr-checks`

## What's being built

`.github/workflows/checks.yml`: on `pull_request`, a job on `ubuntu-latest` and `macos-latest` that runs, in order:
1. `gofmt -l .` must print nothing
2. `go vet ./...`
3. `make test`
4. `make build-linux`
5. smoke test: `make build`, a stub `claude` on `PATH`, `claude-rc add -dry-run <temp repo>` must print a unit with that stub's path, and `lspm .` must run

## Constraints

- Mirrors the "before declaring work done" rule in AGENTS.md, plus `gofmt`, plus the smoke test.
- Read-only token (`permissions: contents: read`); the workflow doesn't need to write anything.
- No real `systemctl` or `claude`: the runner has neither a user systemd session nor a Claude login, so only `-dry-run` is exercised end to end.
- Runners run as a normal user, so the `chmod 000` tests (PR #5) work.
- **TDD does not fit** a workflow file. How we know it works: the PR that adds it runs it (green), and a throwaway commit that breaks a test makes it red.

## What I found in the codebase

- No `.github/` folder yet.
- `go.mod`: `go 1.26.4` — `setup-go` can read the version from it.
- `Makefile`: `test` (`go test ./...`), `build` (into `bin/`), `build-linux` (into `bin/linux/`).
- `internal/remotecontrol` tests use `t.TempDir()` and `sh`; nothing machine-specific.
- `claude-rc add` needs `claude` on `PATH` (`exec.LookPath`) even in a dry run.

## What the docs say

- GitHub Actions workflow syntax: `on: pull_request` (default types opened, synchronize, reopened); `strategy.matrix.os`; `permissions`; `concurrency` with `cancel-in-progress` to drop outdated runs of the same PR.
- `actions/checkout` latest release v7.0.1 (2026-07-20); `actions/setup-go` latest v7.0.0 (2026-07-16), `go-version-file: go.mod`, module cache on by default.

## Approach

One workflow file, one job, OS matrix, fail-fast off so both OSes always report. The smoke test is a few shell lines in the workflow using `$RUNNER_TEMP`.

## Red / Green

- The PR adding the workflow shows two green checks.
- A throwaway commit with a failing test turns the `make test` step red (checked once, then dropped).

## Most important file changes

- `.github/workflows/checks.yml` (new)
- `AGENTS.md`: note that PRs run these checks
- `README.md`: one line about CI

## Open questions

- None

## Subtasks

Each item is one commit, ordered outside-in.
1. Add `.github/workflows/checks.yml` with format, vet, test, Linux build and smoke test on Ubuntu and macOS.
2. Mention the checks in AGENTS.md and README.
