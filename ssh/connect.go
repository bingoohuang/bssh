package ssh

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/bssh/misc"
	"github.com/bingoohuang/bssh/sshlib"
	"github.com/bingoohuang/ngg/gnet"
	"golang.org/x/net/proxy"
)

// CreateSSHConnect return *sshlib.Connect
// this vaule in ssh.Client with proxy.
func (r *Run) CreateSSHConnect(serverConfig *conf.ServerConfig, server string) (connect *sshlib.Connect, err error) {
	if serverConfig == nil {
		serverConfig, err = r.getServerConfig(server)
		if err != nil {
			return nil, err
		}
	}

	// create proxyRoute
	proxyRoute, err := getProxyRoute(server, r.Conf)
	if err != nil {
		return
	}

	// Connect ssh-agent
	if r.agent == nil {
		r.agent = sshlib.ConnectSshAgent()
	}

	// setup dialer
	var dialer proxy.Dialer = gnet.DialerTimeoutBean{ConnTimeout: 10 * time.Second}

	// Connect loop proxy server
	for _, p := range proxyRoute {
		config := r.Conf

		switch p.Type {
		case misc.HTTP, misc.HTTPS, misc.Socks, misc.Socks5:
			c := config.Proxy[p.Name]
			pxy := &sshlib.Proxy{Type: p.Type, Forwarder: dialer, Addr: c.Addr, Port: c.Port, User: c.User, Password: c.Pass}
			dialer, err = pxy.CreateProxyDialer()
		case misc.Command:
			dialer, err = (&sshlib.Proxy{Type: p.Type, Command: p.Name}).CreateProxyDialer()
		default:
			c, name := findServer(config.Server, p.Name)
			pxy := &sshlib.Connect{ProxyDialer: dialer}
			err := pxy.CreateClient(c.Addr, c.Port, c.User, r.serverAuthMethodMap[name], c.Brg)
			if err != nil {
				return connect, err
			}

			dialer = pxy.Client
		}
	}

	if len(proxyRoute) == 0 {
		proxyDialer, err := proxyByEnv(serverConfig, dialer)
		if err != nil {
			return nil, err
		}
		if proxyDialer != nil {
			dialer = proxyDialer
		}
	}

	if err != nil {
		return nil, err
	}

	x11 := serverConfig.X11 || r.X11 // set x11

	// connect target server
	connect = &sshlib.Connect{
		ProxyDialer: dialer, ForwardAgent: serverConfig.SSHAgentUse,
		Agent: r.agent, ForwardX11: x11, TTY: r.IsTerm, ConnectTimeout: serverConfig.ConnectTimeout,
		SendKeepAliveMax: serverConfig.ServerAliveCountMax, SendKeepAliveInterval: serverConfig.ServerAliveCountInterval,
	}

	addr, port := resolveIP2Override(serverConfig.Addr, serverConfig.Port)
	authMethods := r.serverAuthMethodMap[serverConfig.ID]
	err = connect.CreateClient(addr, port, serverConfig.User, authMethods, serverConfig.Brg)
	if err != nil && serverConfig.DirectServer {
		r.Conf.WriteTempHosts(server, serverConfig)
	}

	return connect, err
}

func proxyByEnv(serverConfig *conf.ServerConfig, forwarder proxy.Dialer) (proxy.Dialer, error) {
	// 按优先级顺序检查代理环境变量
	name, env := "PROXY", sshlib.Getenv("PROXY")
	if env == "" {
		name, env = "https_proxy", sshlib.Getenv("https_proxy")
	}
	if env == "" {
		name, env = "http_proxy", sshlib.Getenv("http_proxy")
	}

	if env == "" {
		return nil, nil
	}

	log.Printf("proxy by $%s = %s", name, env)

	if strings.HasPrefix(env, "command://") {
		val := strings.TrimPrefix(env, "command://")
		val = expandProxyCommand(val, *serverConfig)
		dialer, err := (&sshlib.Proxy{Type: misc.Command, Command: val}).CreateProxyDialer()
		if err != nil {
			return nil, fmt.Errorf("create command proxy %q dialer: %w", val, err)
		}
		return dialer, nil
	}

	p, err := url.Parse(env)
	if err != nil {
		return nil, fmt.Errorf("url parse %q: %w", env, err)
	}

	pxy := &sshlib.Proxy{Type: p.Scheme, Forwarder: forwarder, Addr: p.Hostname(), Port: p.Port()}
	if p.User != nil {
		pxy.User = p.User.Username()
		pxy.Password, _ = p.User.Password()
	}
	dialer, err := pxy.CreateProxyDialer()
	if err != nil {
		return nil, fmt.Errorf("create proxy dialer %q: %w", env, err)
	}

	return dialer, nil
}

func findServer(servers map[string]conf.ServerConfig, name string) (conf.ServerConfig, string) {
	c, ok := servers[name]
	if !ok {
		log.Fatalf("fail to find server : %s", name)
	}

	return c, name
}

// proxyRouteData is proxy struct.
type proxyRouteData struct {
	Name string
	Type string
	Port string
}

