package remotecontrol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

func (m *Manager) Status(out io.Writer) error {
	units, err := filepath.Glob(filepath.Join(m.unitDir, unitFileName("*")))
	if err != nil {
		return fmt.Errorf("list units in %q: %w", m.unitDir, err)
	}

	if len(units) == 0 {
		fmt.Fprintln(out, "no repos set up")
	} else {
		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATE\tREPO")
		for _, path := range units {
			unit := filepath.Base(path)
			state, err := m.unitState(unit)
			if err != nil {
				return err
			}
			name := strings.TrimSuffix(strings.TrimPrefix(unit, unitPrefix), unitSuffix)
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, state, repoOf(path))
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}

	fmt.Fprintf(out, "\nlogged in: %s\n", m.loggedIn())
	return nil
}

func (m *Manager) unitState(unit string) (string, error) {
	out, err := m.machine.Run("systemctl", "--user", "is-active", unit)
	state := strings.TrimSpace(string(out))
	if state == "" && err != nil {
		return "", fmt.Errorf("systemctl --user is-active %s: %w", unit, err)
	}
	return state, nil
}

func (m *Manager) loggedIn() string {
	out, _ := m.machine.Run("claude", "auth", "status", "--json")
	var auth struct {
		LoggedIn *bool `json:"loggedIn"`
	}
	if json.Unmarshal(out, &auth) != nil || auth.LoggedIn == nil {
		return "unknown"
	}
	if *auth.LoggedIn {
		return "yes"
	}
	return "no"
}

func repoOf(unitPath string) string {
	f, err := os.Open(unitPath)
	if err != nil {
		return "?"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if repo, ok := strings.CutPrefix(scanner.Text(), "WorkingDirectory="); ok {
			return strings.ReplaceAll(repo, "%%", "%")
		}
	}
	return "?"
}
