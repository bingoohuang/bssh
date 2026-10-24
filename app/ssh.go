package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/bssh/list"
	"github.com/bingoohuang/bssh/misc"
	sshcmd "github.com/bingoohuang/bssh/ssh"
	"github.com/bingoohuang/ngg/ss"
	"github.com/bingoohuang/ngg/ver"
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

    # check connectivity of all registered servers, print the reachable ones
    {{.Name}} --check
`

// Lssh ssh ...
func Lssh() (app *cli.App) {
	cli.AppHelpTemplate = sshAppHelpTemplate

	app = cli.NewApp()
	// app.UseShortOptionHandling = true
	app.Name = "bssh"
	app.Usage = "TUI list select and parallel ssh client command."
	app.Copyright = misc.Copyright
	app.Version = ver.Version()

	// TDXX(blacknon): オプションの追加
	//     -f       ... バックグラウンドでの接続(X11接続やport forwardingをバックグラウンドで実行する場合など)。
	//                  「ssh -f」と同じ。 (v0.6.1)
	//                  (https://github.com/sevlyar/go-daemon)
	//     -a       ... 自動接続モード(接続が切れてしまった場合、自動的に再接続を試みる)。autossh的なoptionとして追加。  (v0.6.1)
	//     -A <num> ... 自動接続モード(接続が切れてしまった場合、自動的に再接続を試みる)。再試行の回数指定(デフォルトは3回?)。  (v0.6.1)
	//     -w       ... コマンド実行時にサーバ名ヘッダの表示をする (v0.6.0)
	//     -W       ... コマンド実行時にサーバ名ヘッダの表示をしない (v0.6.0)
	//     --read_profile
	//              ... デフォルトではlocalrc読み込みでのshellではsshサーバ上のprofileは読み込まないが、このオプションを指定することで読み込まれるようになる (v0.6.1)

	// TDXX(blacknon): コマンドオプションの指定方法(特にポートフォワーディング)をOpenSSHに合わせる

	// Set options
	envHosts := cli.StringSlice(strings.Split(os.Getenv("HOST"), ","))
	app.Flags = []cli.Flag{
		// common option
		cli.StringSliceFlag{Name: "host,H", Usage: "connect `servername`.", Value: &envHosts},
		cli.StringFlag{
			Name: "cnf,c", Value: ss.ExpandHome("~/.bssh/.bssh.toml"),
			Usage: "config `filepath`.",
		},

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
		cli.BoolFlag{Name: "parallel,p", Usage: "run command parallel node(tail -c etc...)."},
		cli.BoolFlag{Name: "localrc", Usage: "use local bashrc shell."},
		cli.BoolFlag{Name: "not-localrc", Usage: "not use local bashrc shell."},
		cli.BoolFlag{Name: "pshell,s", Usage: "use parallel-shell(pshell) (alpha)."},
		cli.BoolFlag{Name: "list,l", Usage: "print server list from config."},
		cli.BoolFlag{Name: "help,h", Usage: "print this help"},

		// connectivity check option
		cli.BoolFlag{Name: "check,k", Usage: "check connectivity of the servers, and print the reachable ones."},
		cli.IntFlag{Name: "check-timeout", Value: 3, Usage: "connect `timeout` seconds per server in check mode."},
		cli.IntFlag{Name: "check-parallel", Value: 32, Usage: "concurrent `num` in check mode."},
		cli.BoolFlag{Name: "check-all", Usage: "print unreachable servers in the result table too."},
		cli.StringFlag{Name: "check-mode", Value: "ssh", Usage: "check `mode`: ssh(handshake and auth) or tcp(port only)."},
		cli.BoolFlag{Name: "check-strict", Usage: "exit with code 1 if any server is unreachable."},
	}
	app.EnableBashCompletion = true
	app.HideHelp = true
	app.Action = lsshAction

	return app
}

// lsshAction actions ssh functions.
func lsshAction(c *cli.Context) error {
	common.CheckHelpFlag(c)

	confpath := c.String("cnf")

	data := conf.ReadConf(confpath)
	isMulti := parseMultiFlag(c)
	names := data.GetNameSortedList()
	hosts, searchNames := data.ExpandHosts(c, nil)
	if searchNames != nil {
		names = searchNames
	}

	processListFlag(c, names, data)
	processCheckFlag(c, hosts, names, data, confpath)

	r := sshcmd.NewRun(confpath)
	r.ServerList = parseSelected("bssh>>", hosts, names, data, isMulti)
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

	if err := dealPortForward(c, r); err != nil {
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

func processListFlag(c *cli.Context, names []string, cnf conf.Config) {
	if !c.Bool("list") {
		return
	}

	cnf.PrintServerList(names, true)
	os.Exit(0)
}

// processCheckFlag 处理 --check: 探测全部（或 -H 指定的）主机连通性，输出可达列表后退出。
func processCheckFlag(c *cli.Context, hosts, names []string, cnf conf.Config, confpath string) {
	if !c.Bool("check") {
		return
	}

	servers := names
	if h := nonEmptyHosts(hosts); len(h) > 0 {
		servers = h
	}

	r := sshcmd.NewRun(confpath)
	r.Conf = cnf

	results := r.CheckHosts(servers, sshcmd.CheckOptions{
		Timeout:  time.Duration(c.Int("check-timeout")) * time.Second,
		Parallel: c.Int("check-parallel"),
		ShowAll:  c.Bool("check-all"),
		Mode:     c.String("check-mode"),
	})

	if c.Bool("check-strict") {
		for _, v := range results {
			if !v.OK {
				os.Exit(1)
			}
		}
	}

	os.Exit(0)
}

func nonEmptyHosts(hosts []string) []string {
	result := make([]string, 0, len(hosts))
	for _, h := range hosts {
		if strings.TrimSpace(h) != "" {
			result = append(result, h)
		}
	}

	return result
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

func parseSelected(prompt string, hosts, names []string, data conf.Config, isMulti bool) []string {
	if len(hosts) == 0 {
		return list.ShowServersView(&data, prompt, names, isMulti)
	}

	return hosts
}
