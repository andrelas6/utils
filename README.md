# utils

Small Go programs for Linux automations.

## Layout

- `cmd/<name>/` — one folder per program (`package main`). The binary takes the folder's name.
- `internal/` — code shared between programs.

## Programs

| Program     | What it does                                                                      |
| ----------- | --------------------------------------------------------------------------------- |
| `lspm`      | Lists a folder's contents with their permissions, owner and group.                |
| `claude-rc` | Starts a Claude Remote Control server per repo at login, using systemd user units. |

## Usage

```bash
make build            # build all programs into bin/
make build-linux      # cross-compile for Linux into bin/linux/
make install          # install into $(go env GOPATH)/bin
make test

./bin/lspm /etc
```

Every pull request runs formatting, `go vet`, the tests, the Linux build and a smoke test on Ubuntu and macOS (`.github/workflows/checks.yml`).
An `e2e` job also runs `claude-rc` for real against systemd on Ubuntu, with a fake `claude`; see [End-to-end tests](#end-to-end-tests).

## claude-rc

Runs `claude remote-control` in each repo you add, as a systemd user service that starts when you log in and restarts a minute after it stops. You then open the sessions from claude.ai/code or the Claude mobile app.

```bash
claude-rc add [-name n] [-dry-run] <repo>   # set up a repo (name defaults to the folder name)
claude-rc remove [-dry-run] <name>          # stop it and delete its unit
claude-rc status                            # each repo's state, and whether claude is logged in
```

Flags go before the repo or name.

**Before adding a repo**

1. Log in to Claude on the machine: `claude auth login`.
2. Run `claude remote-control` once by hand in the repo and answer the trust and "Enable Remote Control?" prompts. The service can't answer them and will keep failing until you do.
3. Try `claude-rc add -dry-run <repo>` to see the unit file and commands first.

**When something is wrong**

`claude-rc status` shows each repo's state, how often systemd restarted it and how it last exited, plus where to look for anything that isn't running:

```
NAME       STATE       RESTARTS  LAST EXIT  REPO
untrusted  restarting  4         1          /home/me/workspace/untrusted
utils      active      0         0          /home/me/workspace/utils

see why untrusted isn't running: journalctl --user -u claude-rc-untrusted

logged in: yes
```

- `restarting` or `failed`: claude keeps exiting; the `journalctl` line shows why. Usually the repo isn't trusted yet (run `claude remote-control` once in it) or the login expired.
- `unknown`: `systemctl` couldn't be asked; the reason is printed under the table.
- `logged in: no`: the Claude login expired. Run `claude auth login`; the services pick it up on their next restart.
- Units live in `~/.config/systemd/user/claude-rc-<name>.service`. They start at login, not at boot.

## End-to-end tests

The `e2e` job in `.github/workflows/checks.yml` runs `e2e/run.sh` on Ubuntu for every pull request. It turns on lingering so the runner has a systemd user session, then drives the real `claude-rc` binary against real `systemctl --user`. A real `claude` needs a claude.ai login, so `e2e/fake-claude` stands in. It answers `claude auth status --json` with `loggedIn: true`. For `remote-control`, it logs its arguments and folder and keeps running, or exits with 1 ("Workspace not trusted") if the repo has a `.fail` file.

The cases, in the order they run. Each prints `PASS` or `FAIL`, and any failure prints the units and the user journal and fails the job.

**Setup**
1. `systemctl --user` answers.

**Healthy repo**

2. `claude-rc add <repo>` succeeds.
3. The unit file `~/.config/systemd/user/claude-rc-healthy.service` is written.
4. The unit becomes `active` within 30s.
5. claude runs in the repo folder as `remote-control --name <host>-healthy --permission-mode default`.
6. `claude-rc status` succeeds.
7. `status` shows `healthy  active  0  0  <repo>`.
8. `status` shows `logged in: yes`.

**Restart after a crash**

9. The claude process is killed (`systemctl --user kill --signal=SIGKILL`).
10. systemd starts it again within 90s (`active`, `NRestarts` ≥ 1).
11. `claude-rc status` succeeds.
12. `status` shows `healthy  active` with at least one restart.

**Repo where claude keeps exiting**

13. `claude-rc add <repo with .fail>` succeeds.
14. claude exits with 1 and the unit isn't `active`.
15. `claude-rc status` succeeds.
16. `status` shows `broken  restarting` (or `activating` / `failed`) with last exit `1`.
17. `status` shows `see why broken isn't running: journalctl --user -u claude-rc-broken`.

**Refusals** (each must fail with the right message and change nothing)

18. `add $HOME` → "home folder".
19. `add` a repo that's already set up → "already set up".
20. `add` a folder whose name has a line break → "line break".
21. `remove ../healthy` → "isn't a valid name".
22. `remove nope` → `no repo set up as "nope"`.
23. Still exactly two units.

**Remove**

24. `claude-rc remove healthy` succeeds.
25. `claude-rc remove broken` succeeds.
26. No `claude-rc-*` unit files are left.
27. `healthy` is no longer running.
28. `broken` is no longer running.
29. `claude-rc status` succeeds.
30. `status` shows `no repos set up`.

The job takes about 2 minutes, mostly waiting on `RestartSec=60` in the restart case. It can't run on macOS (no systemd).

## Adding a program

1. Create `cmd/<name>/main.go` with `package main`.
2. Put reusable logic in `internal/<pkg>/`.
3. `make build`.
