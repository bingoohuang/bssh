package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/bingoohuang/bssh/check"
	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/bssh/list"
	"github.com/bingoohuang/bssh/misc"
	"github.com/bingoohuang/bssh/scp"
	"github.com/bingoohuang/ngg/ss"
	"github.com/bingoohuang/ngg/ver"
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
	app.Name = "bssh scp"
	app.Usage = "TUI list select and parallel scp client command."
	app.Copyright = misc.Copyright
	app.Version = ver.Version()

	envHosts := cli.StringSlice(strings.Split(os.Getenv("BSSH_HOST"), ","))
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{Name: "host,H", Usage: "connect server names", Value: &envHosts},
		cli.StringFlag{
			Name: "cnf,c", Value: ss.ExpandHome("~/.bssh.toml"),
			Usage: "config file path",
		},
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
	args, argOptions := conf.ParseMoreOptions(c.Args())
	nargs := len(args)
	if nargs < 2 {
		_, _ = fmt.Fprintln(os.Stderr, "Too few arguments.")
		_ = cli.ShowAppHelp(c)

		os.Exit(1)
	}

	// Set args path
	fromArgs, toArg := args[:nargs-1], args[nargs-1]
	isFromInRemote, isFromInLocal := parseFromLocation(fromArgs)

	isToRemote, _ := check.ParseScpPath(toArg)
	confpath := c.String("cnf")
	data := conf.ReadConf(confpath)
	names := data.GetNameSortedList()
	hosts, searchNames := data.ExpandHosts(c, &argOptions)
	if searchNames != nil {
		names = searchNames
	}

	// Check from and to Type
	check.TypeError(isFromInRemote, isFromInLocal, isToRemote, len(hosts))

	toServer, fromServer := parseFromToServer(hosts, names, isFromInRemote, isToRemote, data)

	// scpService struct
	scpService := new(scp.Scp)

	setFrom(fromArgs, scpService)

	scpService.From.Server = fromServer

	// set to info
	isToRemote, toPath := check.ParseScpPath(toArg)
	scpService.To.IsRemote = isToRemote

	if isToRemote {
		toPath = check.EscapePath(toPath)
	}

	scpService.To.Path = []string{toPath}
	scpService.To.Server = toServer

	scpService.Config = data

	printFromTo(isFromInRemote, scpService, isToRemote)

	scpService.Start(confpath)

	return nil
}

func printFromTo(isFromInRemote bool, scp *scp.Scp, isToRemote bool) {
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
}

func parseFromToServer(hosts, names []string, isFromInRemote, isToRemote bool, data conf.Config) ([]string, []string) {
	var selected, toServer, fromServer []string

	// view server list
	switch {
	// connectHost is set
	case len(hosts) != 0:
		if isFromInRemote {
			fromServer = hosts
		}

		if isToRemote {
			toServer = hosts
		}
	// remote to remote scp
	case isFromInRemote && isToRemote:
		fromServer = list.ShowServersView(&data, "bssh scp(from)>>", names, false)
		toServer = list.ShowServersView(&data, "bssh scp(to)>>", names, true)
	default:
		selected = list.ShowServersView(&data, "bssh scp>>", names, true)

		if isFromInRemote {
			fromServer = selected
		} else {
			toServer = selected
		}
	}

	return toServer, fromServer
}

func parseFromLocation(fromArgs cli.Args) (bool, bool) {
	isFromInRemote, isFromInLocal := false, false

	for _, from := range fromArgs {
		// parse args
		if isFromRemote, _ := check.ParseScpPath(from); isFromRemote {
			isFromInRemote = true
		} else {
			isFromInLocal = true
		}
	}

	return isFromInRemote, isFromInLocal
}

func setFrom(fromArgs cli.Args, scp *scp.Scp) {
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
}
