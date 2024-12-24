package ssh

import (
	"fmt"
	"log"
	"net/url"
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
		config, ok := r.Conf.Server[server]
		if !ok {
			config = r.parseDirectServer(server)
		}
		serverConfig = &config
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
		// 配置文件中，如果没有配置，查看环境变量 PROXY 是否指定配置
		// 例如：export PROXY=socks5://127.0.0.1:6000
		if proxyEnv := sshlib.Getenv("PROXY"); proxyEnv != "" {
			log.Printf("use env PROXY: %s", proxyEnv)
			p, err := url.Parse(proxyEnv)
			if err != nil {
				return nil, fmt.Errorf("parse $PROXY's value %q error: %w", proxyEnv, err)
			}

			pxy := &sshlib.Proxy{Type: p.Scheme, Forwarder: dialer, Addr: p.Hostname(), Port: p.Port()}
			if p.User != nil {
				pxy.User = p.User.Username()
				pxy.Password, _ = p.User.Password()
			}
			dialer, err = pxy.CreateProxyDialer()
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

	if err = connect.CreateClient(serverConfig.Addr, serverConfig.Port, serverConfig.User, r.serverAuthMethodMap[serverConfig.ID], serverConfig.Brg); err != nil && serverConfig.DirectServer {
		r.Conf.WriteTempHosts(serverConfig.ID, server, serverConfig.Pass)
	}

	return connect, err
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
