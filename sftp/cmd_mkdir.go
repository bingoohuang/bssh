// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// This file describes the code of the built-in command used by lsftp.
// It is quite big in that relationship. Maybe it will be separated or repaired soon.

package sftp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/misc"
	"github.com/urfave/cli"
)

func (r *RunSftp) mkdir(args []string) {
	// create app
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	// set parameter
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "p", Usage: "no error if existing, make parent directories as needed"},
	}

	// set help message
	app.CustomAppHelpTemplate = helptext
	app.Name = misc.Mkdir
	app.Usage = "bssh ftp build-in command: mkdir [remote machine mkdir]"
	app.ArgsUsage = misc.Path
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true

	// action
	app.Action = r.mkdirAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)

	_ = app.Run(args)
}

func (r *RunSftp) mkdirAction(c *cli.Context) error {
	// TDXX(blacknon): 複数のディレクトリ受付(v0.6.1以降)
	if len(c.Args()) != 1 {
		fmt.Println("Requires one arguments")
		fmt.Println("mkdir [path]")

		return nil
	}

	exit := make(chan bool)

	for s, cl := range r.Client {
		server, client := s, cl
		path := c.Args()[0]

		go func() {
			defer func() { exit <- true }()

			// get writer
			client.Output.Create(server)
			w := client.Output.NewWriter()

			// set arg path
			if !filepath.IsAbs(path) {
				path = filepath.Join(client.Pwd, path)
			}

			// create directory
			var err error
			if c.Bool("p") {
				err = client.Connect.MkdirAll(path)
			} else {
				err = client.Connect.Mkdir(path)
			}

			// check error
			if err != nil {
				fmt.Fprintf(w, "%s\n", err)
			}

			fmt.Fprintf(w, "make directory: %s\n", path)
		}()
	}

	for range r.Client {
		<-exit
	}

	return nil
}

func (r *RunSftp) lmkdir(args []string) {
	// create app
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	// set parameter
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "p", Usage: "no error if existing, make parent directories as needed"},
	}

	// set help message
	app.CustomAppHelpTemplate = helptext
	app.Name = misc.Lmkdir
	app.Usage = "bssh ftp build-in command: lmkdir [local machine mkdir]"
	app.ArgsUsage = misc.Path
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true

	// action
	app.Action = func(c *cli.Context) error {
		// TDXX(blacknon): 複数のディレクトリ受付(v0.6.1以降)
		if len(c.Args()) != 1 {
			fmt.Println("Requires one arguments")
			fmt.Println("lmkdir [path]")

			return nil
		}

		path := c.Args()[0]

		var err error

		if c.Bool("p") {
			err = os.MkdirAll(path, 0o755)
		} else {
			err = os.Mkdir(path, 0o755)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}

		return nil
	}

	// parse short options
	args = common.ParseArgs(app.Flags, args)

	_ = app.Run(args)
}
