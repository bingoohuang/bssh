// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// This file describes the code of the built-in command used by lsftp.

package sftp

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/blacknon/lssh/misc"

	"github.com/blacknon/lssh/common"
	"github.com/pkg/sftp"
	"github.com/urfave/cli"
)

// chgrp
func (r *RunSftp) chgrp(args []string) {
	// create app
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	// set help message
	app.CustomAppHelpTemplate = helptext
	app.Name = misc.Chgrp
	app.Usage = "lsftp build-in command: chgrp [remote machine chgrp]"
	app.ArgsUsage = "[group path]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = r.chgrpAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)
	_ = app.Run(args)
}

func (r *RunSftp) chgrpAction(c *cli.Context) error {
	if len(c.Args()) != 2 {
		fmt.Println("Requires two arguments")
		fmt.Println("chgrp group path")

		return nil
	}

	exit := make(chan bool)

	for s, cl := range r.Client {
		server, client := s, cl

		group := c.Args()[0]
		path := c.Args()[1]

		go func() {
			defer func() { exit <- true }()

			// get writer
			client.Output.Create(server)
			w := client.Output.NewWriter()

			// set arg path
			if !filepath.IsAbs(path) {
				path = filepath.Join(client.Pwd, path)
			}

			groupid, err := strconv.Atoi(group)
			var gid, uid int
			if err != nil {
				// read /etc/group
				groupFile, err := client.Connect.Open("/etc/group")
				if err != nil {
					fmt.Fprintf(w, "%s\n", err)
					return
				}
				groupByte, err := ioutil.ReadAll(groupFile)
				if err != nil {
					fmt.Fprintf(w, "%s\n", err)
					return
				}
				groups := string(groupByte)

				// get gid
				gid32, err := common.GetIDFromName(groups, group)
				if err != nil {
					fmt.Fprintf(w, "%s\n", err)
					return
				}

				gid = int(gid32)
			} else {
				gid = groupid
			}

			// ge`t current uid
			stat, err := client.Connect.Lstat(path)
			if err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}

			sys := stat.Sys()
			if fstat, ok := sys.(*sftp.FileStat); ok {
				uid = int(fstat.UID)
			}

			// set gid
			if err = client.Connect.Chown(path, uid, gid); err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}

			fmt.Fprintf(w, "chgrp: set %s's group as %s\n", path, group)
		}()
	}

	for range r.Client {
		<-exit
	}

	return nil
}
