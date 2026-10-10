package remotecontrol

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The dry-run machine prints the command it was given and runs nothing.
func TestDryRunMachineRun(t *testing.T) {
	var out strings.Builder

	got, err := dryRunMachine{out: &out}.Run("systemctl", "--user", "daemon-reload")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %q, %v; want no output and no error", got, err)
	}
	if out.String() != "would run: systemctl --user daemon-reload\n" {
		t.Errorf("printed %q", out.String())
	}
}

// The dry-run machine prints the file it would write and its content, and
// creates nothing on disk.
func TestDryRunMachineWriteFile(t *testing.T) {
	var out strings.Builder
	path := filepath.Join(t.TempDir(), "user", "claude-rc-utils.service")

	if err := (dryRunMachine{out: &out}).WriteFile(path, []byte("[Unit]\n")); err != nil {
		t.Fatal(err)
	}
	if want := "would write: " + path + "\n\n[Unit]\n\n"; out.String() != want {
		t.Errorf("printed %q, want %q", out.String(), want)
	}
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Error("folder was created")
	}
}

// The dry-run machine prints the file it would delete and leaves it in place.
func TestDryRunMachineRemove(t *testing.T) {
	var out strings.Builder
	path := filepath.Join(t.TempDir(), "claude-rc-utils.service")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := (dryRunMachine{out: &out}).Remove(path); err != nil {
		t.Fatal(err)
	}
	if out.String() != "would delete: "+path+"\n" {
		t.Errorf("printed %q", out.String())
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file was deleted: %v", err)
	}
}

// The real machine creates the missing folder before writing the file.
func TestRealMachineWriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "systemd", "user", "claude-rc-utils.service")

	if err := (realMachine{}).WriteFile(path, []byte("[Unit]\n")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "[Unit]\n" {
		t.Errorf("file = %q, %v", got, err)
	}
}

// The real machine deletes the file it's given.
func TestRealMachineRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-rc-utils.service")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := (realMachine{}).Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file still there: %v", err)
	}
}

// When a program fails, the error includes what it printed on stderr, so the
// user sees why systemctl failed.
func TestRealMachineRunErrorHasStderr(t *testing.T) {
	_, err := realMachine{}.Run("sh", "-c", "echo 'Failed to connect to bus' >&2; exit 1")
	if err == nil || !strings.Contains(err.Error(), "Failed to connect to bus") {
		t.Fatalf("err = %v, want it to include stderr", err)
	}
}

// The real machine returns what the program printed even when it exits with
// an error, like systemctl is-active does for a stopped unit.
func TestRealMachineRunKeepsOutputOnError(t *testing.T) {
	out, err := realMachine{}.Run("sh", "-c", "echo failed; exit 3")
	if err == nil {
		t.Fatal("expected an error")
	}
	if string(out) != "failed\n" {
		t.Errorf("out = %q, want %q", out, "failed\n")
	}
}
