package app

import (
	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/gg/pkg/v"
	"github.com/bingoohuang/gou/str"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/bingoohuang/bssh/misc"

	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/bssh/sftp"
	"github.com/urfave/cli"
)

// nolint
const appHelpTemplate = `NAME:
    {{.Name}} - {{.Usage}}
USAGE:
    {{.HelpName}} {{if .VisibleFlags}}[options]{{end}}
    {{if len .Authors}}
AUTHOR:
    {{range .Authors}}{{ . }}{{end}}
    {{end}}{{if .Commands}}
COMMANDS:
    {{range .Commands}}{{if not .HideHelp}}{{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
OPTIONS:
    {{range .VisibleFlags}}{{.}}
    {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
    {{.Copyright}}
    {{end}}{{if .Version}}
VERSION:
    {{.Version}}
    {{end}}
USAGE:
	# start lsftp shell
	{{.Name}}
`

// Lsftp sftp ...
func Lsftp() (app *cli.App) {
	cli.AppHelpTemplate = appHelpTemplate
	app = cli.NewApp()
	app.Name = "bssh ftp"
	app.Usage = "TUI list select and parallel sftp client command."
	app.Copyright = misc.Copyright
	app.Version = v.Version()

	app.Flags = []cli.Flag{
		cli.StringSliceFlag{Name: "host,H", Usage: "connect `servername`."},
		cli.StringFlag{Name: "cnf,c", Value: str.PickFirst(homedir.Expand("~/.bssh.toml")),
			Usage: "config file path"},
		cli.BoolFlag{Name: "help,h", Usage: "print this help"},
	}

	app.EnableBashCompletion = true
	app.HideHelp = true

	app.Action = lsftpAction

	return app
}

func lsftpAction(c *cli.Context) error {
	common.CheckHelpFlag(c)

	confpath := c.String("cnf")

	data := conf.ReadConf(confpath)
	names := data.GetNameSortedList()
	hosts, searchNames := data.ExpandHosts(c, nil)
	if searchNames != nil {
		names = searchNames
	}

	// scp struct
	r := new(sftp.RunSftp)
	r.Config = data
	r.SelectServer = parseSelected("bssh ftp>>", hosts, names, data, true)

	r.Start(confpath)

	return nil
}
