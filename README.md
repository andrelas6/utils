# utils

Small Go programs for Linux automations.

## Layout

- `cmd/<name>/` — one folder per program (`package main`). The binary takes the folder's name.
- `internal/` — code shared between programs.

## Programs

| Program  | What it does                                                   |
| -------- | -------------------------------------------------------------- |
| `lspm` | Lists a folder's contents with their permissions, owner and group. |

## Usage

```bash
make build            # build all programs into bin/
make build-linux      # cross-compile for Linux into bin/linux/
make install          # install into $(go env GOPATH)/bin
make test

./bin/lspm /etc
```

## Adding a program

1. Create `cmd/<name>/main.go` with `package main`.
2. Put reusable logic in `internal/<pkg>/`.
3. `make build`.
