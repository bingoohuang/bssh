package conf

import (
	"github.com/bingoohuang/gou/str"
	"github.com/urfave/cli"
	"sort"
)

// ExpandHosts expand hosts to comma-separated or wild match (file name pattern).
func (cf *Config) ExpandHosts(c *cli.Context) ([]string, []string) {
	hosts := c.StringSlice("host")
	expanded := make([]string, 0)

	for _, h := range hosts {
		subHosts := str.SplitN(h, ",", true, true)
		for _, sh := range subHosts {
			if _, ok := cf.Server[sh]; ok {
				expanded = append(expanded, sh)
				continue
			}

			host, search := cf.EnsureSearchHost(sh)
			if len(search) > 0 {
				sort.Strings(search)
				return nil, search
			}

			expanded = append(expanded, host)

		}
	}

	return expanded, nil
}
