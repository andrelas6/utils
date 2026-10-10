package rcservice

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const authCmd = "claude auth status --json"

func isActiveCmd(name string) string {
	return "systemctl --user is-active claude-rc-" + name + ".service"
}

// Two set-up repos are listed sorted by name, each with the state systemd
// reports and its repo folder; files that aren't claude-rc units are ignored.
func TestStatusListsUnits(t *testing.T) {
	dir := t.TempDir()
	writeUnit(t, dir, "web", "/home/a/web")
	writeUnit(t, dir, "utils", "/home/a/utils")
	if err := os.WriteFile(filepath.Join(dir, "other.service"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{replies: map[string]reply{
		isActiveCmd("utils"): {out: "active\n"},
		isActiveCmd("web"):   {out: "failed\n", err: errors.New("exit status 3")},
		authCmd:              {out: `{"loggedIn": true}`},
	}}
	var out bytes.Buffer

	if err := (&Service{UnitDir: dir, Runner: runner}).Status(&out); err != nil {
		t.Fatal(err)
	}

	want := "" +
		"NAME   STATE   REPO\n" +
		"utils  active  /home/a/utils\n" +
		"web    failed  /home/a/web\n" +
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
	unit := unitFile("/home/a/My Repo%", "My-Repo", "/usr/bin/claude")
	if err := os.WriteFile(filepath.Join(dir, unitFileName("My-Repo")), []byte(unit), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{replies: map[string]reply{
		isActiveCmd("My-Repo"): {out: "active\n"},
	}}
	var out bytes.Buffer

	if err := (&Service{UnitDir: dir, Runner: runner}).Status(&out); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "/home/a/My Repo%\n") {
		t.Errorf("repo not shown as /home/a/My Repo%%:\n%s", out.String())
	}
}

// With no units, or no unit folder at all, status says nothing is set up
// and still reports the login.
func TestStatusNoUnits(t *testing.T) {
	for _, dir := range []string{t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		runner := &fakeRunner{replies: map[string]reply{authCmd: {out: `{"loggedIn": true}`}}}
		var out bytes.Buffer

		if err := (&Service{UnitDir: dir, Runner: runner}).Status(&out); err != nil {
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
			runner := &fakeRunner{replies: map[string]reply{authCmd: tt.reply}}
			var out bytes.Buffer

			if err := (&Service{UnitDir: t.TempDir(), Runner: runner}).Status(&out); err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(out.String(), tt.want) {
				t.Errorf("got %q, want it to end with %q", out.String(), tt.want)
			}
		})
	}
}

// When systemctl fails without printing a state, status returns an error that
// names the unit.
func TestStatusSystemctlError(t *testing.T) {
	dir := t.TempDir()
	writeUnit(t, dir, "utils", "/home/a/utils")
	runner := &fakeRunner{replies: map[string]reply{
		isActiveCmd("utils"): {err: errors.New("systemctl not found")},
	}}

	err := (&Service{UnitDir: dir, Runner: runner}).Status(&bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "claude-rc-utils.service") {
		t.Fatalf("err = %v, want it to name claude-rc-utils.service", err)
	}
}
