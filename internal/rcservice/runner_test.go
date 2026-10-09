package rcservice

import (
	"strings"
	"testing"
)

// The dry-run runner prints the command it was given and runs nothing.
func TestDryRunner(t *testing.T) {
	var out strings.Builder

	got, err := DryRunner{Out: &out}.Run("systemctl", "--user", "daemon-reload")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %q, %v; want no output and no error", got, err)
	}
	if out.String() != "would run: systemctl --user daemon-reload\n" {
		t.Errorf("printed %q", out.String())
	}
}

// When a program fails, the error includes what it printed on stderr, so the
// user sees why systemctl failed.
func TestExecRunnerErrorHasStderr(t *testing.T) {
	_, err := ExecRunner{}.Run("sh", "-c", "echo 'Failed to connect to bus' >&2; exit 1")
	if err == nil || !strings.Contains(err.Error(), "Failed to connect to bus") {
		t.Fatalf("err = %v, want it to include stderr", err)
	}
}

// The real runner returns what the program printed even when it exits with an
// error, like systemctl is-active does for a stopped unit.
func TestExecRunnerKeepsOutputOnError(t *testing.T) {
	out, err := ExecRunner{}.Run("sh", "-c", "echo failed; exit 3")
	if err == nil {
		t.Fatal("expected an error")
	}
	if string(out) != "failed\n" {
		t.Errorf("out = %q, want %q", out, "failed\n")
	}
}
