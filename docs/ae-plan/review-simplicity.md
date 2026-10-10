# ae-plan: review-simplicity — dry run is one swap, Manager has its own file

> **Status:** Done
> **Last updated:** 2026-10-10
> **From:** ae-init `review-simplicity`

## What's being built

A `machine` interface — `Run`, `WriteFile`, `Remove` — replacing `Runner`. `realMachine` does the work (`os/exec`, `os`); `dryRunMachine` prints `would run:`, `would write:` + content, `would delete:`. `Add`/`Remove` choose one with `m.machineFor(out, dryRun)` and then follow the same path. `Manager` + `New` move to `manager.go`.

## Constraints

- Output of `add`, `remove`, `status`, with and without `-dry-run`, stays byte-for-byte the same; existing tests are the guard.
- Types stay unexported: nothing outside the package needs them since `New` exists (PR 2).
- PR 3 of 5, stacked on `review/2-null-handling`.

## What I found in the codebase

- `internal/remotecontrol/runner.go`: `Runner`, `ExecRunner` (adds stderr to errors), `DryRunner`.
- `add.go`: `if dryRun` picks `DryRunner`, then again around `MkdirAll`/`WriteFile`, then around the "added" lines.
- `remove.go`: same pattern around `os.Remove`.
- `fake_test.go`: `fakeRunner` records `Run` calls; tests check real files in `t.TempDir()`, so the fake must still write and delete for real.
- `manager_test.go` (PR 2) checks `New` sets `ExecRunner`.

## What the docs say

- Go Code Review Comments, "Interfaces": define small interfaces where they're used; the package that uses `machine` is the one that defines it.

## Approach

Move the type first (no logic change). Then grow the runner into `machine` with tests for the new file methods. Then switch `Add`/`Remove` to it and delete the extra branches; the existing dry-run tests prove the output didn't change.

## Red / Green

- `dryRunMachine.WriteFile` prints `would write: <path>` and the content, and creates nothing.
- `dryRunMachine.Remove` prints `would delete: <path>` and deletes nothing.
- `realMachine.WriteFile` creates the missing parent folder and writes the file.
- `realMachine.Run` keeps output on error and adds stderr to the error (existing tests, renamed).
- `New` sets `realMachine`.
- All existing `Add`/`Remove` dry-run and real tests pass unchanged.

## Most important file changes

- `internal/remotecontrol/manager.go` (new): `Manager`, `New`, `machineFor`
- `internal/remotecontrol/machine.go` (renamed from `runner.go`): `machine`, `realMachine`, `dryRunMachine`
- `internal/remotecontrol/add.go`, `remove.go`: one pick instead of the branches
- `internal/remotecontrol/fake_test.go`: `fakeRunner` → `fakeMachine` (files still go to disk)

## Open questions

- None

## Subtasks

Each item is one commit, ordered outside-in. The program builds and runs after every commit.
1. Move `Manager` and `New` from `status.go` to `manager.go`.
2. `Runner` becomes `machine` with `WriteFile` and `Remove`; real and dry-run versions with tests.
3. `Add` and `Remove` pick the machine once; the extra `if dryRun` branches go.
