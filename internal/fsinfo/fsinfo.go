// Package fsinfo reads file metadata (permissions and ownership) from a directory.
package fsinfo

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Entry describes one item inside a directory.
type Entry struct {
	Name  string
	Mode  fs.FileMode
	Owner string
	Group string
	IsDir bool
}

// List returns the entries of dir with their permissions and owner.
func List(dir string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		// Lstat so symlinks are reported as links, not as their targets.
		info, err := os.Lstat(filepath.Join(dir, de.Name()))
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", de.Name(), err)
		}

		owner, group := ownerOf(info)
		entries = append(entries, Entry{
			Name:  de.Name(),
			Mode:  info.Mode(),
			Owner: owner,
			Group: group,
			IsDir: info.IsDir(),
		})
	}
	return entries, nil
}
