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
			if len(tc.t) > 1 {
				key += fmt.Sprintf("-%d", i+1)
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
		c.Note += "-"
	}

	c.Note += c.User + "@" + c.Addr + ":" + c.Port
}

// Tmpl represents the structure of remote host information for ssh.
type Tmpl struct {
	Host     string
	Port     string
	User     string
	Password string // empty when using public key
}

// ParseTmpl parses the tmpl.
func ParseTmpl(tmpl string) []Tmpl {
	hosts := make([]Tmpl, 0)

	fields := str.FieldsX(tmpl, "(", ")", -1)
	if len(fields) < 2 {
		logrus.Warnf("bad format for host %s", tmpl)
		return hosts
	}

	host, port := parseHostPort(fields[0])
	user, pass := parseUserPass(fields[1])

	t := Tmpl{Host: host, Port: port, User: user, Password: pass}
	expanded := expandTmpls(t)
	hosts = append(hosts, expanded...)

	return hosts
}

func expandTmpls(host Tmpl) []Tmpl {
	hosts := str.MakeExpand(host.Host).MakePart()
	ports := str.MakeExpand(host.Port).MakePart()
	users := str.MakeExpand(host.User).MakePart()
	passes := str.MakeExpand(host.Password).MakePart()
	maxExpands := mat.MaxInt(hosts.Len(), ports.Len(), users.Len(), passes.Len())

	tmpls := make([]Tmpl, maxExpands)

	for i := 0; i < maxExpands; i++ {
		tmpls[i] = Tmpl{
			Host:     hosts.Part(i),
			Port:     ports.Part(i),
			User:     users.Part(i),
			Password: passes.Part(i)}
	}

	return tmpls
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
