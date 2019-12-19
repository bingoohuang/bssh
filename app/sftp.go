package app

import (
	"os"
	"sort"

	"github.com/bingoohuang/gou/str"
	"github.com/mitchellh/go-homedir"

	"github.com/blacknon/lssh/list"
	"github.com/blacknon/lssh/misc"

	"github.com/blacknon/lssh"
	"github.com/blacknon/lssh/conf"
	"github.com/blacknon/lssh/sftp"
	"github.com/urfave/cli"
)

func Lsftp() (app *cli.App) {
	// nolint
	cli.AppHelpTemplate = `NAME:
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
	// Create app
	app = cli.NewApp()
	// app.UseShortOptionHandling = true
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

	app.Action = func(c *cli.Context) error {
		// show help messages
		if c.Bool("help") {
			_ = cli.ShowAppHelp(c)

			os.Exit(0)
		}

		// hosts := c.StringSlice("host")
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

		// start lsftp shell
		runSftp.Start()

		return nil
	}

	return app
}
