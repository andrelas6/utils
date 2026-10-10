# ae-plan: review-null-handling — a Manager can't be built half-empty

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-init `review-null-handling`

## What's being built

`remotecontrol.New(unitDir, home string) *Manager`, which sets the real runner (`ExecRunner`) and `exec.LookPath`. The `Manager` fields become unexported so `New` is the only way in from outside the package.

## Constraints

- No behaviour change for the user.
- Tests stay in the package and keep setting fakes on the (now unexported) fields.
- PR 2 of 5, stacked on `review/1-naming`.

## What I found in the codebase

- `internal/remotecontrol/status.go`: `type Manager struct { UnitDir, Home string; Runner Runner; LookPath func(...) }`, all exported.
- `cmd/claude-rc/main.go`: the only place outside tests that builds a `Manager`.
- Tests build `&Manager{...}` in `add_test.go`, `remove_test.go`, `status_test.go` (same package).
- Receivers are named `s` (left over from `Service`).

## What the docs say

- Effective Go, "Constructors and composite literals": a `New` function when the zero value isn't useful.

## Approach

Add `New` with a test, switch `main.go` to it, then unexport the fields (and rename the receiver `s` → `m`) so the struct literal can't be used from outside.

## Red / Green

- `New("/u", "/h")` returns a `Manager` with that unit folder and home, `ExecRunner` as runner, and a non-nil `lookPath`.
- All existing tests pass with the unexported fields.

## Most important file changes

- `internal/remotecontrol/status.go`: `New`, unexported fields, receiver `m`
- `internal/remotecontrol/manager_test.go` (new): test for `New`
- `cmd/claude-rc/main.go`: uses `New`
- other `internal/remotecontrol/*.go`: field and receiver names

## Open questions

- None

## Subtasks

Each item is one commit, ordered outside-in. The program builds and runs after every commit.
1. `main.go` builds the manager with `remotecontrol.New`; `New` added with its test.
2. Unexport the `Manager` fields and rename the receiver to `m`.
