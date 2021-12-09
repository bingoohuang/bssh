// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/bingoohuang/bssh/misc"

	"github.com/bingoohuang/bssh/common"
	"github.com/urfave/cli"
	"github.com/vbauerster/mpb"
)

// TDXX(blacknon): リファクタリング(v0.6.1).
func (r *RunSftp) put(args []string) {
	app := cli.NewApp()
	app.CustomAppHelpTemplate = helptext
	app.Name = misc.Put
	app.Usage = "bssh ftp build-in command: put"
	app.ArgsUsage = "[source(local) target(remote)]"
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = r.putAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)
	_ = app.Run(args)
}

func (r *RunSftp) putAction(c *cli.Context) error {
	if len(c.Args()) != 2 { // nolint:gomnd
		fmt.Println("Requires two arguments")
		fmt.Println("put source(local) target(remote)")

		return nil
	}

	// Create Progress
	r.ProgressWG = new(sync.WaitGroup)
	r.Progress = mpb.New(mpb.WithWaitGroup(r.ProgressWG))

	// set path
	source := common.ExpandHomeDir(c.Args()[0])
	target := c.Args()[1]

	data, err := common.WalkDir(source)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return nil
	}

	sort.Strings(data)

	// get local host directory walk data
	pathSet := PathSet{Base: filepath.Dir(source), PathSlice: data}

	// parallel push data
	exit := make(chan bool)

	for s, c := range r.Client {
		server, client := s, c

		go func() {
			defer func() { exit <- true }()

			client.Output.Progress = r.Progress
			client.Output.ProgressWG = r.ProgressWG

			client.Output.Create(server)

			base := pathSet.Base
			data := pathSet.PathSlice

			for _, path := range data {
				if err := r.pushPath(client, target, base, path); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			}
		}()
	}

	// wait exit
	for range r.Client {
		<-exit
	}

	r.Progress.Wait() // wait Progress

	time.Sleep(300 * time.Millisecond) // nolint:gomnd

	return nil
}

func (r *RunSftp) pushPath(client *Connect, target, base, path string) (err error) {
	rpath, _ := filepath.Rel(base, path)

	if filepath.IsAbs(target) {
		rpath = filepath.Join(target, rpath)
	} else {
		target = filepath.Join(client.Pwd, target)
		rpath = filepath.Join(target, rpath)
	}

	fInfo, _ := os.Lstat(path)
	if fInfo.IsDir() { // directory
		_ = client.Connect.Mkdir(rpath)
	} else { //file
		localFile, err := os.Open(path)
		if err != nil {
			return err
		}

		defer localFile.Close()

		if err = r.pushFile(client, localFile, rpath, fInfo.Size()); err != nil {
			return err
		}
	}

	_ = client.Connect.Chmod(rpath, fInfo.Mode())

	return nil
}

// pushFile put file to path.
func (r *RunSftp) pushFile(c *Connect, localFile io.Reader, path string, size int64) (err error) {
	dir := filepath.Dir(path)
	if err := c.Connect.MkdirAll(dir); err != nil {
		return err
	}

	remoteFile, err := c.Connect.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}

	defer remoteFile.Close()

	rd := io.TeeReader(common.CreateRateLimit(localFile), remoteFile)

	r.ProgressWG.Add(1)
	return c.Output.ProgressPrinter(size, rd, path)
	return nil
}
