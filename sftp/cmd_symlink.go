// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"fmt"
	"path/filepath"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/misc"
	"github.com/urfave/cli"
)

// TDXX(blacknon): 転送時の進捗状況を表示するプログレスバーの表示はさせること.
func (r *RunSftp) symlink(args []string) {
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	app.CustomAppHelpTemplate = helptext
	app.Name = misc.Symlink
	app.Usage = "bssh ftp build-in command: symlink [remote machine symlink]"
	app.ArgsUsage = "[source target]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = r.symlinkAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)

	_ = app.Run(args)
}

func (r *RunSftp) symlinkAction(c *cli.Context) error {
	if len(c.Args()) != 2 { // nolint:gomnd
		fmt.Println("Requires two arguments")
		fmt.Println("symlink source target")

		return nil
	}

	exit := make(chan bool)

	for s, cl := range r.Client {
		server, client := s, cl

		source := c.Args()[0]
		target := c.Args()[1]

		go func() {
			defer func() { exit <- true }()

			// get writer
			client.Output.Create(server)
			w := client.Output.NewWriter()

			// set arg path
			if !filepath.IsAbs(source) {
				source = filepath.Join(client.Pwd, source)
			}

			if !filepath.IsAbs(target) {
				target = filepath.Join(client.Pwd, target)
			}

			if err := client.Connect.Symlink(source, target); err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}
		}()
	}

	for range r.Client {
		<-exit
	}

	return nil
}
