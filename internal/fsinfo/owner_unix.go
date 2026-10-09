//go:build unix

package fsinfo

import (
	"io/fs"
	"os/user"
	"strconv"
	"syscall"
)

// ownerOf resolves the user and group names that own the file.
// It falls back to the numeric id when the name cannot be looked up.
func ownerOf(info fs.FileInfo) (owner, group string) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "?", "?"
	}

	uid := strconv.FormatUint(uint64(stat.Uid), 10)
	gid := strconv.FormatUint(uint64(stat.Gid), 10)

	owner, group = uid, gid
	if u, err := user.LookupId(uid); err == nil {
		owner = u.Username
	}
	if g, err := user.LookupGroupId(gid); err == nil {
		group = g.Name
	}
	return owner, group
}
