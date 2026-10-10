package remotecontrol

import "os/exec"

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
