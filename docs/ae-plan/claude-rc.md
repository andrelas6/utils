# ae-plan: claude-rc — start Claude Remote Control per repo at login

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-init `claude-rc`

## What's being built

A new program, `claude-rc`, with three commands:

- `claude-rc add [-name n] [-dry-run] <repo>` — writes a systemd user service for that repo and turns it on.
- `claude-rc remove [-dry-run] <name>` — turns it off and deletes it.
- `claude-rc status` — lists each repo's service and whether it's running, plus whether Claude is logged in.

Each service runs `claude remote-control --name <host>-<name> --permission-mode default` in the repo folder, starts at login, and is restarted by systemd if it stops. `claude-rc` never runs Claude itself.

## Constraints

- Go standard library only. Same layout as `lspm`: thin `cmd/claude-rc/main.go`, logic in `internal/`.
- Development is on macOS (no systemd). Every piece of logic must be testable without systemd or `claude`. Real behaviour is checked by hand on the user's Ubuntu machine.
- `add` and `remove` change files and services, so both support `-dry-run` (AGENTS.md).
- Start at login, no lingering. Permission mode `default`. Spawn mode `same-dir`. Fresh session on restart, no `-c`.
- Never bypass the workspace trust or "Enable Remote Control?" prompts.
- **Build outside-in:** the `claude-rc` command comes first, backed by mocked output. Each later subtask replaces one mocked piece with the real thing, until nothing is mocked.

## What I found in the codebase

- `cmd/lspm/main.go`: `flag` + `run(out io.Writer, ...) error` + `log.Fatal`. `claude-rc` copies this shape, with one `flag.FlagSet` per subcommand. Unlike `lspm`, `run` gets a test file, because the command is built and tested first.
- `internal/fsinfo`: logic in `internal/`, tested with `t.TempDir()`. The new package follows the same pattern.
- No code in the repo runs other programs yet. The new package introduces a small "runner" so tests can fake `systemctl` and `claude`.
- Makefile `build` / `build-linux` already pick up any `cmd/*` folder; no Makefile change needed.

## What the docs say

- **Remote Control docs** (code.claude.com/docs/en/remote-control):
  - Workspace trust: when stdin/stdout is not a terminal, `claude remote-control` "can't ask and exits with a `Workspace not trusted` error". Trust is never saved for the home directory.
  - There is also a one-time `Enable Remote Control? (y/n)` prompt before the first server start.
  - Without an eligible login, `claude remote-control` exits with an error.
  - If the network is down, server mode gives up after ~10 minutes and the process exits.
- **systemd.service(5)** (man7.org):
  - A non-absolute `ExecStart` command is resolved with "a fixed search path determined at compilation time" (`/usr/local/bin`, `/usr/bin`, …), **not** the service's PATH. So `ExecStart=claude` will not find `~/.local/bin/claude`, even though the earlier `systemd-run … sh -c 'command -v claude'` check passed (that check went through `sh`, which does use PATH). → `claude-rc add` resolves the full path of `claude` and writes it into the unit.
  - `Restart=always` restarts on clean exits too; `on-failure` does not. `RestartSec` defaults to 100ms.
- **systemd.unit(5)** (man7.org): unit names may only use ASCII letters, digits, `:`, `-`, `_`, `.`, `\`. When the start rate limit is hit, systemd stops restarting the unit.

## Approach

`cmd/claude-rc/main.go` parses the subcommand and flags, then calls a small `services` interface with `Add`, `Remove` and `Status`. In the first subtask that interface is backed by a mock that returns canned output. Later subtasks build the real `internal/rcservice` package one method at a time and swap it in.

One plain unit file per repo at `~/.config/systemd/user/claude-rc-<name>.service`, not a template: the working directory, session name and `claude` path are written straight in, so the file reads on its own and needs no path escaping. The unit files are the only state; no separate config file. `<name>` defaults to the repo folder's name, cleaned to allowed unit-name characters; `-name` overrides it.

Unit content:

```ini
[Unit]
Description=Claude Remote Control for <name> (<repo>)

[Service]
WorkingDirectory=<repo>
ExecStart=<abs path to claude> remote-control --name %H-<name> --permission-mode default
Restart=always
RestartSec=60

