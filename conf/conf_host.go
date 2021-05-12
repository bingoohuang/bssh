package conf

import (
	"sort"
	"strings"

	"github.com/bingoohuang/gou/str"
	"github.com/urfave/cli"
)

type ArgOption struct {
	Name   string
	Short  string
	Values []string
}

type ArgOptions []*ArgOption

func (r ArgOptions) FindByName(name string) *ArgOption {
	for _, op := range r {
		if op.Name == name {
			return op
		}
	}
	return nil
}

func (r ArgOptions) FindByShort(name string) *ArgOption {
	for _, op := range r {
		if op.Short == name {
			return op
		}
	}
	return nil
}

func (r ArgOptions) Values(name string) []string {
	o := r.FindByName(name)
	if o == nil {
		return nil
	}

	return o.Values
}

func ParseMoreOptions(args []string) ([]string, ArgOptions) {
	options := ArgOptions([]*ArgOption{{Name: "host", Short: "H"}})
	newArgs := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		var op *ArgOption
		if strings.HasPrefix(arg, "--") {
			op = options.FindByName(arg[2:])
		} else if strings.HasPrefix(arg, "-") {
			op = options.FindByShort(arg[1:])
		}

		if op != nil && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			op.Values = append(op.Values, args[i+1])
			i++
		} else {
			newArgs = append(newArgs, arg)
		}
	}

	return newArgs, options
}

// ExpandHosts expand hosts to comma-separated or wild match (file name pattern).
func (cf *Config) ExpandHosts(c *cli.Context, options *ArgOptions) ([]string, []string) {
	hosts := c.StringSlice("host")
	if options != nil {
		hosts = append(hosts, options.Values("host")...)
	}

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
