package remotecontrol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type machine interface {
	Run(name string, args ...string) ([]byte, error)
	WriteFile(path string, content []byte) error
	Remove(path string) error
}

type realMachine struct{}

func (realMachine) Run(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).Output()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		if stderr := bytes.TrimSpace(exitErr.Stderr); len(stderr) > 0 {
			return out, fmt.Errorf("%w: %s", err, stderr)
		}
	}
	return out, err
}

func (realMachine) WriteFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func (realMachine) Remove(path string) error {
	return os.Remove(path)
}

type dryRunMachine struct {
	out io.Writer
}

func (d dryRunMachine) Run(name string, args ...string) ([]byte, error) {
	fmt.Fprintf(d.out, "would run: %s\n", strings.Join(append([]string{name}, args...), " "))
	return nil, nil
}

func (d dryRunMachine) WriteFile(path string, content []byte) error {
	fmt.Fprintf(d.out, "would write: %s\n\n%s\n", path, content)
	return nil
}

func (d dryRunMachine) Remove(path string) error {
	fmt.Fprintf(d.out, "would delete: %s\n", path)
	return nil
}