[Install]
WantedBy=default.target
```

`Restart=always` with `RestartSec=60` covers both TBD cases (below) and the network case: whatever makes the process exit, systemd retries once a minute, and the 60s gap keeps it under the default start rate limit, so it never gives up for good. No `After=network-online.target`: it's a system target and the retry handles a late network.

`add`: check the repo is an existing folder and not `$HOME`; refuse if the unit already exists; resolve `claude`; write the file; `systemctl --user daemon-reload`; `systemctl --user enable --now claude-rc-<name>.service`. It also prints a reminder to run `claude remote-control` once by hand in that repo to answer the trust and enable prompts. With `-dry-run`, print the file and the commands instead.

`remove`: `systemctl --user disable --now`, delete the file, `daemon-reload`. Error if no such unit.

`status`: list `claude-rc-*.service` files; for each, ask `systemctl --user is-active`; show repo, name, state. Then run `claude auth status --json` and show `logged in: yes/no` from `loggedIn`.

All outside commands go through a `Runner` interface (`Run(name string, args ...string) ([]byte, error)`): the real one uses `os/exec`, `-dry-run` uses one that prints, tests use a fake that records calls and returns canned output. The unit folder is a parameter, so tests use `t.TempDir()`.

## Red / Green

Command (`cmd/claude-rc`):
- `add <repo>`, `add -name x <repo>`, `remove <name>`, `status` call the matching `services` method with the right arguments and print what it returns.
- `-dry-run` on `add` / `remove` reaches the service as dry-run.
- No subcommand, unknown subcommand, or missing `<repo>` / `<name>` → usage on stderr and an error.
- An error from the service is returned, not swallowed.

Service (`internal/rcservice`):
- Unit file for repo `/home/a/work/utils`, name `utils`, claude `/home/a/.local/bin/claude` matches the expected text exactly.
- Name from path: `/x/My Repo!` → `My-Repo`; `-name` overrides; empty result is an error.
- `Add` writes the file and calls `daemon-reload` then `enable --now claude-rc-utils.service`, in that order.
- `Add` refuses: missing folder, a file instead of a folder, `$HOME`, unit already exists, `claude` not found. Nothing written, no commands run.
- `Add` in dry-run writes nothing and runs nothing, but prints the unit and the commands.
- `Remove` calls `disable --now`, deletes the file, then `daemon-reload`; unknown name is an error.
- `Remove` in dry-run deletes nothing and runs nothing.
- `Status` lists each unit with the state `is-active` returns (`active`, `failed`, `inactive`), sorted by name; no units → "no repos set up".
- `Status` shows `logged in: yes` / `no` from `claude auth status --json`; if that command fails, shows `logged in: unknown` instead of failing.
- A `systemctl` error is returned with context (which command, which unit).

## Most important file changes

- `cmd/claude-rc/main.go` (new): subcommands, flags, `services` interface, wiring.
- `cmd/claude-rc/main_test.go` (new): command tests against a mocked `services`.
- `internal/rcservice/status.go`, `unit.go`, `add.go`, `remove.go` (new): the real `services`.
- `internal/rcservice/runner.go` (new): `Runner` interface, real (`os/exec`) and dry-run runners.
- `internal/rcservice/*_test.go` (new): tests above, with a fake runner and `t.TempDir()`.
- `README.md`, `AGENTS.md`: add `claude-rc` to the programs table and a short setup note (run `claude remote-control` once per repo; `claude auth login` when the login expires).

## Open questions

- **Untrusted repo — TBD, partly answered by the docs:** without a terminal, `claude remote-control` exits with `Workspace not trusted`. Not yet seen on the user's Ubuntu. Either way the design holds: the unit retries every 60s, `status` shows it, and `journalctl --user -u claude-rc-<name>` shows the error. Fix is to run `claude remote-control` once by hand in the repo.
- **Login expires while running — TBD:** docs say it exits at start without a login; what a running server does when the login expires mid-run is unknown. Either way: if it exits, systemd retries every 60s; if it stays up, `status` still shows `logged in: no`. Check by hand on Ubuntu when it next expires.
- **Exit code on these errors — TBD:** doesn't matter with `Restart=always`, noted only in case we later switch to `on-failure`.

## Subtasks

Each item is one commit. After every commit, `claude-rc` builds and runs; the parts not built yet return mocked output.

1. **The `claude-rc` command, mocked** — `cmd/claude-rc/main.go` + `main_test.go`: `add` / `remove` / `status` subcommands, flags, usage, all calling a `services` interface. Tests use a fake `services`; the binary uses a mock that prints canned output (e.g. `mock: would add utils`).
2. **Real `status`** — `internal/rcservice`: `Runner` interface, real `os/exec` runner, `Status` (list unit files, `is-active`, `claude auth status --json`). Tests with a fake runner and `t.TempDir()`. `main.go` swaps the mocked `status` for the real one.
3. **Unit file text and name** — `unit.go` + tests: name cleaning, `-name` override, exact unit text. Not wired yet; it's what `add` needs next.
4. **Real `add`** — `Add` + tests: folder checks, `$HOME` refused, existing unit refused, `claude` found with `exec.LookPath`, writes file, `daemon-reload` then `enable --now`. `main.go` swaps in the real `add`.
5. **Real `remove`** — `Remove` + tests: `disable --now`, delete, `daemon-reload`, unknown name error. `main.go` swaps in the real `remove`; the mock is deleted.
6. **Dry run** — dry-run runner + tests: `add` / `remove` with `-dry-run` print the unit and commands and never write, delete or run anything.
7. **Docs** — README + AGENTS.md table and setup note. `go vet`, `make test`, `make build-linux` pass. Then manual check on Ubuntu with the ae-init steps.

**Size:** roughly 250 lines of code + 400 of tests ≈ 650 lines, over the 500-line mark. Proposed split, so each PR leaves a working command: **PR 1** = subtasks 1–3 (command with real `status`; `add` / `remove` still mocked); **PR 2** = subtasks 4–7 (real `add` / `remove`, dry run, docs, Ubuntu check).
