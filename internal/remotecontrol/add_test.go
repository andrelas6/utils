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

const claudePath = "/home/a/.local/bin/claude"

func newTestManager(t *testing.T) (*Manager, *fakeRunner) {
	t.Helper()
	runner := &fakeRunner{}
	return &Manager{
		unitDir: filepath.Join(t.TempDir(), "systemd", "user"),
		home:    t.TempDir(),
		runner:  runner,
		lookPath: func(name string) (string, error) {
			if name != "claude" {
				return "", errors.New("unexpected lookup " + name)
			}
			return claudePath, nil
		},
	}, runner
}

// Adding a repo writes its unit file, reloads systemd, then enables and
// starts the unit, and reminds the user about the one-time prompts.
func TestAdd(t *testing.T) {
	svc, runner := newTestManager(t)
	repo := t.TempDir()
	var out bytes.Buffer

	if err := svc.Add(&out, repo, "utils", false); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(svc.unitDir, "claude-rc-utils.service"))
	if err != nil {
		t.Fatal(err)
	}
	if want := unitFileContent(repo, "utils", claudePath); string(got) != want {
		t.Errorf("unit file:\n%s\nwant:\n%s", got, want)
	}

	wantCalls := []string{
		"systemctl --user daemon-reload",
		"systemctl --user enable --now claude-rc-utils.service",
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Errorf("calls = %q, want %q", runner.calls, wantCalls)
	}
	if !strings.Contains(out.String(), "claude remote-control") {
		t.Errorf("output has no reminder about the first manual run:\n%s", out.String())
	}
}

type outputAtFirstCall struct {
	out  *bytes.Buffer
	seen *string
}

func (o outputAtFirstCall) Run(string, ...string) ([]byte, error) {
	if *o.seen == "" {
		*o.seen = o.out.String()
	}
	return nil, nil
}

// The reminder to run claude remote-control once by hand is printed before
// systemctl starts the unit, so the user reads it before an untrusted repo fails.
func TestAddRemindsBeforeStarting(t *testing.T) {
	svc, _ := newTestManager(t)
	var out bytes.Buffer
	var beforeSystemctl string
	svc.runner = outputAtFirstCall{out: &out, seen: &beforeSystemctl}

	if err := svc.Add(&out, t.TempDir(), "utils", false); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(beforeSystemctl, "claude remote-control") {
		t.Errorf("reminder not printed before systemctl ran; output then was:\n%q", beforeSystemctl)
	}
}

// A dry run shows the reminder too, before the commands it would run.
func TestAddDryRunReminds(t *testing.T) {
	svc, _ := newTestManager(t)
	var out bytes.Buffer

	if err := svc.Add(&out, t.TempDir(), "utils", true); err != nil {
		t.Fatal(err)
	}

	reminder := strings.Index(out.String(), "claude remote-control` once")
	firstCommand := strings.Index(out.String(), "would run:")
	if reminder == -1 || reminder > firstCommand {
		t.Errorf("reminder missing or after the commands:\n%s", out.String())
	}
}

// A relative repo path like "." is saved as the full path, since systemd
// needs an absolute WorkingDirectory.
func TestAddRelativeRepo(t *testing.T) {
	svc, _ := newTestManager(t)
	t.Chdir(t.TempDir())
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Add(&bytes.Buffer{}, ".", "here", false); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(svc.unitDir, "claude-rc-here.service"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "WorkingDirectory="+wd+"\n") {
		t.Errorf("unit file doesn't use %s:\n%s", wd, got)
	}
}

// Bad input is refused before anything changes: no unit file is written and
// systemctl is never called.
func TestAddRefuses(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, svc *Manager) (repo, name string)
		want  string
	}{
		{"missing folder", func(t *testing.T, svc *Manager) (string, string) {
			return filepath.Join(t.TempDir(), "nope"), ""
		}, "nope"},
		{"file, not folder", func(t *testing.T, svc *Manager) (string, string) {
			file := filepath.Join(t.TempDir(), "notes.txt")
			if err := os.WriteFile(file, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			return file, ""
		}, "not a folder"},
		{"home folder", func(t *testing.T, svc *Manager) (string, string) {
			return svc.home, "home"
		}, "home folder"},
		{"no usable name", func(t *testing.T, svc *Manager) (string, string) {
			return t.TempDir(), "%%%"
		}, "-name"},
		{"claude not found", func(t *testing.T, svc *Manager) (string, string) {
			svc.lookPath = func(string) (string, error) { return "", errors.New("not found") }
			return t.TempDir(), "utils"
		}, "claude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, runner := newTestManager(t)
			repo, name := tt.setup(t, svc)

			err := svc.Add(&bytes.Buffer{}, repo, name, false)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
			if _, statErr := os.Stat(svc.unitDir); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("unit folder was created")
			}
			if len(runner.calls) != 0 {
				t.Errorf("systemctl called: %q", runner.calls)
			}
		})
	}
}

// Adding a name that is already set up is refused and leaves the existing
// unit file as it was.
func TestAddExistingUnit(t *testing.T) {
	svc, runner := newTestManager(t)
	if err := os.MkdirAll(svc.unitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeUnit(t, svc.unitDir, "utils", "/old/repo")

	err := svc.Add(&bytes.Buffer{}, t.TempDir(), "utils", false)
	if err == nil || !strings.Contains(err.Error(), "already set up") {
		t.Fatalf("err = %v, want 'already set up'", err)
	}
	got, _ := os.ReadFile(filepath.Join(svc.unitDir, "claude-rc-utils.service"))
	if !strings.Contains(string(got), "/old/repo") {
		t.Errorf("existing unit file was changed:\n%s", got)
	}
	if len(runner.calls) != 0 {
		t.Errorf("systemctl called: %q", runner.calls)
	}
}

// When systemctl fails, add returns an error naming the command and tells the
// user how to undo the half-done setup.
func TestAddSystemctlFails(t *testing.T) {
	svc, runner := newTestManager(t)
	runner.replies = map[string]reply{
		"systemctl --user enable --now claude-rc-utils.service": {err: errors.New("Failed to connect to bus")},
	}

	err := svc.Add(&bytes.Buffer{}, t.TempDir(), "utils", false)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"enable --now claude-rc-utils.service", "Failed to connect to bus", "claude-rc remove utils"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want it to mention %q", err, want)
		}
	}
}

// A dry run prints the unit file it would write and the systemctl commands it
// would run, but writes nothing and runs nothing.
func TestAddDryRun(t *testing.T) {
	svc, runner := newTestManager(t)
	repo := t.TempDir()
	var out bytes.Buffer

	if err := svc.Add(&out, repo, "utils", true); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(svc.unitDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("unit folder was created")
	}
	if len(runner.calls) != 0 {
		t.Errorf("systemctl called: %q", runner.calls)
	}
	for _, want := range []string{
		filepath.Join(svc.unitDir, "claude-rc-utils.service"),
		unitFileContent(repo, "utils", claudePath),
		"would run: systemctl --user daemon-reload\n",
		"would run: systemctl --user enable --now claude-rc-utils.service\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output is missing %q:\n%s", want, out.String())
		}
	}
}

// A dry run still checks the input, so it shows the same refusals a real
// add would give.
func TestAddDryRunStillRefuses(t *testing.T) {
	svc, _ := newTestManager(t)

	err := svc.Add(&bytes.Buffer{}, svc.home, "home", true)
	if err == nil || !strings.Contains(err.Error(), "home folder") {
		t.Fatalf("err = %v, want the home folder refusal", err)
	}
}
