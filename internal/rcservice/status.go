package rcservice

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

const (
	unitPrefix = "claude-rc-"
	unitSuffix = ".service"
)

type Service struct {
	UnitDir  string
	Home     string
	Runner   Runner
	LookPath func(name string) (string, error)
}

func (s *Service) Status(out io.Writer) error {
	units, err := filepath.Glob(filepath.Join(s.UnitDir, unitFileName("*")))
	if err != nil {
		return fmt.Errorf("list units in %q: %w", s.UnitDir, err)
	}

	if len(units) == 0 {
		fmt.Fprintln(out, "no repos set up")
	} else {
		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATE\tREPO")
		for _, path := range units {
			unit := filepath.Base(path)
			state, err := s.unitState(unit)
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

	fmt.Fprintf(out, "\nlogged in: %s\n", s.loggedIn())
	return nil
}

func (s *Service) unitState(unit string) (string, error) {
	out, err := s.Runner.Run("systemctl", "--user", "is-active", unit)
	state := strings.TrimSpace(string(out))
	if state == "" && err != nil {
		return "", fmt.Errorf("systemctl --user is-active %s: %w", unit, err)
	}
	return state, nil
}

func (s *Service) loggedIn() string {
	out, _ := s.Runner.Run("claude", "auth", "status", "--json")
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
			return repo
		}
	}
	return "?"
}
