# ae-init: review-null-handling — a Manager can't be built half-empty

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-review `pr-1`, section "null handling"

## What to build

Make it impossible to create a `remotecontrol.Manager` without its runner and its way of finding `claude`. Today all four fields are exported, so a caller can leave one out and get a nil-function panic the first time `add` or `status` runs. After the change, code outside the package gets a `Manager` only through `New`, which fills in everything.

## A -> B

- **A (now):** `cmd/claude-rc/main.go` builds `&remotecontrol.Manager{UnitDir, Home, Runner, LookPath}` by hand. Forget `LookPath` and `claude-rc add` panics with a nil pointer.
- **B (after):** `main.go` calls `remotecontrol.New(unitDir, home)`. The fields are unexported, so leaving one out from outside the package doesn't compile.

## How to reproduce

**Preconditions**
- This repo on `review/1-naming` (A) or `review/2-null-handling` (B).

**Steps**
1. In `cmd/claude-rc/main.go`, delete the `LookPath: exec.LookPath,` line (A) / try writing `&remotecontrol.Manager{}` in `main.go` (B).
2. `go build ./cmd/claude-rc && ./claude-rc add -dry-run .`

**Observe**

| | A (now) | B (after) |
|---|---|---|
| Build (step 2) | Builds fine | Doesn't compile: unexported fields / use `New` |
| Run (step 2) | `panic: runtime error: invalid memory address or nil pointer dereference` | n/a |
| Normal `claude-rc add -dry-run .` | Works | Works, same output |

## Open questions

- None
