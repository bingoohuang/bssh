package conf

import (
	"fmt"
	"strings"

	"github.com/bingoohuang/ngg/gossh/pkg/hostparse"
	"github.com/bingoohuang/ngg/ss"
)

type tmplConfig struct {
	k string
	c ServerConfig
	t []hostparse.Host
}

func (cf *Config) tmplServers(tmplConfigs []tmplConfig) {
	for _, tc := range tmplConfigs {
		for i, t := range tc.t {
			sc := tc.c
			createServerConfigFromHost(t, &sc)
			key := tc.createKey(t.ID, i)
			sc.PassPbeEncrypted = strings.HasPrefix(sc.Pass, `{PBE}`)

			cf.Server[key] = sc
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

func createServerConfigFromHost(t hostparse.Host, c *ServerConfig) {
	c.Addr = t.Addr
	c.Port = t.Port
	c.User = t.User
	c.Pass = t.Password

	if v := t.Props["proxy"]; len(v) > 0 && c.Proxy == "" {
		c.Proxy = v[0]
	}

	if v := t.Props["group"]; len(v) > 0 && len(c.Group) == 0 {
		c.Group = ss.Split(v[0], ",")
	}

	if v := t.Props["note"]; len(v) > 0 && c.Note == "" {
		c.Note = v[0]
	}

	if v := t.Props["id"]; len(v) > 0 && c.ID == "" {
		c.ID = v[0]
	}

	if c.ID == "" {
		if v := t.Props["auto_id"]; len(v) > 0 && c.ID == "" {
			c.ID = v[0]
		}
	}

	if v := t.Props["key"]; len(v) > 0 && c.Key == "" {
		c.Key = v[0]
	}

	if v := t.Props["initial_cmd"]; len(v) > 0 && c.InitialCmd == "" {
		c.InitialCmd = v[0]
	}

	if v := t.Props["IP"]; len(v) > 0 {
		c.Note += ss.If(c.Note != "", " ", "") + "IP: " + v[0]
	}

	if v := t.Props["brg"]; len(v) > 0 {
		c.Brg.Set(ss.Pick1(ss.ParseBool(v[0])))
	}

	c.InitialCmd = substituteProps(c.InitialCmd, t.Props)
	c.Note = substituteProps(c.Note, t.Props)
	c.Raw = t.Raw
	c.Host = &t
}

func substituteProps(s string, props map[string][]string) string {
	if s == "" {
		return s
	}

	for k, v := range props {
		s = strings.ReplaceAll(s, "{"+k+"}", v[0])
	}

	return s
}
