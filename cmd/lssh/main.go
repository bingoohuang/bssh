// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/blacknon/lssh/app"
	"github.com/blacknon/lssh/common"
	"github.com/urfave/cli"
)

func main() {
	var ap *cli.App

	args := os.Args

	if len(args) > 1 {
		switch args[1] {
		case "scp":
			args = append(args[0:1], args[2:]...)
			ap = app.Lscp()
		case "ftp":
			args = append(args[0:1], args[2:]...)
			ap = app.Lsftp()
		case "ssh":
			args = append(args[0:1], args[2:]...)
			ap = app.Lssh()
		case "pbe":
			args = append(args[0:1], args[2:]...)
			ap = app.Lpbe()
		}
	}

	if ap == nil {
		ap = app.Lssh()
	}

	_ = ap.Run(common.ParseArgs(ap.Flags, args))
}
