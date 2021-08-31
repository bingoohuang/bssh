package bssh

import (
	"fmt"
)

var (
	gitCommit  = ""
	buildTime  = ""
	goVersion  = ""
	appVersion = "1.2.8"
)

func AppVersion() string {
	return fmt.Sprintf("\nversion: %s\n", appVersion) +
		fmt.Sprintf("built:\t%s\n", buildTime) +
		fmt.Sprintf("git:\t%s\n", gitCommit) +
		fmt.Sprintf("go:\t%s\n", goVersion)
}
