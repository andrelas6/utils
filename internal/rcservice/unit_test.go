package rcservice

import (
	"strings"
	"testing"
)

// The unit file for a repo runs claude remote-control in that repo with the
// host and name as the session name, default permissions, a restart every
// minute, and starts at login.
func TestUnitFileContent(t *testing.T) {
	got := unitFileContent("/home/a/work/utils", "utils", "/home/a/.local/bin/claude")

	want := `[Unit]
Description=Claude Remote Control for utils (/home/a/work/utils)

[Service]
WorkingDirectory=/home/a/work/utils
ExecStart=/home/a/.local/bin/claude remote-control --name %H-utils --permission-mode default
Restart=always
RestartSec=60

[Install]
WantedBy=default.target
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// A % in a path is doubled so systemd doesn't read it as one of its own
// placeholders like %H.
func TestUnitFileContentEscapesPercent(t *testing.T) {
	got := unitFileContent("/home/a/100%", "x", "/opt/50%/claude")

	for _, want := range []string{
		"WorkingDirectory=/home/a/100%%\n",
		"ExecStart=/opt/50%%/claude remote-control --name %H-x ",
		"(/home/a/100%%)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("unit file is missing %q:\n%s", want, got)
		}
	}
}

// The name comes from the repo folder unless one is given, and anything that
// isn't a letter, digit, dot, dash or underscore becomes a single dash.
func TestUnitName(t *testing.T) {
	tests := []struct {
		repo, name, want string
	}{
		{"/home/a/work/utils", "", "utils"},
		{"/home/a/work/utils/", "", "utils"},
		{"/x/My Repo!", "", "My-Repo"},
		{"/x/a  &  b", "", "a-b"},
		{"/x/my_repo.v2", "", "my_repo.v2"},
		{"/home/a/work/utils", "tools", "tools"},
		{"/home/a/work/utils", "my tools%", "my-tools"},
	}

	for _, tt := range tests {
		got, err := unitName(tt.repo, tt.name)
		if err != nil {
			t.Errorf("unitName(%q, %q): %v", tt.repo, tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("unitName(%q, %q) = %q, want %q", tt.repo, tt.name, got, tt.want)
		}
	}
}

// When nothing usable is left after cleaning, there is no name and it's an error.
func TestUnitNameEmpty(t *testing.T) {
	tests := []struct{ repo, name string }{
		{"/x/!!!", ""},
		{"/", ""},
		{"/home/a/utils", "%%%"},
	}

	for _, tt := range tests {
		if got, err := unitName(tt.repo, tt.name); err == nil {
			t.Errorf("unitName(%q, %q) = %q, want an error", tt.repo, tt.name, got)
		}
	}
}

// The unit file name wraps the name in the claude-rc prefix and .service.
func TestUnitFileName(t *testing.T) {
	if got := unitFileName("utils"); got != "claude-rc-utils.service" {
		t.Errorf("got %q", got)
	}
}
