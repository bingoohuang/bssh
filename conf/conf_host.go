package conf

import (
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/bingoohuang/gou/str"
	"github.com/urfave/cli"
)

// ExpandHosts expand hosts to comma-separated or wild match (file name pattern)
func (cf *Config) ExpandHosts(c *cli.Context) []string {
	hosts := c.StringSlice("host")
	expanded := make([]string, 0)

	for _, h := range hosts {
		subHosts := str.SplitN(h, ",", true, true)
		for _, sh := range subHosts {
			if _, ok := cf.Server[sh]; ok {
				expanded = append(expanded, sh)
				continue
			}

			count := 0

			if strings.Contains(sh, "*") {
				for k := range cf.Server {
					if ok, _ := filepath.Match(sh, k); ok {
						expanded = append(expanded, k)
						count++
					}
				}
			}

			if count == 0 {
				logrus.Warnf("no servers match  %s", sh)
			}
		}
	}

	return expanded
}
