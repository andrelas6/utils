package fsinfo

import (
	"os"
	"path/filepath"
	"testing"
)

// A folder with one file and one subfolder lists both, sorted by name, with
// their permissions, the dir flag and a resolved owner.
func TestList(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}

	entries, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	// os.ReadDir returns entries sorted by name.
	file, sub := entries[0], entries[1]
	if file.Name != "a.txt" || file.IsDir || file.Mode.Perm() != 0o640 {
		t.Errorf("unexpected file entry: %+v", file)
	}
	if sub.Name != "sub" || !sub.IsDir || sub.Mode.Perm() != 0o750 {
		t.Errorf("unexpected dir entry: %+v", sub)
	}
	if file.Owner == "" || file.Owner == "?" {
		t.Errorf("owner not resolved: %q", file.Owner)
	}
}

// Listing a folder that doesn't exist returns an error.
func TestListMissingDir(t *testing.T) {
	if _, err := List(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing directory")
	}
}
