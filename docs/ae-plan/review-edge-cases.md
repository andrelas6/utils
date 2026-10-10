# ae-plan: review-edge-cases — claude-rc add refuses paths that would break the unit

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-init `review-edge-cases`

## What's being built

Three more refusals in `claude-rc add` and one stronger test:
- repo path with `\n` or `\r` → error
- `claude` path with a space, tab, `\n` or `\r` → error
- home check compares `filepath.EvalSymlinks` of both sides
- "missing folder" test asserts `errors.Is(err, os.ErrNotExist)`

## Constraints

- Refuse rather than quote: systemd's `ExecStart` quoting rules are easy to get subtly wrong, and a space in the path to `claude` is rare. The error says what to do.
- `resolveRepo` keeps returning the absolute path as typed (not the symlink target), so `WorkingDirectory=` and `status` show what the user gave.
- PR 5 of 5, stacked on `review/4-error-handling`.

## What I found in the codebase

- `internal/remotecontrol/add.go`: `resolveRepo` (abs, stat, is-folder, `abs == filepath.Clean(m.home)`), then `m.lookPath("claude")`.
- `internal/remotecontrol/unit.go`: `unitFileContent` writes paths straight into the unit (only `%` is escaped).
- `internal/remotecontrol/add_test.go`: `TestAddRefuses` table; each case checks no folder created and no systemctl calls.

## What the docs say

- systemd.syntax(7) / systemd.service(5): unit files are line-based; `ExecStart=` splits the command line on whitespace unless quoted.
- `path/filepath.EvalSymlinks`: returns the path after following all symlinks; fails if the path doesn't exist (both sides exist here: the repo was just `Stat`-ed, and `$HOME` exists).

## Approach

One Red/Green cycle per refusal, as new rows in `TestAddRefuses`, then tighten the existing missing-folder row.

## Red / Green

- `add` with a repo folder whose name has `\n` → error mentions "line break"; nothing written, no systemctl.
- `add` with `lookPath` returning `/tmp/my bin/claude` → error mentions "space"; nothing written.
- `add` with a symlink pointing at `home` → error mentions "home folder".
- `add` with a missing folder → `errors.Is(err, os.ErrNotExist)`.
- All existing tests pass.

## Most important file changes

- `internal/remotecontrol/add.go`: the three checks
- `internal/remotecontrol/add_test.go`: new rows, stronger missing-folder row

## Open questions

- None

## Subtasks

Each item is one commit, ordered outside-in. The program builds and runs after every commit.
1. `add` refuses a repo path with a line break.
2. `add` refuses a `claude` path with a space or line break.
3. `add` follows symlinks when checking for the home folder.
4. The missing-folder test checks for "doesn't exist".
