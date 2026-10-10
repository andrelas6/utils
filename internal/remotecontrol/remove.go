package remotecontrol

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (m *Manager) Remove(out io.Writer, name string, dryRun bool) error {
	unknown := fmt.Errorf("no repo set up as %q, see claude-rc status", name)
	if cleaned, err := unitName("", name); err != nil || cleaned != name {
		return unknown
	}
	unit := unitFileName(name)
	unitPath := filepath.Join(m.unitDir, unit)
	if _, err := os.Stat(unitPath); err != nil {
		return unknown
	}

	mc := m.machineFor(out, dryRun)

	if _, err := mc.Run("systemctl", "--user", "disable", "--now", unit); err != nil {
		return fmt.Errorf("systemctl --user disable --now %s: %w", unit, err)
	}
	if err := mc.Remove(unitPath); err != nil {
		return fmt.Errorf("delete %q: %w", unitPath, err)
	}
	if _, err := mc.Run("systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl --user daemon-reload: %w", err)
	}

	if dryRun {
		return nil
	}
	fmt.Fprintf(out, "removed %s\n", name)
	return nil
}
