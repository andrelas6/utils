# ae-plan: review-error-handling — claude-rc says why something is wrong

> **Status:** Done
> **Last updated:** 2026-10-10
> **From:** ae-init `review-error-handling`

## What's being built

- `status` reads `systemctl --user show <unit> --property=ActiveState,SubState,NRestarts,ExecMainStatus` instead of `is-active`, and prints `NAME STATE RESTARTS LAST EXIT REPO`. `STATE` is `restarting` when `SubState=auto-restart`, otherwise `ActiveState`. After the table, one `see why <name> isn't running: journalctl --user -u claude-rc-<name>` line per repo that isn't `active`.
- A unit whose state can't be read gets an `unknown` row and a `can't read <name>'s state: <error>` line; `status` doesn't stop.
- `remove`: separate errors for an invalid name, a unit that isn't set up, and a unit folder that can't be checked.
- `add`: only "file doesn't exist" means the name is free; any other error stops with "can't check …".
- `add`: when `daemon-reload` or `enable --now` fails, it runs `remove` to undo, and the error says the unit was removed again (or, if that also fails, how to undo by hand).

## Constraints

- Output of a healthy `status` changes (new columns); everything else that works today prints the same.
- Stay with plain errors and `fmt.Errorf`; no new error types needed for a CLI that prints and exits.
- Testable on macOS with the fake machine; `chmod 000` on a temp folder for "can't check".
- PR 4 of 5, stacked on `review/3-simplicity`.

## What I found in the codebase

- `internal/remotecontrol/status.go`: `unitState` runs `is-active`; an error aborts `Status`.
- `internal/remotecontrol/remove.go`: one `unknown` error for three checks.
- `internal/remotecontrol/add.go`: `if _, err := os.Stat(unitPath); err == nil` — other errors fall through.
- `add.go`'s systemctl loop: error message carries "to undo, run: claude-rc remove <name>".
- `fake_test.go`: `fakeMachine` replies keyed by full command line; files go to disk.

## What the docs say

- systemctl(1): `show` takes `--property` with several comma-separated names and prints them as `name=value` (implied by `--value`, which drops the name and `=`). Showing a non-existent unit is allowed.
- org.freedesktop.systemd1(5): Service has `NRestarts`; `ExecMainCode`/`ExecMainStatus` "contain information of the last run of the process or of the current running process".
- systemd.service(5) (read for PR 1): `Restart=always` restarts on any exit; `RestartSec` is the wait.
- `auto-restart` as a sub state isn't in the parts of the man pages I read → open question.

## Approach

Status first (the review's main point), then the error splits, then rollback. Each change is one Red/Green cycle with the fake machine. `status` parses the `show` output into a small map; unknown keys are ignored, missing ones print `?`.

## Red / Green

- `status`: active unit → `active  0  0`; auto-restart unit with `NRestarts=4`, `ExecMainStatus=1` → `restarting  4  1` plus the `journalctl` line; failed unit → `failed` plus the line.
- `status`: `systemctl show` fails for one unit → that row `unknown  ?  ?`, a `can't read` line with the error, other rows still there, no error returned.
- `remove ../x` → "not a valid name"; `remove web` (not set up) → "no repo set up as \"web\""; unreadable unit folder → "can't check"; none run systemctl.
- `add` with an unreadable unit folder → "can't check", nothing written, no systemctl.
- `add` where `enable --now` fails → error mentions systemctl's message and "removed again"; unit file gone; calls: `daemon-reload`, `enable --now`, `disable --now`, `daemon-reload`.
- `add` where `enable --now` fails and the rollback fails too → error keeps "to undo, run: claude-rc remove <name>".

## Most important file changes

- `internal/remotecontrol/status.go`: `show` instead of `is-active`, new columns, `unknown` rows, hint lines
- `internal/remotecontrol/remove.go`: three distinct errors
- `internal/remotecontrol/add.go`: `ErrNotExist` check, rollback via `Remove`
- `internal/remotecontrol/*_test.go`: updated and new cases
- `README.md`: `status` example and "when something is wrong" section

## Open questions

- `SubState=auto-restart` while waiting to restart — TBD on Ubuntu (see ae-init).

## Subtasks

Each item is one commit, ordered outside-in. The program builds and runs after every commit.
1. `status` shows restarting, restart count, last exit and a `journalctl` line per repo that isn't running.
2. `status` shows `unknown` for a unit it can't read instead of stopping.
3. `remove` gives a different error for a bad name, a missing unit and an unreadable folder.
4. `add` stops when it can't tell whether the unit already exists.
5. `add` removes the unit again when systemctl fails.
6. README: new `status` output and troubleshooting.
