//go:build !windows

package sftp

import (
	"os"
	"syscall"
)

func SyscallStat(f os.FileInfo) (uid, gid uint32, size int64, ok bool) {
	sys := f.Sys()
	if stat, ok := sys.(*syscall.Stat_t); ok {
		return stat.Uid, stat.Gid, stat.Size, ok
	}

	return 0, 0, 0, false
}
