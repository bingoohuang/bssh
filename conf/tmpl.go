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

	if len(tc.t) > 1 { // nolint gomnd
		return tc.k + fmt.Sprintf("%d", i+1) // nolint gomnd
	}

	return tc.k
}

func (t Tmpl) createServerConfig(c *ServerConfig) {
	c.Addr = t.Host
	c.Port = t.Port
	c.User = t.User
	c.Pass = t.Password

	c.fixNote()

	if proxy := t.Props["proxy"]; proxy != "" && c.Proxy == "" {
		c.Proxy = proxy
	}

	if group := t.Props["group"]; group != "" && len(c.Group) == 0 {
		c.Group = str.SplitTrim(group, ",")
	}
}

func (c *ServerConfig) fixNote() {
	if strings.Contains(c.Note, c.Addr) {
		return
	}

	if c.Note != "" {
		c.Note += " "
	}

	c.Note += c.User + "@" + c.Addr + ":" + c.Port
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
	if len(fields) < 2 { // nolint gomnd
		logrus.Warnf("bad format for host %s", tmpl)
		return hosts
	}

	host, port := parseHostPort(fields[0])
	user, pass := parseUserPass(fields[1])

	props := parseProps(fields[2:])
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

func parseUserPass(userPass string) (string, string) {
	return str.Split2(userPass, "/", false, false)
}

func parseHostPort(addr string) (string, string) {
	if !strings.Contains(addr, ":") {
		return addr, "22"
	}

	pos := strings.Index(addr, ":")

	return addr[0:pos], addr[pos+1:]
}
