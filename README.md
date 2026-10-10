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

- `claude-rc status` shows `failed` or keeps restarting: look at `journalctl --user -u claude-rc-<name>`.
- `logged in: no`: the Claude login expired. Run `claude auth login`; the services pick it up on their next restart.
- Units live in `~/.config/systemd/user/claude-rc-<name>.service`. They start at login, not at boot.

## Adding a program

1. Create `cmd/<name>/main.go` with `package main`.
2. Put reusable logic in `internal/<pkg>/`.
3. `make build`.
