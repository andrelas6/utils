# ae-review: feat: claude-rc, start Claude Remote Control per repo at login (andrelas6/utils#1)

**problem being solved** before: Remote Control per repo needs a terminal, `cd`, `claude remote-control` by hand; dies when terminal closes. after: `claude-rc add <repo>` once → systemd user unit runs it at every login, restarts on exit, `claude-rc status` shows state + login.

**opportunities to learn**
- consumer-defined interface: `services` lives in `cmd/claude-rc/main.go:24`, not in `rcservice`; Go idiom that made outside-in build + fake possible
- `errors.AsType[*exec.ExitError]` (Go 1.26 generic `errors.As`) `internal/rcservice/runner.go:20`; `Output()` fills `ExitError.Stderr` for free
- `*bool` to tell "field missing" from `false` when decoding JSON `internal/rcservice/status.go:67`
- indexed verbs `%[1]s` reuse one arg several times in a template; `%%H` escapes to systemd specifier `internal/rcservice/unit.go:31`
- `flag.NewFlagSet` + `ContinueOnError` per subcommand, errors returned not `os.Exit` → testable `run` `cmd/claude-rc/main.go:94`
- `t.Chdir` (Go 1.24) for relative-path test, auto-restored `internal/rcservice/add_test.go`
- strategy swap for dry run: same code path, `DryRunner` in place of `ExecRunner` `internal/rcservice/add.go:33`

**error handling**
- `Restart=always` + `RestartSec=60`: unit never reaches `failed`; during the wait `is-active` says `activating`. untrusted repo / expired login looks like "starting" forever in `status`. add restart count or last exit (`systemctl show -p NRestarts,ExecMainStatus`) `internal/rcservice/status.go:55`
- `Remove` returns one `unknown` error for 3 causes: bad name, missing file, `Stat` permission error. last one misreported as "no repo set up" `internal/rcservice/remove.go:11-19`
- `Add` treats any `Stat` error as "not there yet" → permission error falls through to `WriteFile` with a less clear message `internal/rcservice/add.go:23`
- `Status` aborts whole table on first `is-active` error, but login check degrades to `unknown`; inconsistent. show `unknown` per row instead `internal/rcservice/status.go:39-42`
- `Add` half-done on `enable --now` failure leaves unit file; message says how to undo, but nothing rolls back automatically `internal/rcservice/add.go:50`
- in production only `journalctl --user -u claude-rc-<name>` tells why it loops; `status` doesn't point there

**null handling**
- `Service` zero value has nil `Runner` / `LookPath` → nil-func panic if a field is forgotten. all 4 fields required; `rcservice.New(unitDir, home)` filling `ExecRunner` + `exec.LookPath` would make that explicit `internal/rcservice/status.go:19`

**simplicity of the solution**
- `if dryRun` 3× in `Add` and 3× in `Remove`; file write/delete could go behind the runner (or a small `files` interface) so dry run is only a swap `internal/rcservice/add.go:34,55`, `remove.go:22,29,38`
- `Service` struct + `Status` defined in `status.go`; struct belongs in its own file or `rcservice.go`, readers look for it there
- PR carries unrelated repo rules (test comments, co-author, docs commit) + `fsinfo_test.go` comments; separate docs PR would shrink review

**naming and language**
- `unitFile` (content) vs `unitFileName` (file name): one letter apart, different things. `unitFileContent` `internal/rcservice/unit.go:24,28`
- `checkRepo` also resolves to absolute path; name hides the return value. `resolveRepo` `internal/rcservice/add.go:64`
- `Service` is the thing that manages systemd services; clashes with "service" = unit. `Manager` or `Units`
- `rcservice` package: `rc` abbreviation; `remotecontrol` reads plain

**how to reproduce**
- Ubuntu, `claude auth login`, `claude remote-control` once by hand in `~/workspace/utils` (answer trust + enable)
- before: log out/in → `systemctl --user list-units 'claude-rc-*'` empty, no session in claude.ai/code
- after: `make build-linux`, copy binary, `claude-rc add ~/workspace/utils`, log out/in → unit `active`, session `<host>-utils` in claude.ai/code, `claude-rc status` row + `logged in: yes`
- failure path: `claude auth logout` → `status` shows `logged in: no`; `pkill -f 'claude remote-control'` → back within ~60s

**comment usage**
- nothing

**reuse of existing code**
- reuse of existing code: none found in repo (`internal/fsinfo` unrelated). `systemd-escape` exists but plain one-file-per-repo choice avoids needing it

**also worth a look**
- newline in repo path written raw into unit → extra unit lines (`WorkingDirectory=/x\nExecStartPre=...`). self-inflicted only, but reject `\n` in `checkRepo` `internal/rcservice/unit.go:35`
- `claude` path with a space breaks `ExecStart` (known, unquoted) `internal/rcservice/unit.go:36`
- home check is string compare; symlinked home (`/home` → `/var/home`) slips past `internal/rcservice/add.go:76`
- `TestAddRefuses` "missing folder" only asserts error contains `nope`; would pass on unrelated errors `internal/rcservice/add_test.go`

## Suggested improvements

- show why a unit keeps restarting in `status`: restart count and last exit code next to the state, and a hint to `journalctl --user -u claude-rc-<name>`
- split `Remove`'s single "unknown" error into "bad name", "not set up" and "can't read unit folder"; in `Add`, only treat `os.ErrNotExist` as "free"
- in `Status`, show `unknown` for a unit whose state can't be read instead of failing the whole table
- add `rcservice.New(...)` so `Runner` and `LookPath` can't be left nil
- move file writes/deletes behind the runner (or a tiny files interface) so dry run is a single swap and the `if dryRun` branches go away
- rename `unitFile` → `unitFileContent`, `checkRepo` → `resolveRepo`, consider `Service` → `Manager` and `rcservice` → `remotecontrol`
- reject newlines in the repo path; quote `ExecStart` path or refuse paths with spaces
- compare home with `filepath.EvalSymlinks` on both sides
- move the AGENTS.md rule changes and `fsinfo_test.go` comments to their own small PR
