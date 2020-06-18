// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package ssh

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/bssh/conf"

	"github.com/bingoohuang/bssh/output"
	sshlib "github.com/blacknon/go-sshlib"
)

const cmdOPROMPT = "${SERVER} :: "

// cmd is run command.
func (r *Run) cmd() {
	command := strings.Join(r.ExecCmd, " ")
	finished, exitInput := make(chan bool), make(chan bool)

	// print header
	r.PrintSelectServer()
	r.printRunCommand()

	if len(r.ServerList) == 1 {
		r.printProxy(r.ServerList[0])
	}

	connMap := r.createConnMap()

	writers := r.createWriter(connMap)

	// if parallel flag true, and select server is not single,
	// set send stdin.
	var stdinData []byte

	switch {
	case r.IsParallel && len(r.ServerList) > 1:
		if r.isStdinPipe {
			go output.PushPipeWriter(exitInput, writers, os.Stdin)
		} else {
			go output.PushInput(exitInput, writers)
		}
	case !r.IsParallel && len(r.ServerList) > 1:
		if r.isStdinPipe {
			stdinData, _ = ioutil.ReadAll(os.Stdin)
		}
	}

	// run command
	for _, c := range connMap {
		r.runCommand(c, finished, command, stdinData)
	}

	// wait
	for range connMap {
		<-finished
	}

	close(exitInput)

	time.Sleep(300 * time.Millisecond) // nolint:gomnd
}

func (r *Run) createWriter(connMap map[string]*sshlib.Connect) []io.WriteCloser {
	// Run command and print loop
	var writers []io.WriteCloser

	for s, c := range connMap {
		c.Session, _ = c.CreateSession()

		config := r.Conf.Server[s]

		o := &output.Output{
			Templete: cmdOPROMPT, Count: 0, AutoColor: true,
			ServerList: r.ServerList, Conf: r.Conf.Server[s],
			EnableHeader: r.EnableHeader, DisableHeader: r.DisableHeader,
		}
		o.Create(s)

		c.Stdout, c.Stderr = o.NewWriter(), o.NewWriter()

		// if single server, setup port forwarding.
		if len(r.ServerList) == 1 {
			r.setupPortForwarding(&config, c)
		} else if r.IsParallel {
			w, _ := c.Session.StdinPipe()
			writers = append(writers, w)
		}
	}

	return writers
}

func (r *Run) createConnMap() map[string]*sshlib.Connect {
	connMap := map[string]*sshlib.Connect{}

	// Create sshlib.Connect to connMap
	for _, server := range r.ServerList {
		// check count AuthMethod
		if len(r.serverAuthMethodMap[server]) == 0 {
			fmt.Fprintf(os.Stderr, "Error: %s is No AuthMethod.\n", server)
			continue
		}

		conn, err := r.CreateSSHConnect(server)
		if err != nil {
			log.Printf("Error: %s:%s\n", server, err)
			continue
		}

		connMap[server] = conn
	}

	return connMap
}

func (r *Run) runCommand(conn *sshlib.Connect, finished chan bool, command string, stdinData []byte) {
	if r.IsParallel {
		go func() {
			defer func() { finished <- true }()

			_ = conn.Command(command)
		}()

		return
	}

	if len(stdinData) > 0 {
		// get stdin
		rd := bytes.NewReader(stdinData)
		w, _ := conn.Session.StdinPipe()

		// run command
		go func() {
			defer func() { finished <- true }()

			_ = conn.Command(command)
		}()

		// send stdin
		_, _ = io.Copy(w, rd)
		_ = w.Close()
	} else {
		// run command
		_ = conn.Command(command)
		go func() { finished <- true }()
	}
}

func (r *Run) setupPortForwarding(config *conf.ServerConfig, c *sshlib.Connect) {
	// OverWrite port forward mode
	if r.PortForwardMode != "" {
		config.PortForwardMode = r.PortForwardMode
	}

	// Overwrite port forward address
	if r.PortForwardLocal != "" && r.PortForwardRemote != "" {
		config.PortForwardLocal = r.PortForwardLocal
		config.PortForwardRemote = r.PortForwardRemote
	}

	// print header
	r.printPortForward(config.PortForwardMode, config.PortForwardLocal, config.PortForwardRemote)

	// Port Forwarding
	switch config.PortForwardMode {
	case "L", "":
		_ = c.TCPLocalForward(config.PortForwardLocal, config.PortForwardRemote)
	case "R":
		_ = c.TCPRemoteForward(config.PortForwardLocal, config.PortForwardRemote)
	}

	// Dynamic Port Forwarding
	if config.DynamicPortForward != "" {
		r.printDynamicPortForward(config.DynamicPortForward)

		go func() { _ = c.TCPDynamicForward("localhost", config.DynamicPortForward) }()
	}

	// if tty
	if r.IsTerm {
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}
}
