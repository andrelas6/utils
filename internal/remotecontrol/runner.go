package remotecontrol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).Output()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		if stderr := bytes.TrimSpace(exitErr.Stderr); len(stderr) > 0 {
			return out, fmt.Errorf("%w: %s", err, stderr)
		}
	}
	return out, err
}

type DryRunner struct {
	Out io.Writer
}

func (d DryRunner) Run(name string, args ...string) ([]byte, error) {
	fmt.Fprintf(d.Out, "would run: %s\n", strings.Join(append([]string{name}, args...), " "))
	return nil, nil
}
