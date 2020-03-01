// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"fmt"
	"path/filepath"

	"github.com/bingoohuang/bssh/common"
	"github.com/urfave/cli"
)

//
func (r *RunSftp) rm(args []string) {
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	// set help message
	app.CustomAppHelpTemplate = helptext

	// set parameter
	// TDXX(blacknon): walkerでPATHを取得して各個削除する
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "r", Usage: "remove directories and their contents recursively"},
	}

	app.Name = "rm"
	app.Usage = "bssh ftp build-in command: rm [remote machine rm]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = r.rmAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)

	_ = app.Run(args)
}

func (r *RunSftp) rmAction(c *cli.Context) error {
	if len(c.Args()) != 1 { // nolint gomnd
		fmt.Println("Requires one arguments")
		fmt.Println("rm [path]")

		return nil
	}

	exit := make(chan bool)

	for s, cl := range r.Client {
		go r.doRM(exit, s, cl, c.Args()[0], c)
	}

	for range r.Client {
		<-exit
	}

	return nil
}

func (r *RunSftp) doRM(exit chan bool, server string, client *Connect, path string, c *cli.Context) {
	defer func() { exit <- true }()

	// get writer
	client.Output.Create(server)
	w := client.Output.NewWriter()

	// set arg path
	if !filepath.IsAbs(path) {
		path = filepath.Join(client.Pwd, path)
	}

	// get current directory
	if c.Bool("r") {
		// create walker
		walker := client.Connect.Walk(path)

		var data []string

		for walker.Step() {
			err := walker.Err()
			if err != nil {
				fmt.Fprintf(w, "Error: %s\n", err)
				return
			}

			p := walker.Path()
			data = append(data, p)
		}

		// reverse slice
		for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 { // nolint gomnd
			data[i], data[j] = data[j], data[i]
		}

		for _, p := range data {
			err := client.Connect.Remove(p)
			if err != nil {
				fmt.Fprintf(w, "%s\n", err)
				return
			}
		}
	} else {
		err := client.Connect.Remove(path)
		if err != nil {
			fmt.Fprintf(w, "%s\n", err)
			return
		}
	}

	fmt.Fprintf(w, "remove: %s\n", path)
}
