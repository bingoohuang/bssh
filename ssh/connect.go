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

	addr := serverConfig.Addr
	port := serverConfig.Port
	// // 应对场景，金良小主机 IP 经常发生变化，可以通过 IP2 环境变量来重置配置中的 IP
	if ip2 := os.Getenv("IP2"); ip2 != "" {
		if strings.HasPrefix(ip2, ":") {
			port = ip2[1:]
			log.Printf("replace port %s by $IP2: %s", serverConfig.Port, ip2)
		} else if strings.Contains(ip2, ":") {
			idx := strings.LastIndex(ip2, ":")
			addr = ip2[:idx]
			port = ip2[idx+1:]
			log.Printf("replace %s:%s by $IP2: %s", serverConfig.Addr, serverConfig.Port, ip2)
		} else {
			addr = ip2
			log.Printf("replace %s by $IP2: %s", addr, ip2)
		}
	}
	authMethods := r.serverAuthMethodMap[serverConfig.ID]
	err = connect.CreateClient(addr, port, serverConfig.User, authMethods, serverConfig.Brg)
	if err != nil && serverConfig.DirectServer {
		r.Conf.WriteTempHosts(serverConfig.ID, server, serverConfig.Pass)
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
