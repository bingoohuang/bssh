package conf

import (
	"fmt"
	"strings"

	"github.com/bingoohuang/gou/mat"
	"github.com/bingoohuang/gou/str"
	"github.com/sirupsen/logrus"
)

type tmplConfig struct {
	k string
	c ServerConfig
	t []Tmpl
}

func (cf *Config) tmplServers(tmplConfigs []tmplConfig) {
	for _, tc := range tmplConfigs {
		for i, t := range tc.t {
			t.createServerConfig(&tc.c)

			cf.Server[tc.createKey(t.ID, i)] = tc.c
		}
	}
}

func (tc tmplConfig) createKey(tid string, i int) string {
	if tid != "" {
		return tc.k + tid
	}

	if len(tc.t) > 1 {
		return tc.k + fmt.Sprintf("%d", i+1)
	}

	return tc.k
}

func (t Tmpl) createServerConfig(c *ServerConfig) {
	c.Addr = t.Host
	c.Port = t.Port
	c.User = t.User
	c.Pass = t.Password

	if v := t.Props["proxy"]; v != "" && c.Proxy == "" {
		c.Proxy = v
	}

	if v := t.Props["group"]; v != "" && len(c.Group) == 0 {
		c.Group = str.SplitTrim(v, ",")
	}

	if v := t.Props["note"]; v != "" && c.Note == "" {
		c.Note = v
	}
}

// Tmpl represents the structure of remote host information for ssh.
type Tmpl struct {
	ID       string
	Host     string
	Port     string
	User     string
	Password string // empty when using public key

	Props map[string]string
}

// ParseTmpl parses the tmpl.
func ParseTmpl(tmpl string) []Tmpl {
	hosts := make([]Tmpl, 0)

	fields := str.FieldsX(tmpl, "(", ")", -1)
	if len(fields) < 2 { // nolint:gomnd
		if IsDirectServer(tmpl) {
			s := ParseDirectServer(tmpl)

			return []Tmpl{{
				ID:       "",
				Host:     s.Addr,
				Port:     s.Port,
				User:     s.User,
				Password: s.Pass,
				Props:    make(map[string]string)},
			}
		}

		logrus.Warnf("bad format for host %s", tmpl)

		return hosts
	}

	host, port, user, pass := "", "", "", ""

	var props map[string]string

	if atPos := strings.LastIndex(fields[0], "@"); atPos > 0 {
		user, pass = splitBySep(fields[0][0:atPos], ":")
		host, port = splitHostPort(fields[0][atPos+1:])
		props = parseProps(fields[1:])
	} else {
		host, port = splitHostPort(fields[0])
		user, pass = splitBySep(fields[1], "/")
		props = parseProps(fields[2:])
	}

	t := Tmpl{ID: props["id"], Host: host, Port: port, User: user, Password: pass, Props: props}
	hosts = append(hosts, expandTmpl(t)...)

	return hosts
}

func expandTmpl(host Tmpl) []Tmpl {
	hosts := str.MakeExpand(host.Host).MakePart()
	ports := str.MakeExpand(host.Port).MakePart()
	users := str.MakeExpand(host.User).MakePart()
	passes := str.MakeExpand(host.Password).MakePart()
	ids := str.MakeExpand(host.ID).MakePart()
	maxExpands := mat.MaxInt(hosts.Len(), ports.Len(), users.Len(), passes.Len(), ids.Len())

	tmpls := make([]Tmpl, maxExpands)

	for i := 0; i < maxExpands; i++ {
		tmpls[i] = Tmpl{
			ID:       ids.Part(i),
			Host:     hosts.Part(i),
			Port:     ports.Part(i),
			User:     users.Part(i),
			Password: passes.Part(i),
			Props:    host.Props,
		}
	}

	return tmpls
}

func parseProps(fields []string) map[string]string {
	props := make(map[string]string)

	for i := 0; i < len(fields); i++ {
		k, v := str.Split2(fields[i], "=", true, true)
		props[k] = v
	}

	return props
}

func splitBySep(userPass, sep string) (string, string) {
	return str.Split2(userPass, sep, false, false)
}

func splitHostPort(addr string) (string, string) {
	if !strings.Contains(addr, ":") {
		return addr, "22"
	}

	pos := strings.Index(addr, ":")

	return addr[0:pos], addr[pos+1:]
}
