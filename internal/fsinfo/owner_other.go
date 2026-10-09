//go:build !unix

package fsinfo

import "io/fs"

// ownerOf is not supported outside unix systems.
func ownerOf(fs.FileInfo) (owner, group string) {
	return "?", "?"
}
