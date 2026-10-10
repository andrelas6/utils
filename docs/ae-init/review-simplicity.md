# ae-init: review-simplicity — dry run is one swap, Manager has its own file

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-review `pr-1`, section "simplicity of the solution"

## What to build

Make `claude-rc`'s dry run a single swap instead of `if dryRun` checks spread through `add` and `remove`. Everything that changes the machine (running a program, writing a file, deleting a file) goes through one small interface; a dry run just uses the version that prints instead of acting. Also move the `Manager` type and `New` out of `status.go` into their own file, where a reader would look for them.

The review's third simplicity point (move the AGENTS.md rule changes to their own PR) is dropped: those changes are already merged.

## A -> B

- **A (now):** `Add` checks `dryRun` 3 times (pick the runner, write or print the file, print "added" or not) and `Remove` 3 times. `Manager` and `New` live in `status.go`.
- **B (after):** `Add` and `Remove` each pick the real or dry-run machine once, then run the same steps either way; one `if dryRun` is left for the final "added"/"removed" line. `Manager` and `New` are in `manager.go`. Output of every command is unchanged.

## How to reproduce

**Preconditions**
- This repo on `review/2-null-handling` (A) or `review/3-simplicity` (B).

**Steps**
1. `grep -c "dryRun" internal/remotecontrol/add.go internal/remotecontrol/remove.go`
2. `grep -n "type Manager" internal/remotecontrol/*.go`
3. `make build && ./bin/claude-rc add -dry-run .`
4. `make test && go vet ./... && make build-linux`

**Observe**

| | A (now) | B (after) |
|---|---|---|
| `dryRun` mentions (step 1) | 4 in `add.go`, 4 in `remove.go` | 3 in each (parameter, one pick, one final line) |
| `type Manager` (step 2) | `status.go` | `manager.go` |
| Dry run output (step 3) | would write + unit + reminder + would run ×2 | Exactly the same |
| Checks (step 4) | Pass | Pass |

## Open questions

- None
