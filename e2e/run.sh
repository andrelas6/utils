#!/usr/bin/env bash
set -u

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

user_manager_answers() {
	systemctl --user list-units --no-pager >/dev/null
}

check "systemctl --user answers" user_manager_answers

echo
echo "$failures failed"
[ "$failures" -eq 0 ]
