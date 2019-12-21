package common

import (
	"os"

	"github.com/urfave/cli"
)

// CheckHelpFlag checks the help flag then show app help and exit
func CheckHelpFlag(c *cli.Context) {
	if c.Bool("help") {
		_ = cli.ShowAppHelp(c)

		os.Exit(0)
	}
}
