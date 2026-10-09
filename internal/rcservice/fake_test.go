package rcservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type reply struct {
	out string
	err error
}

type fakeRunner struct {
	replies map[string]reply
	calls   []string
}

func (f *fakeRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, cmd)
	r := f.replies[cmd]
	return []byte(r.out), r.err
}

func writeUnit(t *testing.T, dir, name, repo string) {
	t.Helper()
	content := "[Service]\nWorkingDirectory=" + repo + "\n"
	if err := os.WriteFile(filepath.Join(dir, "claude-rc-"+name+".service"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
