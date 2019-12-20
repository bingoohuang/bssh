package app

import (
	"os"

	"github.com/bingoohuang/gou/str"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/blacknon/lssh/misc"

	"github.com/blacknon/lssh"
	"github.com/blacknon/lssh/conf"
	"github.com/blacknon/lssh/sftp"
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
	app.Name = "lssh ftp"
	app.Usage = "TUI list select and parallel sftp client command."
	app.Copyright = misc.Copyright
	app.Version = lssh.AppVersion

	app.Flags = []cli.Flag{
		cli.StringSliceFlag{Name: "host,H", Usage: "connect `servername`."},
		cli.StringFlag{Name: "file,F", Value: str.PickFirst(homedir.Expand("~/.lssh.conf")),
			Usage: "config file path"},
		cli.BoolFlag{Name: "help,h", Usage: "print this help"},
	}

	app.EnableBashCompletion = true
	app.HideHelp = true

	app.Action = lsftpAction

	return app
}

func lsftpAction(c *cli.Context) error {
	if c.Bool("help") {
		_ = cli.ShowAppHelp(c)

		os.Exit(0)
	}

	hosts := c.StringSlice("host")
	confpath := c.String("file")

	data := conf.ReadConf(confpath)
	names := conf.GetNameSortedList(data)

	// scp struct
	r := new(sftp.RunSftp)
	r.Config = data
	r.SelectServer = parseSelected("lssh ftp>>", hosts, names, data, true)

	r.Start()

	return nil
}
