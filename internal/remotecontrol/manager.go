package remotecontrol

import (
	"io"
	"os/exec"
)

type Manager struct {
	unitDir  string
	home     string
	machine  machine
	lookPath func(name string) (string, error)
}

func New(unitDir, home string) *Manager {
	return &Manager{
		unitDir:  unitDir,
		home:     home,
		machine:  realMachine{},
		lookPath: exec.LookPath,
	}
}

func (m *Manager) machineFor(out io.Writer, dryRun bool) machine {
	if dryRun {
		return dryRunMachine{out: out}
	}
	return m.machine
}
