# ae-init: review-edge-cases — claude-rc add refuses paths that would break the unit

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-review `pr-1`, section "also worth a look"

## What to build

Close the small gaps in `claude-rc add` that the review found: paths that would quietly produce a broken or wrong unit file, a home-folder check that a symlink can get around, and a test that would pass for the wrong reason.

## A -> B

- **A (now):**
  - A repo path with a line break is written into the unit as-is; the part after the break becomes its own unit line.
  - If `claude` lives in a folder with a space, `ExecStart` splits the path and the unit can't start.
  - `add` compares the repo with `$HOME` as text, so a symlink to the home folder (or `/home` → `/var/home`) gets past the check.
  - The "missing folder" test only checks that the error mentions `nope`.
- **B (after):**
  - `add` refuses a repo path with a line break.
  - `add` refuses a `claude` path with a space or line break, saying why.
  - The home check follows symlinks on both sides.
  - The test checks that the error is "file doesn't exist".

## How to reproduce

**Preconditions**
- macOS or Linux, binary from `make build` on `review/4-error-handling` (A) or `review/5-edge-cases` (B). All steps use `-dry-run`, so nothing changes.

**Steps**
1. `mkdir "$TMPDIR/two"$'\n'"lines" && claude-rc add -dry-run "$TMPDIR/two"$'\n'"lines"`
2. `mkdir -p "/tmp/my bin" && ln -sf "$(command -v claude)" "/tmp/my bin/claude" && PATH="/tmp/my bin:$PATH" claude-rc add -dry-run .`
3. `ln -s "$HOME" /tmp/myhome && claude-rc add -dry-run /tmp/myhome`
4. `go test ./internal/remotecontrol/ -run TestAddRefuses -v`

**Observe**

| | A (now) | B (after) |
|---|---|---|
| Line break (step 1) | Prints a unit with a broken `WorkingDirectory=` and an extra line | Error: the path has a line break |
| Space in claude path (step 2) | `ExecStart=/tmp/my bin/claude …` | Error: can't use a claude path with a space |
| Symlinked home (step 3) | Prints a unit for the home folder | Error: that's your home folder |
| Missing folder test (step 4) | Passes on any error mentioning `nope` | Passes only when the error is "doesn't exist" |

## Open questions

- None
