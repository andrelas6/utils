package remotecontrol

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const authCmd = "claude auth status --json"

func showCmd(name string) string {
	return "systemctl --user show claude-rc-" + name + ".service --property=ActiveState,SubState,NRestarts,ExecMainStatus"
}

func showReply(active, sub, restarts, exit string) reply {
	return reply{out: "ActiveState=" + active + "\nSubState=" + sub + "\nNRestarts=" + restarts + "\nExecMainStatus=" + exit + "\n"}
}

// Two set-up repos are listed sorted by name, each with its state, restart
// count, last exit status and repo folder, and the one that isn't running gets
// a line saying where to look; files that aren't claude-rc units are ignored.
func TestStatusListsUnits(t *testing.T) {
	dir := t.TempDir()
	writeUnit(t, dir, "web", "/home/a/web")
	writeUnit(t, dir, "utils", "/home/a/utils")
	if err := os.WriteFile(filepath.Join(dir, "other.service"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeMachine{replies: map[string]reply{
		showCmd("utils"): showReply("active", "running", "0", "0"),
		showCmd("web"):   showReply("failed", "failed", "5", "1"),
		authCmd:          {out: `{"loggedIn": true}`},
	}}
	var out bytes.Buffer

	if err := (&Manager{unitDir: dir, machine: runner}).Status(&out); err != nil {
		t.Fatal(err)
	}

	want := "" +
		"NAME   STATE   RESTARTS  LAST EXIT  REPO\n" +
		"utils  active  0         0          /home/a/utils\n" +
		"web    failed  5         1          /home/a/web\n" +
		"\n" +
		"see why web isn't running: journalctl --user -u claude-rc-web\n" +
		"\n" +
		"logged in: yes\n"
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

// A repo path with a % is shown as the real path, not with the doubled %%
// that the unit file needs for systemd.
func TestStatusUnescapesPercent(t *testing.T) {
	dir := t.TempDir()
	unit := unitFileContent("/home/a/My Repo%", "My-Repo", "/usr/bin/claude")
	if err := os.WriteFile(filepath.Join(dir, unitFileName("My-Repo")), []byte(unit), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeMachine{replies: map[string]reply{
		showCmd("My-Repo"): showReply("active", "running", "0", "0"),
	}}
	var out bytes.Buffer

	if err := (&Manager{unitDir: dir, machine: runner}).Status(&out); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "/home/a/My Repo%\n") {
		t.Errorf("repo not shown as /home/a/My Repo%%:\n%s", out.String())
	}
}

// A unit that keeps failing and is waiting for its next try shows as
// restarting with its restart count and last exit status, not as starting up.
func TestStatusRestarting(t *testing.T) {
	dir := t.TempDir()
	writeUnit(t, dir, "untrusted", "/home/a/untrusted")
	runner := &fakeMachine{replies: map[string]reply{
		showCmd("untrusted"): showReply("activating", "auto-restart", "4", "1"),
	}}
	var out bytes.Buffer

	if err := (&Manager{unitDir: dir, machine: runner}).Status(&out); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"untrusted  restarting  4         1          /home/a/untrusted\n",
		"see why untrusted isn't running: journalctl --user -u claude-rc-untrusted\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output is missing %q:\n%s", want, out.String())
		}
	}
}

// With no units, or no unit folder at all, status says nothing is set up
// and still reports the login.
func TestStatusNoUnits(t *testing.T) {
	for _, dir := range []string{t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		runner := &fakeMachine{replies: map[string]reply{authCmd: {out: `{"loggedIn": true}`}}}
		var out bytes.Buffer

		if err := (&Manager{unitDir: dir, machine: runner}).Status(&out); err != nil {
			t.Fatal(err)
		}

		want := "no repos set up\n\nlogged in: yes\n"
		if out.String() != want {
			t.Errorf("got %q, want %q", out.String(), want)
		}
	}
}

// The login line says yes or no from claude's JSON, and unknown when claude
// fails or prints something that isn't the expected JSON.
func TestStatusLogin(t *testing.T) {
	tests := []struct {
		name  string
		reply reply
		want  string
	}{
		{"logged in", reply{out: `{"loggedIn": true, "authMethod": "claude.ai"}`}, "logged in: yes\n"},
		{"logged out", reply{out: `{"loggedIn": false}`, err: errors.New("exit status 1")}, "logged in: no\n"},
		{"claude missing", reply{err: errors.New("executable file not found")}, "logged in: unknown\n"},
		{"not json", reply{out: "oops"}, "logged in: unknown\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeMachine{replies: map[string]reply{authCmd: tt.reply}}
			var out bytes.Buffer

			if err := (&Manager{unitDir: t.TempDir(), machine: runner}).Status(&out); err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(out.String(), tt.want) {
				t.Errorf("got %q, want it to end with %q", out.String(), tt.want)
			}
		})
	}
}

// When systemctl can't read one unit, that row shows unknown with the reason
// underneath, and the other repos are still listed.
func TestStatusUnreadableUnit(t *testing.T) {
	dir := t.TempDir()
	writeUnit(t, dir, "utils", "/home/a/utils")
	writeUnit(t, dir, "web", "/home/a/web")
	runner := &fakeMachine{replies: map[string]reply{
		showCmd("utils"): {err: errors.New("Failed to connect to bus")},
		showCmd("web"):   showReply("active", "running", "0", "0"),
	}}
	var out bytes.Buffer

	if err := (&Manager{unitDir: dir, machine: runner}).Status(&out); err != nil {
		t.Fatalf("status failed: %v", err)
	}

	rows := map[string][]string{}
	for line := range strings.Lines(out.String()) {
		if fields := strings.Fields(line); len(fields) == 5 {
			rows[fields[0]] = fields
		}
	}
	if got := rows["utils"]; !slices.Equal(got, []string{"utils", "unknown", "?", "?", "/home/a/utils"}) {
		t.Errorf("utils row = %q", got)
	}
	if got := rows["web"]; !slices.Equal(got, []string{"web", "active", "0", "0", "/home/a/web"}) {
		t.Errorf("web row = %q", got)
	}
	if !strings.Contains(out.String(), "can't read utils's state: systemctl --user show claude-rc-utils.service") ||
		!strings.Contains(out.String(), "Failed to connect to bus") {
		t.Errorf("no reason shown for utils:\n%s", out.String())
	}
}
