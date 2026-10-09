package rcservice

import "testing"

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
