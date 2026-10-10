package rcservice

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) Add(out io.Writer, repo, name string, dryRun bool) error {
	repo, err := s.checkRepo(repo)
	if err != nil {
		return err
	}
	name, err = unitName(repo, name)
	if err != nil {
		return err
	}

	unit := unitFileName(name)
	unitPath := filepath.Join(s.UnitDir, unit)
	if _, err := os.Stat(unitPath); err == nil {
		return fmt.Errorf("%s is already set up (%s), remove it first", name, unitPath)
	}

	claude, err := s.LookPath("claude")
	if err != nil {
		return fmt.Errorf("can't find claude: %w", err)
	}

	content := unitFile(repo, name, claude)
	runner := s.Runner
	if dryRun {
		fmt.Fprintf(out, "would write: %s\n\n%s\n", unitPath, content)
		runner = DryRunner{Out: out}
	} else {
		if err := os.MkdirAll(s.UnitDir, 0o755); err != nil {
			return fmt.Errorf("create %q: %w", s.UnitDir, err)
		}
		if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %q: %w", unitPath, err)
		}
	}

	fmt.Fprintf(out, "first time in this repo? run `claude remote-control` once in %s to answer the trust and enable prompts\n", repo)
	for _, args := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "--now", unit},
	} {
		if _, err := runner.Run("systemctl", args...); err != nil {
			return fmt.Errorf("systemctl %s: %w (to undo, run: claude-rc remove %s)", strings.Join(args, " "), err, name)
		}
	}

	if dryRun {
		return nil
	}
	fmt.Fprintf(out, "added %s: %s\n", name, unitPath)
	fmt.Fprintln(out, "check it with: claude-rc status")
	return nil
}

func (s *Service) checkRepo(repo string) (string, error) {
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
	if abs == filepath.Clean(s.Home) {
		return "", fmt.Errorf("%s is your home folder; claude never saves trust there, pick a repo", abs)
	}
	return abs, nil
}
