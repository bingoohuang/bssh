// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/bingoohuang/bssh/misc"

	"github.com/bingoohuang/bssh/app"
	"github.com/bingoohuang/bssh/common"
	"github.com/urfave/cli"
)

func main() {
	var ap *cli.App

	args := os.Args

HERE:
	if len(args) > 1 { // nolint gomnd
		sub := args[1]
		switch sub {
		case "scp":
			args = append(args[0:1], args[2:]...)
			ap = app.Lscp()
		case "ftp":
			args = append(args[0:1], args[2:]...)
			ap = app.Lsftp()
		case misc.SSH:
			args = append(args[0:1], args[2:]...)
			ap = app.Lssh()
		case "pbe":
			args = append(args[0:1], args[2:]...)
			ap = app.Lpbe()
		case "l", "last":
			args = append(args[0:1], args[2:]...)
			if lastArgs, ok := app.Last(); ok {
				args = lastArgs
				goto HERE
			}
		}
	}

	if ap == nil {
		ap = app.Lssh()
	}

	common.SaveArgsLastLog()

	_ = ap.Run(common.ParseArgs(ap.Flags, args))
}
