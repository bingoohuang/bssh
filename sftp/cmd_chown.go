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

	"github.com/bingoohuang/bssh/misc"

	"github.com/bingoohuang/bssh/common"
	"github.com/pkg/sftp"
	"github.com/urfave/cli"
)

// chown
func (r *RunSftp) chown(args []string) {
	// create app
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	// set help message
	app.CustomAppHelpTemplate = helptext
	app.Name = misc.Chown
	app.Usage = "bssh ftp build-in command: chown [remote machine chown]"
	app.ArgsUsage = "[user path]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = r.chownAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)
	_ = app.Run(args)
}

func (r *RunSftp) chownAction(c *cli.Context) error {
	if len(c.Args()) != 2 {
		fmt.Println("Requires two arguments")
		fmt.Println("chown group path")

		return nil
	}

	exit := make(chan bool)

	for s, cl := range r.Client {
		server, client := s, cl

		user := c.Args()[0]
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

			//
			userid, err := strconv.Atoi(user)
			var gid, uid int
			if err != nil {
				// read /etc/passwd
				passwdFile, err := client.Connect.Open("/etc/passwd")
				if err != nil {
					fmt.Fprintf(w, "%s\n", err)
					return
				}
				passwdByte, err := ioutil.ReadAll(passwdFile)
				if err != nil {
					fmt.Fprintf(w, "%s\n", err)
					return
				}
				passwd := string(passwdByte)

				// get gid
				uid32, err := common.GetIDFromName(passwd, user)
				if err != nil {
					fmt.Fprintf(w, "%s\n", err)
					return
				}

				uid = int(uid32)
			} else {
				uid = userid
			}

			// get current uid
			stat, err := client.Connect.Lstat(path)
			if err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}

			sys := stat.Sys()
			if fstat, ok := sys.(*sftp.FileStat); ok {
				gid = int(fstat.GID)
			}

			// set gid
			err = client.Connect.Chown(path, uid, gid)
			if err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}

			fmt.Fprintf(w, "chown: set %s's user as %s\n", path, user)
		}()
	}

	for range r.Client {
		<-exit
	}

	return nil
}
