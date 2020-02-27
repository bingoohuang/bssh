// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package scp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/bssh/output"
	sshl "github.com/bingoohuang/bssh/ssh"
	"github.com/pkg/sftp"
	"github.com/vbauerster/mpb"
	"golang.org/x/crypto/ssh"
)

const (
	oprompt = "${SERVER} :: "
)

// Scp ...
type Scp struct {
	// ssh Run
	Run *sshl.Run

	// From and To data
	From Info
	To   Info

	Config  conf.Config
	AuthMap map[sshl.AuthKey][]ssh.AuthMethod

	// send parallel flag
	Parallel    bool
	ParallelNum int

	// progress bar
	Progress   *mpb.Progress
	ProgressWG *sync.WaitGroup
}

// Info ...
type Info struct {
	// is remote flag
	IsRemote bool

	// connect server list
	Server []string

	// path list
	Path []string
}

// Connect ...
type Connect struct {
	// server name
	Server string

	// ssh connect
	Connect *sftp.Client

	// Output
	Output *output.Output
}

// PathSet ...
type PathSet struct {
	Base      string
	PathSlice []string
}

// Start scp, switching process.
func (cp *Scp) Start(confpath string) {
	slist := append(cp.To.Server, cp.From.Server...)

	cp.Run = sshl.NewRun(confpath)
	cp.Run.ServerList = slist
	cp.Run.Conf = cp.Config
	cp.Run.CreateAuthMethodMap()

	// Create Progress bar struct
	cp.ProgressWG = new(sync.WaitGroup)
	cp.Progress = mpb.New(mpb.WithWaitGroup(cp.ProgressWG))

	switch {
	// remote to remote
	case cp.From.IsRemote && cp.To.IsRemote:
		cp.viaPush()

	// remote to local
	case cp.From.IsRemote && !cp.To.IsRemote:
		cp.pull()

	// local to remote
	case !cp.From.IsRemote && cp.To.IsRemote:
		cp.push()
	}
}

// push data from local to remote machine
func (cp *Scp) push() {
	// set target hosts
	targets := cp.To.Server

	// create channel
	exit := make(chan bool)

	// create connection parallel
	clients := cp.createScpConnects(targets)
	if len(clients) == 0 {
		fmt.Fprintf(os.Stderr, "There is no host to connect to\n")
		return
	}

	// get local host directory walk data
	pathset := make([]PathSet, len(cp.From.Path))

	for i, p := range cp.From.Path {
		data, err := common.WalkDir(p)
		if err != nil {
			continue
		}

		sort.Strings(data)

		pathset[i] = PathSet{Base: filepath.Dir(p), PathSlice: data}
	}

	// parallel push data
	for _, c := range clients {
		client := c

		go func() {
			defer func() { exit <- true }()

			client.Output.Create(client.Server)
			ow := client.Output.NewWriter()
			ftp := client.Connect

			// push path
			for _, p := range pathset {
				base := p.Base

				for _, path := range p.PathSlice {
					_ = cp.pushPath(ftp, ow, client.Output, base, path)
				}
			}
		}()
	}

	// wait send data
	for range clients {
		<-exit
	}

	// wait 0.3 sec
	time.Sleep(300 * time.Millisecond) // nolint gomnd

	// exit messages
	fmt.Println("all push exit.")
}

//
func (cp *Scp) pushPath(ftp *sftp.Client, ow io.Writer, output *output.Output, base, path string) (err error) {
	// get rel path
	relpath, _ := filepath.Rel(base, path)
	rpath := filepath.Join(cp.To.Path[0], relpath)

	// get local file info
	fInfo, _ := os.Lstat(path)
	if fInfo.IsDir() {
		err = ftp.Mkdir(rpath)
		if err != nil {
			fmt.Fprintf(ow, "%s\n", err)
			return err
		}
	} else {
		lf, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(ow, "%s\n", err)
			return err
		}

		defer lf.Close()

		// get file size
		lstat, _ := os.Lstat(path)
		size := lstat.Size()

		err = cp.pushFile(lf, ftp, output, rpath, size)
		if err != nil {
			fmt.Fprintf(ow, "%s\n", err)
			return err
		}
	}

	return ftp.Chmod(rpath, fInfo.Mode())
}

// pushfile put file to path.
func (cp *Scp) pushFile(lf io.Reader, ftp *sftp.Client, output *output.Output, path string, size int64) (err error) {
	ow := output.NewWriter()

	dir := filepath.Dir(path)
	err = ftp.MkdirAll(dir)

	if err != nil {
		fmt.Fprintf(ow, "%s\n", err)
		return
	}

	rf, err := ftp.OpenFile(path, os.O_RDWR|os.O_CREATE)
	if err != nil {
		fmt.Fprintf(ow, "%s\n", err)
		return
	}

	defer rf.Close()

	rd := io.TeeReader(lf, rf)

	// copy to data
	cp.ProgressWG.Add(1) // nolint gomnd
	output.ProgressPrinter(size, rd, path)

	return
}

func (cp *Scp) viaPush() {
	from := cp.From.Server[0] // string
	to := cp.To.Server        // []string

	fclient := cp.createScpConnects([]string{from})
	tclient := cp.createScpConnects(to)

	if len(fclient) == 0 || len(tclient) == 0 {
		fmt.Fprintf(os.Stderr, "There is no host to connect to\n")
		return
	}

	// pull and push data
	for _, path := range cp.From.Path {
		cp.viaPushPath(path, fclient[0], tclient)
	}

	// wait 0.3 sec
	time.Sleep(300 * time.Millisecond) // nolint gomnd

	// exit messages
	fmt.Println("all push exit.")
}

