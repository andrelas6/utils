package main

import (
	"fmt"
	"io"
)

type mockServices struct{}

func (mockServices) Add(out io.Writer, repo, name string, dryRun bool) error {
	fmt.Fprintf(out, "mock: would add %s (name %q, dry run %t)\n", repo, name, dryRun)
	return nil
}

func (mockServices) Remove(out io.Writer, name string, dryRun bool) error {
	fmt.Fprintf(out, "mock: would remove %s (dry run %t)\n", name, dryRun)
	return nil
}
