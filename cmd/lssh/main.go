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
	// ssh/scp/ftp
	var ap *cli.App
	args := os.Args
	if len(args) > 0 {
		switch args[0] {
		case "scp":
			args = args[1:]
			ap = app.Lscp()
		case "ftp":
			args = args[1:]
			ap = app.Lsftp()
		case "ssh":
			args = args[1:]
			ap = app.Lssh()
		default:
			ap = app.Lssh()
		}
	}

	ap.Run(common.ParseArgs(ap.Flags, args))
}
