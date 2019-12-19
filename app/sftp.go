package app

import (
	"os"
	"sort"

	"github.com/bingoohuang/gou/str"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/blacknon/lssh/list"
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
	app.Name = "lsftp"
	app.Usage = "TUI list select and parallel sftp client command."
	app.Copyright = misc.Copyright
	app.Version = lssh.AppVersion

	app.Flags = []cli.Flag{
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

	confpath := c.String("file")

	// Get config data
	data := conf.ReadConf(confpath)

	// Get Server Name List (and sort List)
	names := conf.GetNameList(data)
	sort.Strings(names)

	selectedGroup := list.ShowGroupsView(&data)
	selected := list.ShowServersView(&data, "lsftp>>", selectedGroup, names, true)

	// scp struct
	runSftp := new(sftp.RunSftp)
	runSftp.Config = data
	runSftp.SelectServer = selected

	runSftp.Start()

	return nil
}
