package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/bingoohuang/gou/str"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/blacknon/lssh/list"
	"github.com/blacknon/lssh/misc"

	"github.com/blacknon/lssh"
	"github.com/blacknon/lssh/check"
	"github.com/blacknon/lssh/common"
	"github.com/blacknon/lssh/conf"
	"github.com/blacknon/lssh/scp"
	"github.com/urfave/cli"
)

// nolint
const lscpAppHelpTemplate = `NAME:
    {{.Name}} - {{.Usage}}
USAGE:
    {{.HelpName}} {{if .VisibleFlags}}[options]{{end}} (local|remote):from_path... (local|remote):to_path
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
    # local to remote scp
    {{.Name}} /path/to/local... remote:/path/to/remote

    # remote to local scp
    {{.Name}} remote:/path/to/remote... /path/to/local

    # remote to remote scp
    {{.Name}} remote:/path/to/remote... remote:/path/to/local
`

// Lscp scp ...
func Lscp() (app *cli.App) {
	cli.AppHelpTemplate = lscpAppHelpTemplate
	app = cli.NewApp()
	app.Name = "lssh scp"
	app.Usage = "TUI list select and parallel scp client command."
	app.Copyright = misc.Copyright
	app.Version = lssh.AppVersion

	app.Flags = []cli.Flag{
		cli.StringSliceFlag{Name: "host,H", Usage: "connect server names"},
		cli.StringFlag{Name: "file,F", Value: str.PickFirst(homedir.Expand("~/.lssh.conf")),
			Usage: "config file path"},
		cli.BoolFlag{Name: "help,h", Usage: "print this help"},
	}
	app.EnableBashCompletion = true
	app.HideHelp = true
	app.Action = lscpAction

	return app
}

func lscpAction(c *cli.Context) error {
	common.CheckHelpFlag(c)

	// check count args
	args := c.Args()
	if len(args) < 2 {
		_, _ = fmt.Fprintln(os.Stderr, "Too few arguments.")
		_ = cli.ShowAppHelp(c)

		os.Exit(1)
	}

	// Set args path
	fromArgs, toArg := args[:c.NArg()-1], args[c.NArg()-1]

	isFromInRemote, isFromInLocal := false, false

	for _, from := range fromArgs {
		// parse args
		isFromRemote, _ := check.ParseScpPath(from)

		if isFromRemote {
			isFromInRemote = true
		} else {
			isFromInLocal = true
		}
	}

	isToRemote, _ := check.ParseScpPath(toArg)

	hosts := c.StringSlice("host")
	confpath := c.String("file")

	// Check from and to Type
	check.TypeError(isFromInRemote, isFromInLocal, isToRemote, len(hosts))

	data := conf.ReadConf(confpath)
	names := conf.GetNameSortedList(data)

	var selected, toServer, fromServer []string

	// view server list
	switch {
	// connectHost is set
	case len(hosts) != 0:
		if !check.ExistServer(hosts, names) {
			fmt.Fprintln(os.Stderr, "Input Server not found from list.")
			os.Exit(1)
		}

		toServer = hosts
	// remote to remote scp
	case isFromInRemote && isToRemote:
		fromServer = list.ShowServersView(&data, "lssh scp(from)>>", names, false)
		toServer = list.ShowServersView(&data, "lssh scp(to)>>", names, true)
	default:
		selected = list.ShowServersView(&data, "lssh scp>>", names, true)

		if isFromInRemote {
			fromServer = selected
		} else {
			toServer = selected
		}
	}

	// scp struct
	scp := new(scp.Scp)

	// set from info
	for _, from := range fromArgs {
		isFromRemote, fromPath := check.ParseScpPath(from)

		// Check local file exists
		if !isFromRemote {
			_, err := os.Stat(common.GetFullPath(fromPath))
			if err != nil {
				fmt.Fprintf(os.Stderr, "not found path %s \n", fromPath)
				os.Exit(1)
			}

			fromPath = common.GetFullPath(fromPath)
		}

		// set from data
		scp.From.IsRemote = isFromRemote

		if isFromRemote {
			fromPath = check.EscapePath(fromPath)
		}

		scp.From.Path = append(scp.From.Path, fromPath)
	}

	scp.From.Server = fromServer

	// set to info
	isToRemote, toPath := check.ParseScpPath(toArg)
	scp.To.IsRemote = isToRemote

	if isToRemote {
		toPath = check.EscapePath(toPath)
	}

	scp.To.Path = []string{toPath}
	scp.To.Server = toServer

	scp.Config = data

	// print from
	if !isFromInRemote {
		_, _ = fmt.Fprintf(os.Stderr, "From local:%s\n", scp.From.Path)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "From remote(%s):%s\n", strings.Join(scp.From.Server, ","), scp.From.Path)
	}

	// print to
	if !isToRemote {
		_, _ = fmt.Fprintf(os.Stderr, "To   local:%s\n", scp.To.Path)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "To   remote(%s):%s\n", strings.Join(scp.To.Server, ","), scp.To.Path)
	}

	scp.Start()

	return nil
}
