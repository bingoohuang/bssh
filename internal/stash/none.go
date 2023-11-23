//go:build no_stash

package stash

import (
	"github.com/bingoohuang/bssh/sshlib"
	"github.com/pkg/sftp"
)

func InitFileStash(
	port int,
	connect *sshlib.Connect,
	execCmd func(connect *sshlib.Connect, cmd string) ([]byte, error),
	SftpUpload func(client *sftp.Client, remote string, data []byte) error,
) (int, error) {
	return port, nil
}
