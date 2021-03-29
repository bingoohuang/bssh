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
			sc := tc.c
			t.createServerConfig(&sc)
			cf.Server[tc.createKey(t.ID, i)] = sc
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

func (t *Tmpl) createServerConfig(c *ServerConfig) {
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

	c.InitialCmd = substituteProps(c.InitialCmd, t.Props)
	c.Note = substituteProps(c.Note, t.Props)
}

func substituteProps(s string, props map[string]string) string {
	if s == "" {
		return s
	}

	for k, v := range props {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}

	return s
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
			s, _ := ParseDirectServer(tmpl)

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
		user, pass = splitBySep(fields[0][0:atPos], []string{":", "/"})
		host, port = splitHostPort(fields[0][atPos+1:])
		props = parseProps(fields[1:])
	} else {
		host, port = splitHostPort(fields[0])
		user, pass = splitBySep(fields[1], []string{":", "/"})
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

	propsExpands := make(map[string]str.Part)
	for k, v := range host.Props {
		expandV := str.MakeExpand(v)
		propsExpands[k] = expandV.MakePart()
		if l := expandV.MaxLen(); maxExpands < l {
			maxExpands = l
		}
	}

	partPropsFn := func(i int) map[string]string {
		m := make(map[string]string)
		for k, v := range propsExpands {
			m[k] = v.Part(i)
		}
		return m
	}

	tmpls := make([]Tmpl, maxExpands)

	for i := 0; i < maxExpands; i++ {
		tmpls[i] = Tmpl{
			ID:       ids.Part(i),
			Host:     hosts.Part(i),
			Port:     ports.Part(i),
			User:     users.Part(i),
			Password: passes.Part(i),
			Props:    partPropsFn(i),
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

func splitBySep(s string, seps []string) (string, string) {
	for _, sep := range seps {
		if strings.Contains(s, sep) {
			return str.Split2(s, sep, false, false)
		}
	}

	return s, ""
}

func splitHostPort(addr string) (string, string) {
	if !strings.Contains(addr, ":") {
		return addr, "22"
	}

	pos := strings.Index(addr, ":")

	return addr[0:pos], addr[pos+1:]
}
