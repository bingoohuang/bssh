package app

import (
	"github.com/blacknon/lssh/common"
	"github.com/spf13/pflag"
)

// Last pbe passwords in the conf file
func Last() ([]string, bool) {
	var (
		lastLog *common.LastLogBean
		ok      bool
	)

	if lastLog, ok = common.ReadLastLog(); !ok {
		return nil, false
	}

	pf := pflag.NewFlagSet("last", pflag.ContinueOnError)
	pf.StringSliceP("host", "H", nil, "connect `servername`.")
	_ = pf.Parse(lastLog.Args)
	hosts, _ := pf.GetStringSlice("host")
	diffHosts := make([]string, 0)

	for _, lastHost := range lastLog.ServerNames {
		if !common.Contains(hosts, lastHost) {
			diffHosts = append(diffHosts, lastHost)
		}
	}

	for _, diffHost := range diffHosts {
		lastLog.Args = append(lastLog.Args, "-H", diffHost)
	}

	lastLog.CurrentLastMode = true

	return lastLog.Args, true
}
