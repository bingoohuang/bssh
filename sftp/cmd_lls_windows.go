package sftp

import (
	"os"
)

func SyscallStat(f os.FileInfo) (uid, gid uint32, size int64, ok bool) {
	return 0, 0, 0, false
}
