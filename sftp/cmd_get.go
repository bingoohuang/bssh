// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/blacknon/lssh/misc"

	"github.com/blacknon/lssh/common"
	"github.com/urfave/cli"
	"github.com/vbauerster/mpb"
)

// TODO(blacknon): リファクタリング(v0.6.1)

func (r *RunSftp) get(args []string) {
	// create app
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	// set help message
	app.CustomAppHelpTemplate = helptext

	// set parameter
	app.Name = misc.Get
	app.Usage = "lsftp build-in command: get"
	app.ArgsUsage = "[source(remote) target(local)]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true

	// action
	app.Action = r.getAction
	// parse short options
	args = common.ParseArgs(app.Flags, args)
	_ = app.Run(args)
}

func (r *RunSftp) pullPath(client *Connect, path, target string) {
	// set arg path
	var rpath string

	switch {
	case filepath.IsAbs(path):
		rpath = path
	case !filepath.IsAbs(path):
		rpath = filepath.Join(client.Pwd, path)
	}

	base := filepath.Dir(rpath)

	// get writer
	ow := client.Output.NewWriter()

	// expantion path
	epath, _ := client.Connect.Glob(rpath)

	// for walk
	for _, ep := range epath {
		walker := client.Connect.Walk(ep)

		for walker.Step() {
			err := walker.Err()
			if err != nil {
				fmt.Fprintf(ow, "Error: %s\n", err)
				continue
			}

			p := walker.Path()
			relpath, _ := filepath.Rel(base, p)
			stat := walker.Stat()

			localpath := filepath.Join(target, relpath)

			//
			if stat.IsDir() { // is directory
				_ = os.Mkdir(localpath, 0755)
			} else if err := pullFile(stat, client, localpath, p, r); err != nil { // is not directory
				fmt.Fprintf(ow, "Error: %s\n", err)
				continue
			}

			_ = os.Chmod(localpath, stat.Mode())
		}
	}
}

func (r *RunSftp) getAction(c *cli.Context) error {
	if len(c.Args()) != 2 {
		fmt.Println("Requires two arguments")
		fmt.Println("get source(remote) target(local)")

		return nil
	}

	// Create Progress
	r.ProgressWG = new(sync.WaitGroup)
	r.Progress = mpb.New(mpb.WithWaitGroup(r.ProgressWG))

	// set path
	source := c.Args()[0]
	target := common.ExpandHomeDir(c.Args()[1])

	// get target directory abs
	target, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return nil
	}

	// mkdir local target directory
	err = os.MkdirAll(target, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return nil
	}

	// get directory data, copy remote to local
	exit := make(chan bool)

	for s, c := range r.Client {
		server, client := s, c

		targetdir := target
		if len(r.Client) > 1 {
			targetdir = filepath.Join(target, server)
			// mkdir local target directory
			err = os.MkdirAll(targetdir, 0755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return nil
			}
		}

		go func() {
			defer func() { exit <- true }()

			// set Progress
			client.Output.Progress = r.Progress
			client.Output.ProgressWG = r.ProgressWG

			// create output
			client.Output.Create(server)

			// local target
			target, _ = filepath.Abs(target)

			r.pullPath(client, source, targetdir)
		}()
	}

	// wait exit
	for range r.Client {
		<-exit
	}

	// wait Progress
	r.Progress.Wait()

	// wait 0.3 sec
	time.Sleep(300 * time.Millisecond)

	return nil
}

func pullFile(stat os.FileInfo, client *Connect, localpath, p string, r *RunSftp) error {
	// get size
	size := stat.Size()

	// open remote file
	remotefile, err := client.Connect.Open(p)
	if err != nil {
		return err
	}

	defer remotefile.Close()

	// open local file
	localfile, err := os.OpenFile(localpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	defer localfile.Close()

	// set tee reader
	rd := io.TeeReader(remotefile, localfile)

	r.ProgressWG.Add(1)
	client.Output.ProgressPrinter(size, rd, p)

	return nil
}
