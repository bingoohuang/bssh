// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package main

import (
	"os"
	"strings"

	"github.com/bingoohuang/bssh/app"
	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/misc"
	"github.com/bingoohuang/ngg/ss"
	"github.com/spf13/pflag"
	"github.com/urfave/cli"
)

func main() {
	flagSet := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flagSet.SetInterspersed(false)
	flagSet.StringSliceP("host", "H", strings.Split(os.Getenv("HOST"), ","), "connect server names")
	flagSet.StringP("cnf", "c", ss.ExpandHome("~/.bssh/.bssh.toml"), " config file path")
	_ = flagSet.Parse(os.Args[1:])

	var ap *cli.App

	args := os.Args
	nargs := len(os.Args[1:])

	if len(flagSet.Args()) > 0 {
		sub := flagSet.Args()[0]
		i := nargs - flagSet.NArg() + 1
		switch sub {
		case "scp":
			args = append(os.Args[0:1], os.Args[1:i]...)
			args = append(args, flagSet.Args()[1:]...)
			ap = app.Lscp()
		case "ftp":
			args = append(os.Args[0:1], os.Args[1:i]...)
			args = append(args, flagSet.Args()[1:]...)
			ap = app.Lsftp()
		case misc.SSH:
			args = append(os.Args[0:1], os.Args[1:i]...)
			args = append(args, flagSet.Args()[1:]...)
			ap = app.Lssh()
		}
	}

	if ap == nil {
		ap = app.Lssh()
	}

	_ = ap.Run(common.ParseArgs(ap.Flags, args))
}
