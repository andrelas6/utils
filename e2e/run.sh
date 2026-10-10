#!/usr/bin/env bash
set -u

here=$(cd "$(dirname "$0")" && pwd)
claude_rc="$here/../bin/claude-rc"
unit_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
work=$(mktemp -d "${RUNNER_TEMP:-/tmp}/claude-rc-e2e.XXXXXX")
calls="$work/bin/calls.log"

mkdir -p "$work/bin" "$work/healthy" "$work/broken"
touch "$work/broken/.fail"
line_break_repo="$work/two"$'\n'"lines"
mkdir -p "$line_break_repo"
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

keeps_failing() {
	! is_active "$1" && [ "$(systemctl --user show --property=ExecMainStatus --value "$1")" = "1" ]
}

refused() {
	! run_claude_rc "$@"
}

unit_count() {
	find "$unit_dir" -maxdepth 1 -name 'claude-rc-*.service' 2>/dev/null | wc -l | tr -d ' '
}

has_units() {
	[ "$(unit_count)" = "$1" ]
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
echo "== repo where claude keeps exiting"
check "add succeeds" run_claude_rc add "$work/broken"
check "claude exits with 1 and the unit isn't active" wait_for 30 keeps_failing claude-rc-broken.service
echo "systemd says: $(systemctl --user show --property=ActiveState,SubState,NRestarts claude-rc-broken.service | tr '\n' ' ')"
check "status succeeds" run_claude_rc status
check "status shows it restarting or failed, last exit 1" last_output_has "^broken +(restarting|activating|failed) +[0-9]+ +1 +"
check "status says where to look" last_output_has "^see why broken isn't running: journalctl --user -u claude-rc-broken\$"

echo
echo "== refusals"
check "add refuses the home folder" refused add "$HOME"
check "  and says it's the home folder" last_output_has "home folder"
check "add refuses a repo that's already set up" refused add "$work/healthy"
check "  and says it's already set up" last_output_has "already set up"
check "add refuses a path with a line break" refused add "$line_break_repo"
check "  and says it has a line break" last_output_has "line break"
check "remove refuses a name pointing outside the unit folder" refused remove ../healthy
check "  and says it isn't a valid name" last_output_has "isn't a valid name"
check "remove refuses a name that isn't set up" refused remove nope
check "  and says it isn't set up" last_output_has 'no repo set up as "nope"'
check "still exactly two units" has_units 2

echo
echo "$failures failed"
[ "$failures" -eq 0 ]
