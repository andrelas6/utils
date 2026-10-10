# AGENTS.md

Guidance for AI coding agents working in this repository.

## Goal

A collection of small, single-purpose Go programs that automate everyday tasks on Linux.
Each program does one thing, has few or no dependencies, and is easy to read.
Targets Linux; development often happens on macOS, so programs must cross-compile with `make build-linux`.

## Layout

```
cmd/<name>/main.go    # one folder per program (package main); binary takes the folder name
internal/<pkg>/       # code shared between programs; not importable outside this module
Makefile              # build, build-linux, install, test, clean
```

Existing programs:

| Program     | What it does                                                                                       |
| ----------- | -------------------------------------------------------------------------------------------------- |
| `lspm`      | Lists a folder's contents with their permissions, owner and group.                                 |
| `claude-rc` | Sets up systemd user units that run `claude remote-control` per repo at login (`add`, `remove`, `status`). |

`claude-rc` can't be run for real on macOS (no systemd): test logic with the fake runner in `internal/remotecontrol`, use `-dry-run` locally, and check real behaviour on Linux.

## Conventions

- Single Go module: `github.com/andrelas6/utils`.
- Prefer the standard library (`flag`, `os`, `os/exec`, `path/filepath`, `log`). Add a dependency only when it clearly pays off.
- Keep `main.go` thin: parse flags, call a `run(...) error`, `log.Fatal` on error. Put logic in `internal/` so it can be tested.
- Wrap errors with context: `fmt.Errorf("read dir %q: %w", dir, err)`.
- Use build tags (`//go:build unix`, `//go:build linux`) for OS-specific code, with a fallback file so the module still compiles elsewhere.
- Programs that delete, move or overwrite files must support a `-dry-run` flag.
- Every `internal/` package has tests. Use `t.TempDir()` for filesystem tests.
- Put a comment on top of every test function that explains the scenario it tests, in plain words (e.g. `// Bad arguments print the usage and never reach the services.`). This is the one exception to "no comments" in code.
- When adding a program, add it to the table above and to `README.md`.

## Commands

```bash
make test          # go test ./...
make build         # builds all programs into bin/
make build-linux   # cross-compiles for Linux into bin/linux/
go vet ./...
```

Before declaring work done: `go vet ./...` and `make test` pass, and `make build-linux` compiles.

## Workflow

Every change goes through these skills, in order:

1. **`/ae-init`**: pin down the change. What to build, A → B (current state → state after), and how to reproduce both A and B so the change can be checked before and after. No design, no code.
2. **`/ae-plan`**: turn the ae-init doc into a blueprint: approach, files that change, and a Red/Green TDD build order split into subtasks. **Each subtask is one commit. Stop after each subtask and wait for the user to review before committing.**
3. **`/ae-arch`** *(only when needed)*: for changes that add a new program with several parts, new shared `internal/` packages, or anything touching how programs fit together. Produces a C4-style design with mermaid diagrams. Skip it for small, single-file changes.
4. **`/ae-review`**: review the finished branch or diff before it is merged. Read-only; it reports findings and does not change code.

Rules for agents:

- Don't write feature code before an ae-plan exists and the user has asked to build.
- Follow the plan's subtask order; write the failing test first, then the code that makes it pass.
- Commit every ae-init, ae-plan and ae-review document with the change it describes: copy it from `~/.aecli/artifacts/utils/<ae-init|ae-plan|ae-review>/<slug>.md` to `docs/<ae-init|ae-plan|ae-review>/<slug>.md`, and update the copy whenever the doc changes. ae-review slugs are `pr-<number>`.
- Don't commit or push unless the user asks.
- Never add a `Co-Authored-By:` line (or any other AI attribution) to commit messages or PR descriptions.
