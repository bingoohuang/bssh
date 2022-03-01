package ssh

import (
	"fmt"
	"strings"
	"time"

	"github.com/bingoohuang/bssh/misc"

	"github.com/bingoohuang/gonet"

	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/bssh/sshlib"
	"golang.org/x/net/proxy"
)

// CreateSSHConnect return *sshlib.Connect
// this vaule in ssh.Client with proxy.
func (r *Run) CreateSSHConnect(server string) (connect *sshlib.Connect, err error) {
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
	var dialer proxy.Dialer = gonet.DialerTimeoutBean{ConnTimeout: 10 * time.Second} // nolint:gomnd

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
			c := config.Server[p.Name]
			pxy := &sshlib.Connect{ProxyDialer: dialer}
			err := pxy.CreateClient(c.Addr, c.Port, c.User, r.serverAuthMethodMap[p.Name])

			if err != nil {
				return connect, err
			}

			dialer = pxy.Client
		}
	}

	if err != nil {
		return nil, err
	}

	s, ok := r.Conf.Server[server] // server conf
	isTempHost := !ok
	if isTempHost {
		s, _ = r.parseDirectServer(server)
	}

	x11 := s.X11 || r.X11 // set x11

	// connect target server
	connect = &sshlib.Connect{
		ProxyDialer: dialer, ForwardAgent: s.SSHAgentUse,
		Agent: r.agent, ForwardX11: x11, TTY: r.IsTerm, ConnectTimeout: s.ConnectTimeout,
		SendKeepAliveMax: s.ServerAliveCountMax, SendKeepAliveInterval: s.ServerAliveCountInterval,
	}

	err = connect.CreateClient(s.Addr, s.Port, s.User, r.serverAuthMethodMap[server])
	if err != nil && isTempHost {
		r.Conf.WriteTempHosts(server, s.Pass)
	}

	return connect, err
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
			err = fmt.Errorf("not Found proxy : %s", server) // nolint:goerr113
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
