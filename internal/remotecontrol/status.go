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

type unitInfo struct {
	state, restarts, lastExit string
}

func (m *Manager) Status(out io.Writer) error {
	units, err := filepath.Glob(filepath.Join(m.unitDir, unitFileName("*")))
	if err != nil {
		return fmt.Errorf("list units in %q: %w", m.unitDir, err)
	}

	if len(units) == 0 {
		fmt.Fprintln(out, "no repos set up")
	} else {
		var notes []string
		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATE\tRESTARTS\tLAST EXIT\tREPO")
		for _, path := range units {
			unit := filepath.Base(path)
			name := strings.TrimSuffix(strings.TrimPrefix(unit, unitPrefix), unitSuffix)
			st, err := m.unitState(unit)
			switch {
			case err != nil:
				st = unitInfo{state: "unknown", restarts: "?", lastExit: "?"}
				notes = append(notes, fmt.Sprintf("can't read %s's state: %v", name, err))
			case st.state != "active":
				notes = append(notes, fmt.Sprintf("see why %s isn't running: journalctl --user -u %s%s", name, unitPrefix, name))
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, st.state, st.restarts, st.lastExit, repoOf(path))
		}
		if err := w.Flush(); err != nil {
			return err
		}
		if len(notes) > 0 {
			fmt.Fprintf(out, "\n%s\n", strings.Join(notes, "\n"))
		}
	}

	fmt.Fprintf(out, "\nlogged in: %s\n", m.loggedIn())
	return nil
}

func (m *Manager) unitState(unit string) (unitInfo, error) {
	out, err := m.machine.Run("systemctl", "--user", "show", unit, "--property=ActiveState,SubState,NRestarts,ExecMainStatus")
	if err != nil {
		return unitInfo{}, fmt.Errorf("systemctl --user show %s: %w", unit, err)
	}

	props := map[string]string{}
	for line := range strings.Lines(string(out)) {
		if key, value, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			props[key] = value
		}
	}
	orUnknown := func(key string) string {
		if v := props[key]; v != "" {
			return v
		}
		return "?"
	}

	st := unitInfo{state: orUnknown("ActiveState"), restarts: orUnknown("NRestarts"), lastExit: orUnknown("ExecMainStatus")}
	if props["SubState"] == "auto-restart" {
		st.state = "restarting"
	}
	return st, nil
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
