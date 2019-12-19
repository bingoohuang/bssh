package app

import (
	"fmt"
	"os"
	"sort"

	"github.com/bingoohuang/gou/str"

	"github.com/mitchellh/go-homedir"

	"github.com/blacknon/lssh/list"
	"github.com/blacknon/lssh/misc"

	"github.com/blacknon/lssh"
	"github.com/blacknon/lssh/check"
	"github.com/blacknon/lssh/common"
	"github.com/blacknon/lssh/conf"
	sshcmd "github.com/blacknon/lssh/ssh"
	"github.com/urfave/cli"
)

// nolint
const sshAppHelpTemplate = `NAME:
    {{.Name}} - {{.Usage}}
USAGE:
    {{.HelpName}} {{if .VisibleFlags}}[options]{{end}} [commands...]
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
    # connect ssh
    {{.Name}}

    # parallel run command in select server over ssh
    {{.Name}} -p command...

    # parallel run command in select server over ssh, do it interactively.
    {{.Name}} -s
`

// Lssh ssh ...
func Lssh() (app *cli.App) {
	cli.AppHelpTemplate = sshAppHelpTemplate

	app = cli.NewApp()
	// app.UseShortOptionHandling = true
	app.Name = "lssh"
	app.Usage = "TUI list select and parallel ssh client command."
	app.Copyright = misc.Copyright
	app.Version = lssh.AppVersion

	// TODO(blacknon): オプションの追加
	//     -f       ... バックグラウンドでの接続(X11接続やport forwardingをバックグラウンドで実行する場合など)。
	//                  「ssh -f」と同じ。 (v0.6.1)
	//                  (https://github.com/sevlyar/go-daemon)
	//     -a       ... 自動接続モード(接続が切れてしまった場合、自動的に再接続を試みる)。autossh的なoptionとして追加。  (v0.6.1)
	//     -A <num> ... 自動接続モード(接続が切れてしまった場合、自動的に再接続を試みる)。再試行の回数指定(デフォルトは3回?)。  (v0.6.1)
	//     -w       ... コマンド実行時にサーバ名ヘッダの表示をする (v0.6.0)
	//     -W       ... コマンド実行時にサーバ名ヘッダの表示をしない (v0.6.0)
	//     --read_profile
	//              ... デフォルトではlocalrc読み込みでのshellではsshサーバ上のprofileは読み込まないが、このオプションを指定することで読み込まれるようになる (v0.6.1)

	// TODO(blacknon): コマンドオプションの指定方法(特にポートフォワーディング)をOpenSSHに合わせる

	// Set options
	app.Flags = []cli.Flag{
		// common option
		cli.StringSliceFlag{Name: "host,H", Usage: "connect `servername`."},
		cli.StringFlag{Name: "file,F", Value: str.PickFirst(homedir.Expand("~/.lssh.conf")),
			Usage: "config `filepath`."},

		// port forward option
		cli.StringFlag{Name: "L", Usage: "Local port forward mode.Specify a `[bind_address:]port:remote_addr:port`."},
		cli.StringFlag{Name: "R", Usage: "Remote port forward mode.Specify a `[bind_address:]port:remote_addr:port`."},
		cli.StringFlag{Name: "D", Usage: "Dynamic port forward mode(Socks5). Specify a `port`."},
		// cli.StringFlag{Name: "portforward-local", Usage: "port forwarding parameter,
		//			`address:port`. use local-forward or reverse-forward. (local port(ex. 127.0.0.1:8080))."},
		// cli.StringFlag{Name: "portforward-remote", Usage: "port forwarding parameter,
		//			`address:port`. use local-forward or reverse-forward. (remote port(ex. 127.0.0.1:80))."},

		// Other bool
		cli.BoolFlag{Name: "w", Usage: "Displays the server header when in command execution mode."},
		cli.BoolFlag{Name: "W", Usage: "Not displays the server header when in command execution mode."},
		cli.BoolFlag{Name: "not-execute,N", Usage: "not execute remote command and shell."},
		cli.BoolFlag{Name: "x11,X", Usage: "x11 forwarding(forward to ${DISPLAY})."},
		cli.BoolFlag{Name: "term,t", Usage: "run specified command at terminal."},
		cli.BoolFlag{Name: "parallel,p", Usage: "run command parallel node(tail -F etc...)."},
		cli.BoolFlag{Name: "localrc", Usage: "use local bashrc shell."},
		cli.BoolFlag{Name: "not-localrc", Usage: "not use local bashrc shell."},
		cli.BoolFlag{Name: "pshell,s", Usage: "use parallel-shell(pshell) (alpha)."},
		cli.BoolFlag{Name: "list,l", Usage: "print server list from config."},
		cli.BoolFlag{Name: "help,h", Usage: "print this help"},
	}
	app.EnableBashCompletion = true
	app.HideHelp = true

	// Run command action
	app.Action = lsshAction

	return app
}

