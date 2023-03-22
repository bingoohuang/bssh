// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// This file describes the code of the built-in command used by lsftp.

package sftp

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bingoohuang/bssh/common"
	"github.com/urfave/cli"
)

// chmod ...
func (r *RunSftp) chmod(args []string) {
	app := cli.NewApp()

	app.CustomAppHelpTemplate = helptext
	app.Name = "chmod"
	app.Usage = "bssh ftp build-in command: chmod [remote machine chmod]"
	app.ArgsUsage = "[perm path]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = r.chmodAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)

	_ = app.Run(args)
}

func (r *RunSftp) chmodAction(c *cli.Context) error {
	if len(c.Args()) != 2 {
		fmt.Println("Requires two arguments")
		fmt.Println("chmod mode path")

		return nil
	}

	exit := make(chan bool)

	for s, cl := range r.Client {
		server, client := s, cl

		mode, path := c.Args()[0], c.Args()[1]

		go func() {
			defer func() { exit <- true }()

			// get writer
			client.Output.Create(server)
			w := client.Output.NewWriter()

			// set arg path
			if !filepath.IsAbs(path) {
				path = filepath.Join(client.Pwd, path)
			}

			// get mode
			modeint, err := strconv.ParseUint(mode, 8, 32)
			if err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}

			filemode := os.FileMode(modeint)

			// set filemode
			if err = client.Connect.Chmod(path, filemode); err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}

			fmt.Fprintf(w, "chmod: set %s's permission as %o(%s)\n", path, filemode.Perm(), filemode.String())
		}()
	}

	for range r.Client {
		<-exit
	}

	return nil
}
