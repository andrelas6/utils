# ae-init: claude-rc — start Claude Remote Control per repo at login

> **Status:** Done — built in andrelas6/utils#1 (Ubuntu check still to do)
> **Last updated:** 2026-10-10

## What to build

A new program in this repo, `claude-rc`, for a Linux machine. You add repositories one at a time with `claude-rc add <repo>`. For each one, it sets up a systemd user service that runs `claude remote-control` in that repo's folder. When you log in, a Remote Control server for every added repo comes up on its own. You can then open and drive sessions in those repos from claude.ai/code or the Claude mobile app. systemd keeps the processes running and restarts them if they crash. `claude-rc` only sets up, removes and reports on the services; it never runs Claude itself. Sessions use the `default` permission mode, so Claude asks before edits and commands.

## A -> B

- **A (now):** The repo has one program, `lspm` (`cmd/lspm`). Nothing starts Claude. After logging in, you must SSH or open a terminal, `cd` into each repo and run `claude remote-control` by hand. Closing the terminal stops it.
- **B (after):** You run `claude-rc add <repo>` once per repo. After every login, each added repo shows up in claude.ai/code as a Remote Control session with no manual step, and `claude-rc status` shows each one as running.

## Decisions

- **Start at login**, not at boot. No lingering.
- **Permission mode:** `default`.
- **Spawn mode:** `same-dir` (the `claude remote-control` default).
- **On restart:** start a fresh session. No `-c` reattach.
- **Login expiry:** the Claude login expires every few days and that is outside our control. The user logs in again by hand (`claude auth login`). `claude-rc` should make an expired login easy to see (e.g. `claude-rc status` runs `claude auth status --json`), not try to fix it.
- **Workspace trust:** Claude Code 2.1.291 has no flag to auto-trust a repo (`claude --help` only mentions trust being skipped in `-p`/non-TTY print mode). Each repo is trusted once by hand with an interactive `claude` run. `claude-rc` should not bypass the trust prompt.
- **Finding `claude`:** `systemd-run --user --wait --pipe sh -c 'command -v claude'` found `claude` on the user's Ubuntu machine (2026-10-10), but that check goes through `sh`, which uses PATH. systemd's `ExecStart` does not use PATH (systemd.service(5): fixed search path only), so the unit can't call `claude` by name. `claude-rc add` writes claude's full path into the unit instead (found with `exec.LookPath` when you run `add`).
- **Testing:** manual, on the user's Ubuntu machine, using the binary from `make build-linux`. Development is on macOS (no systemd), so automated tests cover the logic that does not need systemd.

## How to reproduce

**Preconditions**
- Ubuntu with systemd and a user session (desktop login or SSH).
- Claude Code installed and logged in for this user (`claude auth status` shows logged in).
- Two test repos cloned, e.g. `~/workspace/utils` and `~/workspace/other`.
- `claude remote-control` run once by hand in each repo, with the trust and "Enable Remote Control?" prompts answered `y`.
- `claude` on your shell PATH when you run `claude-rc add` (its full path goes into the unit).
- For B: `claude-rc` built with `make build-linux` and copied to the machine.

**Steps**
1. A: log out and back in (or reboot and log in).
2. A: `systemctl --user list-units 'claude-rc-*'`.
3. A: open claude.ai/code (or the mobile app) and look for Remote Control sessions from this machine.
4. B: `claude-rc add -dry-run <repo>` for each of the two repos, then `claude-rc add <repo>` for real.
5. B: log out and back in (or reboot and log in), then repeat steps 2–3 and run `claude-rc status`.
6. B: kill one `claude remote-control` process and watch it come back.
7. B: `claude-rc remove <name>` for one repo and log in again.
8. B: `claude auth logout`, then `claude-rc status`; then `claude auth login` again.

**Observe**

| | A (now) | B (after) |
|---|---|---|
| `systemctl --user list-units 'claude-rc-*'` | No units | One unit per repo, `active (running)` |
| claude.ai/code / mobile app | No sessions from this machine | One Remote Control session per repo, named after host and repo |
| `claude-rc status` | Command does not exist | Each repo listed with its name, systemd state (`active`, `activating`, `failed`, …) and folder, plus `logged in: yes/no/unknown` |
| `journalctl --user -u 'claude-rc-*'` | No entries | Startup lines from `claude remote-control` per repo |
| Killed process (step 6) | n/a | systemd restarts it within ~60s (`RestartSec=60`), as a fresh session |
| `-dry-run` (step 4) | n/a | Prints the unit file, the trust reminder and the systemctl commands; nothing written or enabled |
| Removed repo (step 7) | n/a | Its unit is disabled and gone; its session no longer appears after login |
| Logged out (step 8) | n/a | `claude-rc status` says Claude is not logged in |
| Before any login | Nothing | Nothing — services start at login, not at boot |

## Open questions

- **Untrusted repo:** the Remote Control docs say it exits with `Workspace not trusted` when there's no terminal; not yet seen on Ubuntu. `claude-rc add` prints a reminder to run `claude remote-control` once by hand before it starts the unit, and systemd retries every 60s.
- **Expired login under systemd:** TBD — exit (systemd retries every 60s) or stay up and fail quietly? Either way `claude-rc status` shows `logged in: no`.