// lsshAction actions ssh functions.
func lsshAction(c *cli.Context) error {
	if c.Bool("help") {
		_ = cli.ShowAppHelp(c)

		os.Exit(0)
	}

	hosts := c.StringSlice("host")
	confpath := c.String("file")

	// Get config data
	data := conf.ReadConf(confpath)

	isMulti := parseMultiFlag(c)

	// Extraction server name list from 'data'
	names := conf.GetNameList(data)
	sort.Strings(names)

	processList(c, names)

	r := new(sshcmd.Run)
	r.ServerList = parseSelected(hosts, names, data, isMulti)
	r.Conf = data

	r.Mode = parseMode(c)
	r.ExecCmd = c.Args() // exec command
	r.IsParallel = c.Bool("parallel")
	r.X11 = c.Bool("x11")          // x11 forwarding
	r.IsTerm = c.Bool("term")      // is tty
	r.IsBashrc = c.Bool("localrc") // local bashrc use
	r.IsNotBashrc = c.Bool("not-localrc")

	// set w/W flag
	if c.Bool("w") {
		fmt.Println("enable w")

		r.EnableHeader = true
	}

	if c.Bool("W") {
		fmt.Println("enable W")

		r.DisableHeader = true
	}

	err := dealPortForward(c, r)

	if err != nil {
		fmt.Printf("Error: %s \n", err)
	}

	// is not execute
	r.IsNone = c.Bool("not-execute")

	// Dynamic port forwarding port
	r.DynamicPortForward = c.String("D")

	r.Start()

	return nil
}

func dealPortForward(c *cli.Context, r *sshcmd.Run) error {
	var err error

	switch {
	case c.String("L") != "":
		r.PortForwardMode = "L"
		r.PortForwardLocal, r.PortForwardRemote, err = common.ParseForwardPort(c.String("L"))

	case c.String("R") != "":
		r.PortForwardMode = "R"
		r.PortForwardLocal, r.PortForwardRemote, err = common.ParseForwardPort(c.String("R"))

	case c.String("L") != "" && c.String("R") != "":
		r.PortForwardMode = "R"
		r.PortForwardLocal, r.PortForwardRemote, err = common.ParseForwardPort(c.String("R"))

	default:
		r.PortForwardMode = ""
	}

	return err
}

func parseMultiFlag(c *cli.Context) bool {
	// Set `exec command` or `shell` flag
	return (len(c.Args()) > 0 || c.Bool("pshell")) && !c.Bool("not-execute")
}

func processList(c *cli.Context, names []string) {
	// Check list flag
	if !c.Bool("list") {
		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "lssh Server List:\n")
	for v := range names {
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", names[v])
	}

	os.Exit(0)
}

func parseMode(c *cli.Context) string {
	switch {
	case c.Bool("pshell") && !c.Bool("not-execute"):
		return "pshell"
	case len(c.Args()) > 0 && !c.Bool("not-execute"):
		// Becomes a shell when not-execute is given.
		return "cmd"
	default:
		return "shell"
	}
}

func parseSelected(hosts []string, names []string, data conf.Config, isMulti bool) []string {
	var selected []string

	if len(hosts) > 0 {
		if !check.ExistServer(hosts, names) {
			_, _ = fmt.Fprintln(os.Stderr, "Input Server not found from list.")
			os.Exit(1)
		} else {
			selected = hosts
		}
	} else {
		selectedGroup := list.ShowGroupsView(&data)
		selected = list.ShowServersView(&data, "lssh>>", selectedGroup, names, isMulti)
	}

	return selected
}