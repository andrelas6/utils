# ae-plan: review-naming — plainer names in claude-rc

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-init `review-naming`

## What's being built

Pure renames in `claude-rc`: `unitFile` → `unitFileContent`, `checkRepo` → `resolveRepo`, `Service` → `Manager`, package `internal/rcservice` → `internal/remotecontrol`.

## Constraints

- No behaviour change. Output of every command stays the same.
- **TDD does not fit:** nothing new to test. The compiler and the existing tests are the check: they must pass unchanged except for the names.
- First PR of a stack of five review follow-ups; later PRs build on these names.

## What I found in the codebase

- `internal/rcservice/unit.go`: `unitFile` and `unitFileName` side by side.
- `internal/rcservice/add.go`: `checkRepo` returns the absolute path that `Add` keeps using.
- `internal/rcservice/status.go`: `type Service struct`, used by `cmd/claude-rc/main.go` and every test.
- `AGENTS.md` names `internal/rcservice` in the `claude-rc` note.
- `docs/ae-*/claude-rc.md` also say `rcservice`; they describe PR 1 as it was built, so they stay as they are.

## What the docs say

- Effective Go, "Package names": short, lower case, single word, no underscores or mixedCaps — `remotecontrol` fits.

## Approach

Rename in three small commits, smallest first, running `go vet` and the tests after each. Use `git mv` for the package folder so history follows the files.

## Red / Green

- All existing tests pass with the new names.
- `claude-rc add -dry-run .` prints the same unit file and commands as before.

## Most important file changes

- `internal/rcservice/*` → `internal/remotecontrol/*` (moved)
- `cmd/claude-rc/main.go`: import and type name
- `AGENTS.md`: package name in the `claude-rc` note

## Open questions

- None

## Subtasks

Each item is one commit, ordered outside-in. The program builds and runs after every commit.
1. Rename `unitFile` → `unitFileContent` and `checkRepo` → `resolveRepo`.
2. Rename `Service` → `Manager` (struct, receivers, tests, `main.go`).
3. Move `internal/rcservice` → `internal/remotecontrol`; update the import and AGENTS.md.
