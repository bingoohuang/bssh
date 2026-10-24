package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/ngg/gossh/pkg/hostparse"
	"github.com/jedib0t/go-pretty/table"
)

// 连通性检查模式（--check-mode）。
const (
	// CheckModeSSH 完整 SSH 握手 + 认证，判定「能否登录」。默认模式。
	CheckModeSSH = "ssh"
	// CheckModeTCP 只探测端口（仍走配置的代理链），判定「机器是否还在」。
	CheckModeTCP = "tcp"
)

// 连通性检查的默认值。
const (
	DefaultCheckTimeout  = 3 * time.Second
	DefaultCheckParallel = 32
)

// CheckOptions 连通性检查参数。
type CheckOptions struct {
	// Timeout 单台超时，<=0 时使用 DefaultCheckTimeout。
	Timeout time.Duration
	// Parallel 并发数，<=0 时使用 DefaultCheckParallel。
	Parallel int
	// ShowAll 为 true 时不可达主机也一并输出到结果表（默认只输出到 stderr）。
	ShowAll bool
	// Mode 取 CheckModeSSH / CheckModeTCP，空值时为 CheckModeSSH。
	Mode string
}

func (o CheckOptions) normalize() CheckOptions {
	if o.Timeout <= 0 {
		o.Timeout = DefaultCheckTimeout
	}

	if o.Parallel <= 0 {
		o.Parallel = DefaultCheckParallel
	}

	if o.Mode == "" {
		o.Mode = CheckModeSSH
	}

	return o
}

// CheckResult 单台主机的连通性检查结果。
type CheckResult struct {
	Name string
	User string
	Addr string
	Port string
	Note string

	// OK 为 true 表示可连通（ssh 模式下表示握手与认证均成功）。
	OK bool
	// Cost 探测耗时。
	Cost time.Duration
	// Reason 结果分类：ok / timeout / refused / dns / auth failed / error。
	Reason string
	// Err 原始错误，OK 为 true 时是 nil。
	Err error
}

// CheckHosts 并发探测 servers 的连通性，打印结果并返回与 servers 同序的结果列表。
//
// 检查过程中不会进入交互式密码输入，也不会回写配置文件。
func (r *Run) CheckHosts(servers []string, opt CheckOptions) []CheckResult {
	opt = opt.normalize()

	r.CheckMode = opt.Mode
	r.CheckTimeoutSeconds = int(opt.Timeout.Seconds())
	r.ServerList = nonEmptyServers(servers)

	// 认证方式与 ssh-agent 都要在并发之前准备好，避免惰性初始化造成 data race。
	r.CreateAuthMethodMap()
	r.SetupSSHAgent()
	targets := r.ensureServerConfigs(r.ServerList)

	results := make([]CheckResult, len(targets))

	sem := make(chan struct{}, opt.Parallel)
	var wg sync.WaitGroup

	start := time.Now()

	for i, target := range targets {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int, target checkTarget) {
			defer func() {
				<-sem
				wg.Done()
			}()

			results[i] = r.checkOne(target)
		}(i, target)
	}

	wg.Wait()

	printCheckResults(results, opt, time.Since(start))

	return results
}

// checkOne 探测单台主机。只做连接，不创建 session、不执行任何远程命令。
func (r *Run) checkOne(target checkTarget) CheckResult {
	cfg := target.Conf
	res := CheckResult{Name: target.Name, User: cfg.User, Addr: cfg.Addr, Port: cfg.Port, Note: cfg.Note}

	start := time.Now()
	conn, err := r.CreateSSHConnect(&cfg, target.Name)
	res.Cost = time.Since(start)

	if err != nil {
		res.Reason = ClassifyCheckErr(err)
		res.Err = err

		return res
	}

	if conn != nil && conn.Client != nil {
		conn.Client.Close()
	}

	res.OK = true
	res.Reason = "ok"

	return res
}

