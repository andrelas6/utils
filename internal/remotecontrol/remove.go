package remotecontrol

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (m *Manager) Remove(out io.Writer, name string, dryRun bool) error {
	if cleaned, err := unitName("", name); err != nil || cleaned != name {
		return fmt.Errorf("%q isn't a valid name, see claude-rc status for the names", name)
	}
	unit := unitFileName(name)
	unitPath := filepath.Join(m.unitDir, unit)
	if _, err := os.Stat(unitPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no repo set up as %q, see claude-rc status", name)
	} else if err != nil {
		return fmt.Errorf("can't check %s: %w", unitPath, err)
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
