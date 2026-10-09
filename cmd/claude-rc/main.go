package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/andrelas6/utils/internal/rcservice"
)

const usage = `usage:
  claude-rc add [-name n] [-dry-run] <repo>
  claude-rc remove [-dry-run] <name>
  claude-rc status
`

var errUsage = errors.New("wrong arguments")

type services interface {
	Add(out io.Writer, repo, name string, dryRun bool) error
	Remove(out io.Writer, name string, dryRun bool) error
	Status(out io.Writer) error
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("claude-rc: ")

	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	svc := struct {
		mockServices
		*rcservice.Service
	}{
		Service: &rcservice.Service{
			UnitDir: filepath.Join(configDir, "systemd", "user"),
			Runner:  rcservice.ExecRunner{},
		},
	}

	if err := run(os.Args[1:], os.Stdout, os.Stderr, svc); err != nil {
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		log.Fatal(err)
	}
}

func run(args []string, out, errOut io.Writer, svc services) error {
	if len(args) == 0 {
		return showUsage(errOut, errors.New("missing command"))
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "add":
		fs := newFlagSet(cmd, errOut)
		name := fs.String("name", "", "name for the service (default: the repo folder's name)")
		dryRun := fs.Bool("dry-run", false, "print what would happen without changing anything")
		repo, err := parseOneArg(fs, args, "<repo>")
		if err != nil {
			return showUsage(errOut, err)
		}
		return svc.Add(out, repo, *name, *dryRun)

	case "remove":
		fs := newFlagSet(cmd, errOut)
		dryRun := fs.Bool("dry-run", false, "print what would happen without changing anything")
		name, err := parseOneArg(fs, args, "<name>")
		if err != nil {
			return showUsage(errOut, err)
		}
		return svc.Remove(out, name, *dryRun)

	case "status":
		if len(args) != 0 {
			return showUsage(errOut, errors.New("status takes no arguments"))
		}
		return svc.Status(out)

	default:
		return showUsage(errOut, fmt.Errorf("unknown command %q", cmd))
	}
}

func newFlagSet(name string, errOut io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.Usage = func() {}
	return fs
}

func parseOneArg(fs *flag.FlagSet, args []string, argName string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() != 1 {
		return "", fmt.Errorf("%s needs exactly one %s", fs.Name(), argName)
	}
	return fs.Arg(0), nil
}

func showUsage(errOut io.Writer, err error) error {
	fmt.Fprintf(errOut, "claude-rc: %v\n\n%s", err, usage)
	return fmt.Errorf("%w: %v", errUsage, err)
}
