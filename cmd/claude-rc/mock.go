package main

import (
	"fmt"
	"io"
)

type mockServices struct{}

func (mockServices) Remove(out io.Writer, name string, dryRun bool) error {
	fmt.Fprintf(out, "mock: would remove %s (dry run %t)\n", name, dryRun)
	return nil
}