// checkTarget 一台待检查的主机：Name 为展示与 -H 复用的名字，Conf 为解析后的配置。
type checkTarget struct {
	Name string
	Conf conf.ServerConfig
}

// ensureServerConfigs 把 -H 形式的直连主机补登记进配置，返回待检查的主机
// （无法解析的主机被丢弃）。
func (r *Run) ensureServerConfigs(servers []string) []checkTarget {
	targets := make([]checkTarget, 0, len(servers))

	for _, server := range servers {
		if cfg, ok := r.Conf.Server[server]; ok {
			targets = append(targets, checkTarget{Name: server, Conf: cfg})
			continue
		}

		if _, ok := hostparse.ParseDirectServer(server); !ok {
			fmt.Fprintf(os.Stderr, "skip invalid server: %s\n", server)
			continue
		}

		// 直连主机在配置中以内部 ID 登记，因此要取 parseDirectServer 的返回值。
		targets = append(targets, checkTarget{Name: server, Conf: r.parseDirectServer(server)})
	}

	return targets
}

// ClassifyCheckErr 把连接错误归类成便于阅读的原因。
func ClassifyCheckErr(err error) string {
	if err == nil {
		return "ok"
	}

	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout"
	}

	msg := err.Error()

	switch {
	case strings.Contains(msg, "unable to authenticate"),
		strings.Contains(msg, "no supported methods remain"):
		return "auth failed"
	case strings.Contains(msg, "i/o timeout"),
		strings.Contains(msg, "deadline exceeded"),
		strings.Contains(msg, "timed out"):
		return "timeout"
	case strings.Contains(msg, "connection refused"):
		return "refused"
	case strings.Contains(msg, "no such host"):
		return "dns"
	default:
		return "error"
	}
}

func printCheckResults(results []CheckResult, opt CheckOptions, elapsed time.Duration) {
	reachable, unreachable := splitCheckResults(results)

	fmt.Fprintf(os.Stdout, "bssh Check Result: %d hosts, %d reachable, %d unreachable, mode=%s, elapsed %s\n\n",
		len(results), len(reachable), len(unreachable), opt.Mode, formatDuration(elapsed))

	showed := reachable
	if opt.ShowAll {
		showed = results
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Server Name", "Connect Info", "Status", "Latency", "Reason", "Note"})

	for i, v := range showed {
		status := "NG"
		if v.OK {
			status = "OK"
		}

		t.AppendRow(table.Row{i + 1, v.Name, connectInfoOf(v), status, formatDuration(v.Cost), v.Reason, v.Note})
	}

	t.Render()
	fmt.Fprintln(os.Stdout)

	if len(reachable) > 0 {
		names := strings.Join(namesOf(reachable), ",")
		fmt.Fprintf(os.Stdout, "Reachable(%d): %s\n", len(reachable), names)
		fmt.Fprintf(os.Stdout, "Use: bssh -H %s <command>\n\n", names)
	}

	if len(unreachable) > 0 && !opt.ShowAll {
		fmt.Fprintf(os.Stderr, "Unreachable(%d) details:\n", len(unreachable))
		for _, v := range unreachable {
			fmt.Fprintf(os.Stderr, "  %-24s %-32s %s: %v\n", v.Name, connectInfoOf(v), v.Reason, v.Err)
		}
	}
}

func splitCheckResults(results []CheckResult) (reachable, unreachable []CheckResult) {
	for _, v := range results {
		if v.OK {
			reachable = append(reachable, v)
		} else {
			unreachable = append(unreachable, v)
		}
	}

	return reachable, unreachable
}

func namesOf(results []CheckResult) []string {
	names := make([]string, 0, len(results))
	for _, v := range results {
		names = append(names, v.Name)
	}

	return names
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
}

func connectInfoOf(v CheckResult) string {
	return v.User + "@" + v.Addr + ":" + v.Port
}

func nonEmptyServers(servers []string) []string {
	result := make([]string, 0, len(servers))
	for _, s := range servers {
		if strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}

	return result
}