func (cp *Scp) viaPushPath(path string, fclient *Connect, tclients []*Connect) {
	// from ftp client
	ftp := fclient.Connect

	// create from sftp walker
	walker := ftp.Walk(path)

	// get from sftp output writer
	fclient.Output.Create(fclient.Server)
	fow := fclient.Output.NewWriter()

	for walker.Step() {
		err := walker.Err()
		if err != nil {
			fmt.Fprintf(fow, "Error: %s\n", err)
			continue
		}

		p := walker.Path()
		stat := walker.Stat()

		if stat.IsDir() { // is directory
			for _, tc := range tclients {
				_ = tc.Connect.Mkdir(p)
			}
		} else { // is file
			func() {
				// open from server file
				file, err := ftp.Open(p)
				if err != nil {
					fmt.Fprintf(fow, "Error: %s\n", err)
					return
				}
				defer file.Close()

				size := stat.Size()

				exit := make(chan bool)
				for _, tc := range tclients {
					tclient := tc

					go func() {
						defer func() { exit <- true }()

						tclient.Output.Create(tclient.Server)

						_ = cp.pushFile(file, tclient.Connect, tclient.Output, p, size)
					}()
				}

				for range tclients {
					<-exit
				}
			}()
		}
	}
}

func (cp *Scp) pull() {
	// set target hosts
	targets := cp.From.Server

	// create channel
	exit := make(chan bool)

	// create connection parallel
	clients := cp.createScpConnects(targets)
	if len(clients) == 0 {
		fmt.Fprintf(os.Stderr, "There is no host to connect to\n")
		return
	}

	// parallel push data
	for _, c := range clients {
		client := c

		go func() {
			defer func() { exit <- true }()

			cp.pullPath(client)
		}()
	}

	// wait send data
	for range clients {
		<-exit
	}

	// wait 0.3 sec
	time.Sleep(300 * time.Millisecond) // nolint gomnd

	// exit messages
	fmt.Println("all pull exit.")
}

// walk return file path list ([]string).
func (cp *Scp) pullPath(client *Connect) {
	// set ftp client
	ftp := client.Connect

	// get output writer
	client.Output.Create(client.Server)
	ow := client.Output.NewWriter()

	// basedir
	baseDir := cp.To.Path[0]

	// if multi pull, servername add baseDir
	if len(cp.From.Server) > 1 { // nolint gomnd
		baseDir = filepath.Join(baseDir, client.Server)
		_ = os.MkdirAll(baseDir, 0755)
	}

	baseDir, _ = filepath.Abs(baseDir)

	// walk remote path
	for _, path := range cp.From.Path {
		globpath, err := ftp.Glob(path)
		if err != nil {
			fmt.Fprintf(ow, "Error: %s\n", err)
			continue
		}

		for _, gp := range globpath {
			walker := ftp.Walk(gp)
			for walker.Step() {
				// basedir
				remoteBase := filepath.Dir(gp)

				err := walker.Err()
				if err != nil {
					fmt.Fprintf(ow, "Error: %s\n", err)
					continue
				}

				p := walker.Path()
				rp, _ := filepath.Rel(remoteBase, p)
				lpath := filepath.Join(baseDir, rp)

				stat := walker.Stat()
				if stat.IsDir() { // create dir
					_ = os.MkdirAll(lpath, 0755)
				} else { // create file
					cp.createFile(stat, p, ow, lpath, client)
				}

				_ = os.Chmod(lpath, stat.Mode())
			}
		}
	}
}

func (cp *Scp) createFile(stat os.FileInfo, p string, ow io.Writer, lpath string, client *Connect) {
	size := stat.Size()
	ftp := client.Connect

	// open remote file
	rf, err := ftp.Open(p)
	if err != nil {
		fmt.Fprintf(ow, "Error: %s\n", err)
		return
	}

	defer rf.Close()

	// open local file
	lf, err := os.OpenFile(lpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(ow, "Error: %s\n", err)
		return
	}

	defer lf.Close()

	rd := io.TeeReader(rf, lf)

	cp.ProgressWG.Add(1) // nolint gomnd
	client.Output.ProgressPrinter(size, rd, p)
}

// createScpConnects return []*ScpConnect.
func (cp *Scp) createScpConnects(targets []string) (result []*Connect) {
	ch := make(chan bool)
	m := new(sync.Mutex)

	for _, target := range targets {
		server := target

		go func() {
			defer func() { ch <- true }()

			// ssh connect
			conn, err := cp.Run.CreateSSHConnect(server)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s connect error: %s\n", server, err)
				return
			}

			// create sftp client
			ftp, err := sftp.NewClient(conn.Client)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s create client error: %s\n", server, err)
				return
			}

			// create output
			o := &output.Output{
				Templete:   oprompt,
				ServerList: targets,
				Conf:       cp.Config.Server[server],
				AutoColor:  true,
				Progress:   cp.Progress,
				ProgressWG: cp.ProgressWG,
			}

			// create ScpConnect
			scpCon := &Connect{Server: server, Connect: ftp, Output: o}

			// append result
			m.Lock()
			result = append(result, scpCon)
			m.Unlock()
		}()
	}

	// wait
	for range targets {
		<-ch
	}

	return result
}
