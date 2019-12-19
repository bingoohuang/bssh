// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"fmt"

	"github.com/blacknon/lssh/misc"

	"github.com/blacknon/lssh/common"
	"github.com/urfave/cli"
)

func (r *RunSftp) rename(args []string) {
	// create app
	app := cli.NewApp()
	app.CustomAppHelpTemplate = helptext
	app.Name = misc.Rename
	app.Usage = "lsftp build-in command: rename [remote machine rename]"
	app.ArgsUsage = "[path path]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = r.renameAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)
	app.Run(args)
}

func (r *RunSftp) renameAction(c *cli.Context) error {
	if len(c.Args()) != 2 {
		fmt.Println("Requires two arguments")
		fmt.Println("rename [old] [new]")

		return nil
	}

	exit := make(chan bool)

	for s, cl := range r.Client {
		server, client := s, cl

		oldname := c.Args()[0]
		newname := c.Args()[1]

		go func() {
			defer func() { exit <- true }()

			// get writer
			client.Output.Create(server)
			w := client.Output.NewWriter()

			// get current directory
			if err := client.Connect.Rename(oldname, newname); err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}

			fmt.Fprintf(w, "rename: %s => %s\n", oldname, newname)
		}()
	}

	for range r.Client {
		<-exit
	}

	return nil
}
