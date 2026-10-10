# ae-init: ci-pr-checks — every pull request runs the checks

> **Status:** Draft
> **Last updated:** 2026-10-10

## What to build

A GitHub Actions workflow that runs on every pull request and fails the PR when something regresses: formatting, `go vet`, the tests, the Linux build, and a quick end-to-end run of the built programs. It runs on Linux (the target) and macOS (where development happens), so a test that only passes on one of them is caught.

## A -> B

- **A (now):** No `.github/` folder. Checks only run when someone remembers `go vet ./... && make test && make build-linux` locally (the "done" rule in AGENTS.md). PR #1 showed "no checks" on GitHub.
- **B (after):** Each PR shows a `checks` run for `ubuntu-latest` and `macos-latest`. It goes red if `gofmt` finds unformatted files, `go vet` complains, a test fails, the Linux build breaks, or `claude-rc add -dry-run` / `lspm` stop working.

## How to reproduce

**Preconditions**
- GitHub repo `andrelas6/utils`, Actions enabled (default for public repos).

**Steps**
1. Open any PR against `main`.
2. Look at the PR's checks section (or `gh pr checks <number>`).
3. On a scratch branch, break something on purpose (e.g. change an expected string in a test, or add badly formatted code), push, and look again.

**Observe**

| | A (now) | B (after) |
|---|---|---|
| PR checks (step 2) | "No checks" | `checks (ubuntu-latest)` and `checks (macos-latest)`, both green |
| Broken PR (step 3) | Nothing stops it | The matching step fails and the check is red |
| `gh pr checks` | No checks reported | Lists both jobs with pass/fail |

## Open questions

- None
