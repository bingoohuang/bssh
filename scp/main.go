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

// push data from local to remote machine.
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
			fmt.Fprintf(os.Stderr, "common.WalkDir error %v\n", err)
			continue
		}

		sort.Strings(data)

		pathset[i] = PathSet{Base: filepath.Dir(p), PathSlice: data}
	}

	// parallel push data
	for _, c := range clients {
		go pushByClient(exit, c, pathset, cp)
	}

	// wait send data
	for range clients {
		<-exit
	}

	// wait 0.3 sec
	time.Sleep(300 * time.Millisecond) // nolint:gomnd

	// exit messages
	fmt.Println("all push exit.")
}

func pushByClient(exit chan bool, client *Connect, pathset []PathSet, cp *Scp) {
	defer func() { exit <- true }()

	client.Output.Create(client.Server)
	ow := client.Output.NewWriter()
	ftp := client.Connect

	// push path
	for _, p := range pathset {
		for _, path := range p.PathSlice {
			if err := cp.pushPath(ftp, ow, client.Output, p.Base, path); err != nil {
				fmt.Fprintf(os.Stderr, "cp.pushPath error %v\n", err)
			}
		}
	}
}

//
func (cp *Scp) pushPath(ftp *sftp.Client, ow io.Writer, output *output.Output, base, path string) (err error) {
	// get rel path
	relpath, _ := filepath.Rel(base, path)
	rpath := filepath.Join(cp.To.Path[0], relpath)

	// get local file info
	fInfo, _ := os.Lstat(path)
	if fInfo.IsDir() {
		if err := ftp.MkdirAll(rpath); err != nil {
			fmt.Fprintf(ow, "ftp.MkdirAll rpath %s error %v\n", rpath, err)
			return err
		}
	} else {
		lf, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(ow, "os.Open path %s error %v\n", path, err)
			return err
		}

		defer lf.Close()

		// get file size
		lstat, _ := os.Lstat(path)
		size := lstat.Size()

		if err := cp.pushFile(lf, ftp, output, rpath, size); err != nil {
			fmt.Fprintf(ow, "cp.pushFile %s->%s error %v\n", path, rpath, err)
			return err
		}
	}

	return ftp.Chmod(rpath, fInfo.Mode())
}

// pushfile put file to path.
func (cp *Scp) pushFile(lf io.Reader, ftp *sftp.Client, output *output.Output, path string, size int64) (err error) {
	ow := output.NewWriter()

	dir := filepath.Dir(path)
	if err = ftp.MkdirAll(dir); err != nil {
		fmt.Fprintf(ow, "ftp.MkdirAll error %v\n", err)

		return err
	}

	rf, err := ftp.OpenFile(path, os.O_RDWR|os.O_CREATE)
	if err != nil {
		fmt.Fprintf(ow, "ftp.OpenFile error %v\n", err)

		return err
	}

	defer rf.Close()

	rd := io.TeeReader(lf, rf)

	// copy to data
	cp.ProgressWG.Add(1)
	output.ProgressPrinter(size, rd, path)

	return err
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
	time.Sleep(300 * time.Millisecond) // nolint:gomnd

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
			fmt.Fprintf(fow, "Error: %v\n", err)
			continue
		}

		p := walker.Path()
		if common.IsHidden(path, p) {
			continue // ignore hidden files.
		}

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
					fmt.Fprintf(fow, "ftp.Open Error: %v\n", err)
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
	time.Sleep(300 * time.Millisecond) // nolint:gomnd

	// exit messages
	fmt.Println("all pull exit.")
}

// pullPath pulls the file or directory from the host to local.
// nolint:funlen
func (cp *Scp) pullPath(client *Connect) {
	ftp := client.Connect

	// get output writer
	client.Output.Create(client.Server)
	ow := client.Output.NewWriter()

	// basedir
	baseDir := common.ExpandHomeDir(cp.To.Path[0])

	// if multi pull, servername add baseDir
	if len(cp.From.Server) > 1 {
		baseDir = filepath.Join(baseDir, client.Server)
		_ = os.MkdirAll(baseDir, 0755)
	}

	baseDir, _ = filepath.Abs(baseDir)

	// walk remote path
	for _, path := range cp.From.Path {
		if _, err := ftp.Stat(path); err != nil {
			fmt.Fprintf(ow, "ftp.Stat path %s Error: %v\n", path, err)
			continue
		}

		if p, err := ftp.ReadLink(path); err == nil {
			fmt.Fprintf(ow, "read link to %s\n", p)
			path = filepath.Join(filepath.Dir(path), p)
		}

		globpath, err := ftp.Glob(path)
		if err != nil {
			fmt.Fprintf(ow, "ftp.Glob path %s Error: %v\n", path, err)
			continue
		}

		for _, gp := range globpath {
			remoteBase := filepath.Dir(gp) // basedir
			walker := ftp.Walk(gp)

			for walker.Step() {
				if err := walker.Err(); err != nil {
					fmt.Fprintf(ow, "walker.Err Error: %v\n", err)
					continue
				}

				p := walker.Path()
				if common.IsHidden(path, p) {
					continue // ignore hidden files.
				}

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
		fmt.Fprintf(ow, "ftp.Open Error: %v\n", err)
		return
	}

	defer rf.Close()

	// open local file
	lf, err := os.OpenFile(lpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(ow, "os.OpenFile Error: %v\n", err)
		return
	}

	defer lf.Close()

	rd := io.TeeReader(rf, lf)

	cp.ProgressWG.Add(1)
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
				fmt.Fprintf(os.Stderr, "cp.Run.CreateSSHConnect %s connect error: %v\n", server, err)
				return
			}

			// create sftp client
			ftp, err := sftp.NewClient(conn.Client)
			if err != nil {
				fmt.Fprintf(os.Stderr, "sftp.NewClient %s create client error: %v\n", server, err)
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
