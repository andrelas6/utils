# ae-plan: e2e-ci — claude-rc end-to-end against real systemd in CI

> **Status:** Done — built in andrelas6/utils#10
> **Last updated:** 2026-10-10
> **From:** ae-init `e2e-ci`

## What's being built

A second job, `e2e`, in `.github/workflows/checks.yml`. On `ubuntu-latest` it turns on lingering for the `runner` user so a systemd user manager is running, builds `claude-rc`, and runs `e2e/run.sh`. The script uses the real binary and real `systemctl --user`, with a fake `claude` (`e2e/fake-claude`) on `PATH`, and goes through: add → active, status, kill → restart, failing claude → restarting, refusals, remove → gone. Any failed check fails the job and prints the user journal.

## Constraints

- No change to `claude-rc` itself; the test drives the binary from the outside.
- `RestartSec=60` stays. Restart scenarios wait up to ~90s each, so the job takes a few minutes. Shorten later (user's call).
- Lingering on the GitHub runner is unproven, so it's the first subtask; if it doesn't work, stop and rethink (fallbacks: `systemctl --user --machine=runner@.host`, or a Lima VM job).
- Can't run on macOS (no systemd). Every subtask is checked by pushing to the PR and reading the `e2e` job.
- **TDD fits loosely:** the script *is* the test, and it checks behaviour that already exists, so new scenarios go green right away. How we know a scenario really checks something: once at the end, a throwaway commit breaks `claude-rc` (`Restart=no`) and the `e2e` job must go red, then the commit is dropped.

## What I found in the codebase

- `.github/workflows/checks.yml`: one `checks` job (Ubuntu + macOS); its "Smoke test" step already makes a stub `claude` in `$RUNNER_TEMP/bin` and runs `add -dry-run`.
- `internal/remotecontrol/unit.go`: `WorkingDirectory=<repo>`, `ExecStart=<full claude path> remote-control --name %H-<name> --permission-mode default`, `Restart=always`, `RestartSec=60`, `WantedBy=default.target`.
- `internal/remotecontrol/add.go`: finds `claude` with `exec.LookPath` at `add` time, so the fake's full path ends up in the unit; refuses `$HOME` (following links), existing units, line breaks.
- `internal/remotecontrol/status.go`: `systemctl --user show … --property=ActiveState,SubState,NRestarts,ExecMainStatus`; prints `NAME STATE RESTARTS LAST EXIT REPO`, `restarting` for `SubState=auto-restart`, a `see why … journalctl` line, `logged in:` from `claude auth status --json`.
- `internal/remotecontrol/remove.go`: `disable --now`, delete, `daemon-reload`; errors "isn't a valid name" / "no repo set up as".
- `cmd/claude-rc/main.go`: unit folder from `os.UserConfigDir()` → `~/.config/systemd/user` on Linux.

## What the docs say

- loginctl(1), `enable-linger`: "a user manager is spawned for the user at boot and kept around after logouts. This allows users who are not logged in to run long-running services." (read from `systemd/systemd` `man/loginctl.xml`; man7.org and freedesktop.org were unreachable.)
- user@.service(5): `user-runtime-dir@UID.service` "creates the user's runtime directory /run/user/UID".
- systemctl(1): `--user` talks "to the service manager of the calling user"; `--machine=` with `.host` "is useful to connect to a specific user's user bus: --user --machine=lennart@.host". `kill` sends a signal to a unit's processes (`--signal=`).
- A unit's processes get the user manager's environment, not the CI step's, so the fake `claude` can't read a step env var to pick its behaviour; it looks for a marker file in its working directory instead.

## Approach

`e2e/fake-claude` is a small shell script: `auth status --json` prints `{"loggedIn": true}`; `remote-control …` writes its arguments and working directory to a log, then exits 1 with "Workspace not trusted" if the current folder has a `.fail` file, else sleeps forever. `e2e/run.sh` puts it on `PATH` as `claude`, makes test repos under `$RUNNER_TEMP`, and runs scenarios with two helpers: `check <label> <command>` (prints PASS/FAIL, counts failures) and `wait_for <seconds> <command>`. A `trap` prints `claude-rc status` and `journalctl --user` on failure. The CI job enables lingering, exports `XDG_RUNTIME_DIR`, waits for `systemctl --user is-system-running` to answer, then runs the script.

## Red / Green

- `systemctl --user` answers on the runner after lingering is on.
- `add <healthy repo>`: unit file exists; `systemctl --user is-active` → `active`; the fake was started in that repo with `remote-control --name <host>-<name> --permission-mode default`.
- `status`: row `<name> active 0 0 <repo>` and `logged in: yes`.
- Kill the healthy unit's process → within ~90s it's `active` again and `NRestarts` ≥ 1.
- `add <failing repo>` → `status` shows it as `restarting` (or `activating`/`failed`), last exit `1`, and `see why <name> isn't running: journalctl --user -u claude-rc-<name>`; the real `SubState` is printed to the log (answers the open question).
- Refusals, nothing changed: `add $HOME`; `add` the same repo twice; `add` a folder with a line break; `remove ../x`; `remove nope`.
- `remove` both → no `claude-rc-*` unit files, `systemctl --user is-active` not `active`, `status` says `no repos set up`.
- A throwaway `Restart=no` commit makes the `e2e` job red.

## Most important file changes

- `.github/workflows/checks.yml`: new `e2e` job
- `e2e/run.sh` (new): scenarios and helpers
- `e2e/fake-claude` (new): stand-in for `claude`
- `AGENTS.md`, `README.md`: one line each about the e2e job

## Open questions

- ~~Lingering on the GitHub runner~~ — works: `sudo loginctl enable-linger runner` + `XDG_RUNTIME_DIR=/run/user/1001` gives a working `systemctl --user` (run 38044380815).
- ~~`SubState=auto-restart`~~ — confirmed: a unit waiting for its next try reports `ActiveState=activating SubState=auto-restart`, and `claude-rc status` shows `restarting` (run 38046266652).
- ~~Does the job catch a real break?~~ — yes: a throwaway `Restart=no` commit failed "systemd starts it again within 90s" and "status shows it active again with a restart" (run 38046653384), then was dropped.
- `RestartSec=60` makes the job take ~2 minutes. Shortening it is left for later, as agreed.

## Subtasks

Each item is one commit, ordered outside-in. The PR is opened after subtask 1 so every later commit runs in CI.
1. `e2e` job: lingering on, `XDG_RUNTIME_DIR` set, waits for the user manager, runs `e2e/run.sh`, which only checks that `systemctl --user` answers. Proves (or disproves) lingering.
2. Fake `claude` + healthy repo: `add` → `active`, started with the right arguments in the right folder; `status` row and `logged in: yes`.
3. Restart: kill the process, wait, `active` again with `NRestarts` ≥ 1.
4. Failing repo: `restarting`/`failed`, last exit 1, `journalctl` hint; log the real `SubState`.
5. Refusals: home, twice, line break, bad remove names.
6. Remove: both units gone, `no repos set up`.
7. AGENTS.md and README; throwaway `Restart=no` commit proves the job goes red, then dropped. Expected size ~200 lines.
