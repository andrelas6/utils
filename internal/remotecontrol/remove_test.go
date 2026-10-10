package remotecontrol

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newManagerWithUnit(t *testing.T, name string) (*Manager, *fakeMachine, string) {
	t.Helper()
	svc, runner := newTestManager(t)
	if err := os.MkdirAll(svc.unitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeUnit(t, svc.unitDir, name, "/home/a/"+name)
	return svc, runner, filepath.Join(svc.unitDir, unitFileName(name))
}

// Removing a set-up repo stops and disables its unit, deletes the unit file,
// then reloads systemd.
func TestRemove(t *testing.T) {
	svc, runner, unitPath := newManagerWithUnit(t, "utils")
	var out bytes.Buffer

	if err := svc.Remove(&out, "utils", false); err != nil {
		t.Fatal(err)
	}

	wantCalls := []string{
		"systemctl --user disable --now claude-rc-utils.service",
		"systemctl --user daemon-reload",
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Errorf("calls = %q, want %q", runner.calls, wantCalls)
	}
	if _, err := os.Stat(unitPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("unit file still there")
	}
	if !strings.Contains(out.String(), "removed utils") {
		t.Errorf("out = %q", out.String())
	}
}

// Each reason remove can't go ahead gets its own message: a name that tries to
// point outside the unit folder, a name that isn't set up, and a unit folder
// that can't be checked. None of them touch anything.
func TestRemoveRefuses(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, m *Manager)
		want  string
	}{
		{"../utils", nil, `"../utils" isn't a valid name`},
		{"web", nil, `no repo set up as "web", see claude-rc status`},
		{"claude-rc-utils.service", nil, `no repo set up as "claude-rc-utils.service"`},
		{"utils", func(t *testing.T, m *Manager) {
			if err := os.Chmod(m.unitDir, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(m.unitDir, 0o755) })
		}, "can't check"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, runner, unitPath := newManagerWithUnit(t, "utils")
			if tt.setup != nil {
				tt.setup(t, svc)
			}

			err := svc.Remove(&bytes.Buffer{}, tt.name, false)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.want)
			}
			if len(runner.calls) != 0 {
				t.Errorf("systemctl called: %q", runner.calls)
			}
			os.Chmod(svc.unitDir, 0o755)
			if _, err := os.Stat(unitPath); err != nil {
				t.Errorf("existing unit file was touched: %v", err)
			}
		})
	}
}

// When systemctl can't disable the unit, remove stops there: the unit file
// is kept and systemd isn't reloaded.
func TestRemoveDisableFails(t *testing.T) {
	svc, runner, unitPath := newManagerWithUnit(t, "utils")
	runner.replies = map[string]reply{
		"systemctl --user disable --now claude-rc-utils.service": {err: errors.New("Failed to connect to bus")},
	}

	err := svc.Remove(&bytes.Buffer{}, "utils", false)
	if err == nil || !strings.Contains(err.Error(), "Failed to connect to bus") {
		t.Fatalf("err = %v, want systemctl's message", err)
	}
	if _, err := os.Stat(unitPath); err != nil {
		t.Errorf("unit file was deleted: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Errorf("calls = %q, want only the disable", runner.calls)
	}
}

// A dry run prints the systemctl commands and the file it would delete, but
// keeps the unit file and runs nothing.
func TestRemoveDryRun(t *testing.T) {
	svc, runner, unitPath := newManagerWithUnit(t, "utils")
	var out bytes.Buffer

	if err := svc.Remove(&out, "utils", true); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(unitPath); err != nil {
		t.Errorf("unit file was deleted: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Errorf("systemctl called: %q", runner.calls)
	}
	want := "" +
		"would run: systemctl --user disable --now claude-rc-utils.service\n" +
		"would delete: " + unitPath + "\n" +
		"would run: systemctl --user daemon-reload\n"
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

// A dry run for a name that isn't set up gives the same error as a real remove.
func TestRemoveDryRunUnknown(t *testing.T) {
	svc, _, _ := newManagerWithUnit(t, "utils")

	err := svc.Remove(&bytes.Buffer{}, "web", true)
	if err == nil || !strings.Contains(err.Error(), "claude-rc status") {
		t.Fatalf("err = %v, want the unknown-name error", err)
	}
}
