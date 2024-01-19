package conf

import (
	"fmt"
	"strings"

	"github.com/bingoohuang/gossh/pkg/hostparse"
	"github.com/bingoohuang/gou/str"
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

func createServerConfigFromHost(t hostparse.Host, c *ServerConfig) {
	c.Addr = t.Addr
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

	if v := t.Props["id"]; v != "" && c.ID == "" {
		c.ID = v
	}

	if v := t.Props["key"]; v != "" && c.Key == "" {
		c.Key = v
	}

	c.InitialCmd = substituteProps(c.InitialCmd, t.Props)
	c.Note = substituteProps(c.Note, t.Props)
	c.Raw = t.Raw
	c.Host = t
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

func splitBySep(s string, seps []string) (string, string) {
	for _, sep := range seps {
		if strings.Contains(s, sep) {
			return str.Split2(s, sep, false, false)
		}
	}

	return s, ""
}
