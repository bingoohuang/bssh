//go:build (!no_base || stash) && !no_stash

package stash

import (
	"fmt"
	"github.com/bingoohuang/bssh/sshlib"
	"github.com/bingoohuang/filestash"
	filestashcommon "github.com/bingoohuang/filestash/server/common"
	"github.com/bingoohuang/filestash/server/middleware"
	"github.com/bingoohuang/filestash/server/model/backend"
	"github.com/bingoohuang/linuxdash"
	"github.com/gorilla/mux"
	"github.com/pkg/sftp"
	"net/http"
	"os"
)

func InitFileStash(
	port int,
	connect *sshlib.Connect,
	execCmd func(connect *sshlib.Connect, cmd string) ([]byte, error),
	SftpUpload func(client *sftp.Client, remote string, data []byte) error,
) (int, error) {
	ftp, err := sftp.NewClient(connect.Client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sftp.NewClient create client error: %v\n", err)
		return 0, err
	}

	if err := SftpUpload(ftp, ".linuxdash_ping_hosts", linuxdash.PingHosts); err != nil {
		return 0, err
	}
	if err := SftpUpload(ftp, ".linuxdash_json_api.sh", []byte(linuxdash.LinuxJsonApiSh)); err != nil {
		return 0, err
	}

	backend.SetSftpClient(&backend.Sftp{SSHClient: nil, SFTPClient: ftp})
	middleware.SftpConfig{}.SetSession()

	if port == 0 {
		port = 8333
	}

	app := filestashcommon.App{}
	rr := mux.NewRouter()
	config := filestash.AppConfig{Port: port, R: rr}
	dashStatic := http.FileServer(http.FS(linuxdash.DashStatic))
	filestash.GET(rr, "/dash", http.StripPrefix("/dash", dashStatic).ServeHTTP)
	filestash.GET(rr, "/fonts-googleapis-com.css", dashStatic.ServeHTTP)
	filestash.GET(rr, "/linuxDash.min.css", dashStatic.ServeHTTP)
	filestash.GET(rr, "/linuxDash.min.js", dashStatic.ServeHTTP)
	filestash.GET(rr, "/server/", linuxdash.MakeDashServe(func(module string) ([]byte, error) {
		return execCmd(connect, "bash .linuxdash_json_api.sh "+module)
	}))
	result := config.Init(&app)
	config.Start()
	return result.Port, nil
}
