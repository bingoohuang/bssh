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
			c := tc.c
			c.Addr = t.Host
			c.Port = t.Port
			c.User = t.User
			c.Pass = t.Password

			fixNote(&c)

			key := tc.k

			if len(tc.t) > 1 { // nolint gomnd
				if t.ID != "" {
					key += t.ID
				} else {
					key += fmt.Sprintf("%d", i+1) // nolint gomnd
				}
			}

			cf.Server[key] = c
		}
	}
}

func fixNote(c *ServerConfig) {
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
	t := Tmpl{ID: findID(props), Host: host, Port: port, User: user, Password: pass, Props: props}
	expanded := expandTmpls(t)
	hosts = append(hosts, expanded...)

	return hosts
}

func expandTmpls(host Tmpl) []Tmpl {
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
			Password: passes.Part(i)}
	}

	return tmpls
}

func findID(props map[string]string) string {
	if v, ok := props["id"]; ok {
		return v
	}

	return ""
}

func parseProps(fields []string) map[string]string {
	props := make(map[string]string)

	for i := 0; i < len(fields); i++ {
		k, v := str.Split2(fields[i], "=", true, true)
		props[k] = v
	}

	return props
}

func parseUserPass(userpass string) (string, string) {
	return str.Split2(userpass, "/", false, false)
}

func parseHostPort(addr string) (string, string) {
	if !strings.Contains(addr, ":") {
		return addr, "22"
	}

	pos := strings.Index(addr, ":")

	return addr[0:pos], addr[pos+1:]
}
