# ae-init: review-error-handling — claude-rc says why something is wrong

> **Status:** Draft
> **Last updated:** 2026-10-10
> **From:** ae-review `pr-1`, section "error handling"

## What to build

Make `claude-rc`'s failures visible and specific. `status` should show when a repo's service is stuck restarting (not trusted, logged out, no network), how often it restarted, how it last exited, and where to look for the reason. One unreadable unit shouldn't hide the whole table. `add` and `remove` should give a different message for each different cause, and a half-done `add` should clean up after itself.

## A -> B

- **A (now):**
  - `status` asks `systemctl is-active`. With `Restart=always` a unit that fails every start never shows `failed`; between tries it shows `activating`, which looks like "starting up".
  - If `systemctl` fails for one unit, `status` stops and prints nothing for the others.
  - `remove` answers "no repo set up as …" for a bad name, a missing unit and an unreadable unit folder alike.
  - `add` treats any error checking for an existing unit as "not there yet".
  - If `systemctl enable --now` fails, `add` leaves the unit file behind and only says how to undo it.
- **B (after):**
  - `status` shows `restarting`, the restart count and the last exit status, plus a `journalctl` line for each repo that isn't running.
  - A unit whose state can't be read shows as `unknown` with the reason; the other rows still show.
  - `remove` says "not a valid name", "no repo set up as …" or "can't check …" depending on the cause.
  - `add` stops with "can't check …" when it can't tell whether the unit exists.
  - A failed `add` removes the unit again and says so.

## How to reproduce

**Preconditions**
- Ubuntu with systemd, Claude logged in, binary from `make build-linux` on `review/3-simplicity` (A) or `review/4-error-handling` (B).
- One repo that was never trusted (no manual `claude remote-control` run), e.g. `~/workspace/untrusted`.

**Steps**
1. `claude-rc add ~/workspace/untrusted`, wait 2 minutes, `claude-rc status`.
2. `claude-rc remove ../x`, then `claude-rc remove nope`.
3. `chmod 000 ~/.config/systemd/user`, `claude-rc remove untrusted`, `claude-rc add ~/workspace/untrusted`, then `chmod 755 ~/.config/systemd/user`.
4. Stop the user manager's D-Bus so `systemctl --user` fails (or test with the fake runner), then `claude-rc add ~/workspace/other`.

**Observe**

| | A (now) | B (after) |
|---|---|---|
| `status` (step 1) | `untrusted  activating  …` | `untrusted  restarting  <n>  <exit>  …` and `see why untrusted isn't running: journalctl --user -u claude-rc-untrusted` |
| `remove ../x` / `remove nope` (step 2) | Same "no repo set up" for both | "not a valid name" / "no repo set up as \"nope\"" |
| Unreadable folder (step 3) | `remove`: "no repo set up"; `add`: write error | Both: "can't check …: permission denied" |
| `systemctl` fails during `add` (step 4) | Error + "to undo, run: claude-rc remove other"; unit file left | Error + "the unit was removed again"; no unit file |

## Open questions

- Does `systemctl show` report `SubState=auto-restart` while waiting for the next try? Expected from systemd's service states, not stated in the man pages read. Check on Ubuntu; if not, `restarting` won't appear but the restart count and last exit still will.
