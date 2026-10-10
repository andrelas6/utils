# ae-init: e2e-ci — claude-rc end-to-end against real systemd in CI

> **Status:** Done — built in andrelas6/utils#10
> **Last updated:** 2026-10-10

## What to build

An end-to-end test that runs the real `claude-rc` binary against a real systemd user session on every pull request, so a change that breaks `add`, `status`, `remove` or the restart behaviour on Linux goes red before it's merged. A real `claude` can't run in CI (it needs a claude.ai login), so a fake `claude` stands in. It can either keep running, like a healthy Remote Control server, or keep exiting, like an untrusted repo or an expired login. It also answers `claude auth status --json`. Running `claude-rc` on a local VM (Lima) is out of scope for now.

## A -> B

- **A (now):** The `checks` workflow only runs `claude-rc add -dry-run` (step "Smoke test" in `.github/workflows/checks.yml`). Nothing ever runs `systemctl --user`, so the unit file, the real `add` / `remove`, `status` and restarts are only checked by hand on Ubuntu.
- **B (after):** Each PR also runs an e2e job on `ubuntu-latest`. It adds a repo for real, sees the unit go `active`, kills it and sees systemd restart it, sees a failing `claude` show as `restarting` / `failed` in `status`, checks the refusals, and removes the repo again. Any wrong result fails the job.

## How to reproduce

**Preconditions**
- GitHub repo `andrelas6/utils` with Actions on; a PR against `main`.
- For a local look at the same thing: an Ubuntu machine with a user systemd session.

**Steps**
1. Open a PR and look at its checks (`gh pr checks <n>`).
2. Open the e2e job's log.
3. Break something on purpose in a throwaway commit (e.g. change `Restart=always` to `Restart=no` in the unit text, or make `remove` skip `disable --now`) and push.

**Observe**

| | A (now) | B (after) |
|---|---|---|
| Checks on a PR (step 1) | `checks (ubuntu-latest)`, `checks (macos-latest)` | Same two, plus an e2e check on Ubuntu |
| e2e log (step 2) | n/a | Each scenario named with pass/fail: add → `active`; `status` row + `logged in: yes`; kill → restarted, `NRestarts` up; failing claude → `restarting`/`failed` + restart count + last exit + `journalctl` hint; refusals; remove → no unit, `no repos set up` |
| Broken change (step 3) | All green: dry run doesn't notice | e2e check red, naming the scenario that broke |

## Open questions

- ~~Lingering on the GitHub runner~~ — works: `sudo loginctl enable-linger runner` + `XDG_RUNTIME_DIR=/run/user/1001` gives a working `systemctl --user` (run 38044380815).
- ~~`SubState=auto-restart`~~ — confirmed: a unit waiting for its next try reports `ActiveState=activating SubState=auto-restart`, and `claude-rc status` shows `restarting` (run 38046266652).
- ~~Does the job catch a real break?~~ — yes: a throwaway `Restart=no` commit failed "systemd starts it again within 90s" and "status shows it active again with a restart" (run 38046653384), then was dropped.
- `RestartSec=60` makes the job take ~2 minutes. Shortening it is left for later, as agreed.
