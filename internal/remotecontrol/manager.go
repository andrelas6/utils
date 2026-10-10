package remotecontrol

import "os/exec"

type Manager struct {
	unitDir  string
	home     string
	runner   Runner
	lookPath func(name string) (string, error)
}

func New(unitDir, home string) *Manager {
	return &Manager{
		unitDir:  unitDir,
		home:     home,
		runner:   ExecRunner{},
		lookPath: exec.LookPath,
	}
}
