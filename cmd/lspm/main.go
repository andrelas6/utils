// lspm prints the contents of a folder with their permissions and owner.
//
// Usage:
//
//	lspm [dir]
//
// dir defaults to the current directory.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"text/tabwriter"

	"github.com/andrelas6/utils/internal/fsinfo"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("lspm: ")

	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: lspm [dir]")
		flag.PrintDefaults()
	}
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	if err := run(os.Stdout, dir); err != nil {
		log.Fatal(err)
	}
}

func run(out io.Writer, dir string) error {
	entries, err := fsinfo.List(dir)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PERMISSIONS\tOCTAL\tOWNER\tGROUP\tNAME")
	for _, e := range entries {
		name := e.Name
		if e.IsDir {
			name += "/"
		}
		fmt.Fprintf(w, "%s\t%04o\t%s\t%s\t%s\n", e.Mode, e.Mode.Perm(), e.Owner, e.Group, name)
	}
	return w.Flush()
}
