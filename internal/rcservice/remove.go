package rcservice

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (s *Service) Remove(out io.Writer, name string, dryRun bool) error {
	if dryRun {
		return errors.New("dry run isn't ready yet")
	}

	unknown := fmt.Errorf("no repo set up as %q, see claude-rc status", name)
	if cleaned, err := unitName("", name); err != nil || cleaned != name {
		return unknown
	}
	unit := unitFileName(name)
	unitPath := filepath.Join(s.UnitDir, unit)
	if _, err := os.Stat(unitPath); err != nil {
		return unknown
	}

	if _, err := s.Runner.Run("systemctl", "--user", "disable", "--now", unit); err != nil {
		return fmt.Errorf("systemctl --user disable --now %s: %w", unit, err)
	}
	if err := os.Remove(unitPath); err != nil {
		return fmt.Errorf("delete %q: %w", unitPath, err)
	}
	if _, err := s.Runner.Run("systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl --user daemon-reload: %w", err)
	}

	fmt.Fprintf(out, "removed %s\n", name)
	return nil
}
