// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"fmt"

	"github.com/blacknon/lssh/common"
	"github.com/urfave/cli"
)

//
func (r *RunSftp) rename(args []string) {
	// create app
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	// set help message
	app.CustomAppHelpTemplate = helptext
	app.Name = "rename"
	app.Usage = "lsftp build-in command: rename [remote machine rename]"
	app.ArgsUsage = "[path path]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true

	// action
	app.Action = func(c *cli.Context) error {
		if len(c.Args()) != 2 {
			fmt.Println("Requires two arguments")
			fmt.Println("rename [old] [new]")
			return nil
		}

		exit := make(chan bool)
		for s, cl := range r.Client {
			server := s
			client := cl

			oldname := c.Args()[0]
			newname := c.Args()[1]

			go func() {
				// get writer
				client.Output.Create(server)
				w := client.Output.NewWriter()

				// get current directory
				err := client.Connect.Rename(oldname, newname)
				if err != nil {
					fmt.Fprintf(w, "%s\n", err)
					exit <- true
					return
				}

				fmt.Fprintf(w, "rename: %s => %s\n", oldname, newname)
				exit <- true
				return
			}()
		}

		for i := 0; i < len(r.Client); i++ {
			<-exit
		}

		return nil
	}

	// parse short options
	args = common.ParseArgs(app.Flags, args)
	app.Run(args)

	return
}
