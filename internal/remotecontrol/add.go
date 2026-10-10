package remotecontrol

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) Add(out io.Writer, repo, name string, dryRun bool) error {
	repo, err := m.resolveRepo(repo)
	if err != nil {
		return err
	}
	name, err = unitName(repo, name)
	if err != nil {
		return err
	}

	unit := unitFileName(name)
	unitPath := filepath.Join(m.unitDir, unit)
	if _, err := os.Stat(unitPath); err == nil {
		return fmt.Errorf("%s is already set up (%s), remove it first", name, unitPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("can't check %s: %w", unitPath, err)
	}

	claude, err := m.lookPath("claude")
	if err != nil {
		return fmt.Errorf("can't find claude: %w", err)
	}
	if strings.ContainsAny(claude, " \t\n\r") {
		return fmt.Errorf("claude is at %q; a space or line break in that path breaks the unit's ExecStart, so put claude (or a link to it) in a folder without one", claude)
	}

	content := unitFileContent(repo, name, claude)
	mc := m.machineFor(out, dryRun)
	if err := mc.WriteFile(unitPath, []byte(content)); err != nil {
		return fmt.Errorf("write %q: %w", unitPath, err)
	}

	fmt.Fprintf(out, "first time in this repo? run `claude remote-control` once in %s to answer the trust and enable prompts\n", repo)
	for _, args := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "--now", unit},
	} {
		if _, err := mc.Run("systemctl", args...); err != nil {
			failed := fmt.Errorf("systemctl %s: %w", strings.Join(args, " "), err)
			if m.Remove(io.Discard, name, dryRun) != nil {
				return fmt.Errorf("%w (to undo, run: claude-rc remove %s)", failed, name)
			}
			return fmt.Errorf("%w (the unit was removed again)", failed)
		}
	}

	if dryRun {
		return nil
	}
	fmt.Fprintf(out, "added %s: %s\n", name, unitPath)
	fmt.Fprintln(out, "check it with: claude-rc status")
	return nil
}

func (m *Manager) resolveRepo(repo string) (string, error) {
	if strings.ContainsAny(repo, "\n\r") {
		return "", fmt.Errorf("%q has a line break in it, which a systemd unit can't hold; rename the folder", repo)
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", repo, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("check repo: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", abs)
	}
	if sameFolder(abs, m.home) {
		return "", fmt.Errorf("%s is your home folder; claude never saves trust there, pick a repo", abs)
	}
	return abs, nil
}

func sameFolder(a, b string) bool {
	realA, errA := filepath.EvalSymlinks(a)
	realB, errB := filepath.EvalSymlinks(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return realA == realB
}
