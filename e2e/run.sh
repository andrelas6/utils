#!/usr/bin/env bash
set -u

here=$(cd "$(dirname "$0")" && pwd)
claude_rc="$here/../bin/claude-rc"
unit_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
work=$(mktemp -d "${RUNNER_TEMP:-/tmp}/claude-rc-e2e.XXXXXX")
calls="$work/bin/calls.log"

mkdir -p "$work/bin" "$work/healthy"
cp "$here/fake-claude" "$work/bin/claude"
chmod +x "$work/bin/claude"
export PATH="$work/bin:$PATH"

failures=0

check() {
	local label=$1
	shift
	if "$@"; then
		echo "PASS  $label"
	else
		echo "FAIL  $label"
		failures=$((failures + 1))
	fi
}

show_state_if_failed() {
	if [ "$failures" -gt 0 ]; then
		echo
		echo "== systemd user units"
		systemctl --user list-units --all --no-pager 'claude-rc-*' || true
		echo
		echo "== user journal"
		journalctl --user --no-pager -n 200 || true
	fi
}
trap show_state_if_failed EXIT

wait_for() {
	local seconds=$1
	shift
	for _ in $(seq 1 "$seconds"); do
		"$@" && return 0
		sleep 1
	done
	"$@"
}

run_claude_rc() {
	echo "\$ claude-rc $*"
	"$claude_rc" "$@" 2>&1 | tee "$work/last-output.txt"
	return "${PIPESTATUS[0]}"
}

last_output_has() {
	grep -qE "$1" "$work/last-output.txt"
}

is_active() {
	[ "$(systemctl --user is-active "$1")" = "active" ]
}

restarts_of() {
	systemctl --user show --property=NRestarts --value "$1"
}

came_back() {
	is_active "$1" && [ "$(restarts_of "$1")" -ge 1 ]
}

user_manager_answers() {
	systemctl --user list-units --no-pager >/dev/null
}

check "systemctl --user answers" user_manager_answers

echo
echo "== healthy repo"
check "add succeeds" run_claude_rc add "$work/healthy"
check "unit file is written" test -f "$unit_dir/claude-rc-healthy.service"
check "unit becomes active" wait_for 30 is_active claude-rc-healthy.service
check "claude runs in the repo with the right arguments" \
	wait_for 10 grep -qxF "$work/healthy remote-control --name $(uname -n)-healthy --permission-mode default" "$calls"
check "status succeeds" run_claude_rc status
check "status shows it active with no restarts" last_output_has "^healthy +active +0 +0 +$work/healthy\$"
check "status says logged in" last_output_has "^logged in: yes\$"

echo
echo "== restart after a crash"
check "the claude process can be killed" systemctl --user kill --signal=SIGKILL claude-rc-healthy.service
check "systemd starts it again within 90s" wait_for 90 came_back claude-rc-healthy.service
check "status succeeds" run_claude_rc status
check "status shows it active again with a restart" last_output_has "^healthy +active +[1-9][0-9]* "

echo
echo "$failures failed"
[ "$failures" -eq 0 ]
