# ae-init: review-naming — plainer names in claude-rc

> **Status:** Done
> **Last updated:** 2026-10-10
> **From:** ae-review `pr-1`, section "naming and language"

## What to build

Rename the parts of `claude-rc` that the PR 1 review found hard to read. No behaviour changes: the program does exactly what it does today, with names a teammate understands at first read.

## A -> B

- **A (now):** `internal/rcservice` has `Service` (which manages systemd services, so "service" means two things), `unitFile` (unit content) next to `unitFileName` (file name), and `checkRepo` (which also returns the resolved path).
- **B (after):** the package is `internal/remotecontrol`, the type is `Manager`, and the functions are `unitFileContent` and `resolveRepo`. Same commands, same output, same tests.

## How to reproduce

**Preconditions**
- This repo on `main` (A) or on `review/1-naming` (B).

**Steps**
1. `grep -rnE "rcservice|\bService\b|unitFile\(|checkRepo" cmd internal AGENTS.md`
2. `make test && go vet ./... && make build-linux`
3. `./bin/claude-rc add -dry-run .` and `./bin/claude-rc status`

**Observe**

| | A (now) | B (after) |
|---|---|---|
| grep (step 1) | Matches in `cmd/`, `internal/rcservice/`, `AGENTS.md` | No matches |
| Checks (step 2) | Pass | Pass |
| Output (step 3) | Unit file + commands; status table | Exactly the same |

## Open questions

- None