// getProxyList return []*pxy function.
func getProxyRoute(server string, config conf.Config) (proxyRoute []*proxyRouteData, err error) {
	var conName, conType, proxyName, proxyType, proxyPort string

	isOk := false

	conName, conType = server, misc.SSH

proxyLoop:
	for {
		switch conType {
		case misc.HTTP, misc.HTTPS, misc.Socks, misc.Socks5:
			var conConf conf.ProxyConfig
			conConf, isOk = config.Proxy[conName]
			proxyName, proxyType, proxyPort = conConf.Proxy, conConf.ProxyType, conConf.Port
		case misc.Command:
			break proxyLoop
		default:
			var conConf conf.ServerConfig
			conConf, isOk = config.Server[conName]

			// If ProxyCommand is set, give priority to that
			switch proxyCommand := conConf.ProxyCommand; proxyCommand {
			case "", "none":
				proxyName, proxyType, proxyPort = conConf.Proxy, conConf.ProxyType, conConf.Port
			default:
				proxyName, proxyType, proxyPort = expandProxyCommand(proxyCommand, conConf), misc.Command, ""
			}
		}

		// not use proxy
		if proxyName == "" {
			break
		}

		if !isOk {
			err = fmt.Errorf("not Found proxy : %s", server)
			return nil, err
		}

		p := &proxyRouteData{Name: proxyName}
		switch proxyType {
		case misc.HTTP, misc.HTTPS, misc.Socks, misc.Socks5, misc.Command:
			p.Type = proxyType
		default:
			p.Type = misc.SSH
		}
		p.Port = proxyPort

		proxyRoute = append(proxyRoute, p)
		conName, conType = proxyName, proxyType
	}

	// reverse proxy slice
	for i, j := 0, len(proxyRoute)-1; i < j; i, j = i+1, j-1 {
		proxyRoute[i], proxyRoute[j] = proxyRoute[j], proxyRoute[i]
	}

	return proxyRoute, err
}

func expandProxyCommand(proxyCommand string, config conf.ServerConfig) string {
	// replace variable
	proxyCommand = strings.Replace(proxyCommand, "%h", config.Addr, -1)
	proxyCommand = strings.Replace(proxyCommand, "%p", config.Port, -1)
	proxyCommand = strings.Replace(proxyCommand, "%r", config.User, -1)

	return proxyCommand
}

// resolveIP2Override 处理 IP2 环境变量来覆盖配置中的地址和端口。
// 应对场景：金良小主机 IP 经常发生变化，可以通过 IP2 环境变量来重置配置中的 IP。
// 支持以下格式：
//   - ":port" - 仅覆盖端口
//   - "addr:port" - 覆盖地址和端口
//   - "full.ip.address" - 完整 IP 地址，覆盖地址
//   - "partial.ip" - 部分 IP，与原始地址合并（例如：IP2=34 或 IP2=230.34，原始 addr=192.168.230.33 -> 192.168.230.34）
func resolveIP2Override(addr, port string) (string, string) {
	ip2 := os.Getenv("IP2")
	if ip2 == "" {
		return addr, port
	}

	newAddr := ip2
	newPort := port
	if strings.HasPrefix(ip2, ":") { // 情况1：仅覆盖端口 (格式: :port)
		newAddr = addr
		newPort = ip2[1:]
	} else if strings.Contains(ip2, ":") { // 情况2：完整地址和端口 (格式: addr:port)
		idx := strings.LastIndex(ip2, ":")
		newAddr = ip2[:idx]
		newPort = ip2[idx+1:]
	}

	// 检查是否为部分 IP (不包含3个点，即少于4段)
	if dots := strings.Count(newAddr, "."); dots < 3 {
		// 部分 IP，需要与原始地址合并
		newAddr = mergePartialIP(addr, newAddr)
	}

	// 完整 IP 地址
	log.Printf("replace %s:%s by $IP2: %s:%s", addr, port, newAddr, newPort)
	return newAddr, newPort
}

// mergePartialIP 将部分 IP 与原始地址合并。
// 例如：mergePartialIP("192.168.230.33", "34") -> "192.168.230.34"
// 例如：mergePartialIP("192.168.230.33", "230.34") -> "192.168.230.34"
func mergePartialIP(originalAddr, partialIP string) string {
	// 将原始地址和部分 IP 按 '.' 分割
	originalParts := strings.Split(originalAddr, ".")
	partialParts := strings.Split(partialIP, ".")

	// 如果原始地址不是有效的 IPv4 格式，直接返回部分 IP
	if len(originalParts) != 4 {
		return partialIP
	}

	// 从右向左替换：部分 IP 的最后一段替换原始地址的最后一段
	// 例如：partialParts = ["34"] -> 替换 originalParts[3]
	// 例如：partialParts = ["230", "34"] -> 替换 originalParts[2] 和 originalParts[3]
	for i := 0; i < len(partialParts); i++ {
		idx := 4 - len(partialParts) + i
		if idx >= 0 && idx < 4 {
			originalParts[idx] = partialParts[i]
		}
	}

	return strings.Join(originalParts, ".")
}
