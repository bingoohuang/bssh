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

	"github.com/bingoohuang/bssh/output"
	sshlib "github.com/blacknon/go-sshlib"
)

const cmdOPROMPT = "${SERVER} :: "

// cmd is run command.
func (r *Run) cmd() {
	command := strings.Join(r.ExecCmd, " ")
	connMap := map[string]*sshlib.Connect{}

	// make channel
	finished := make(chan bool)
	exitInput := make(chan bool)

	// print header
	r.PrintSelectServer()
	r.printRunCommand()

	if len(r.ServerList) == 1 {
		r.printProxy(r.ServerList[0])
	}

	// Create sshlib.Connect to connMap
	for _, server := range r.ServerList {
		// check count AuthMethod
		if len(r.serverAuthMethodMap[server]) == 0 {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %s is No AuthMethod.\n", server)
			continue
		}

		// Create sshlib.Connect
		conn, err := r.CreateSSHConnect(server)
		if err != nil {
			log.Printf("Error: %s:%s\n", server, err)
			continue
		}

		connMap[server] = conn
	}

	// Run command and print loop
	var writers []io.WriteCloser

	for s, c := range connMap {
		c.Session, _ = c.CreateSession()

		config := r.Conf.Server[s]

		o := &output.Output{
			Templete:      cmdOPROMPT,
			Count:         0,
			ServerList:    r.ServerList,
			Conf:          r.Conf.Server[s],
			EnableHeader:  r.EnableHeader,
			DisableHeader: r.DisableHeader,
			AutoColor:     true,
		}
		o.Create(s)

		c.Stdout = o.NewWriter()
		c.Stderr = o.NewWriter()

		// if single server, setup port forwarding.
		if len(r.ServerList) == 1 {
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

				go c.TCPDynamicForward("localhost", config.DynamicPortForward)
			}

			// if tty
			if r.IsTerm {
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
			}
		} else if r.IsParallel {
			w, _ := c.Session.StdinPipe()
			writers = append(writers, w)
		}
	}

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
		conn := c

		if r.IsParallel {
			go func() {
				_ = conn.Command(command)
				finished <- true
			}()
		} else {
			if len(stdinData) > 0 {
				// get stdin
				rd := bytes.NewReader(stdinData)
				w, _ := conn.Session.StdinPipe()

				// run command
				go func() {
					_ = conn.Command(command)
					finished <- true
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
	}

	// wait
	for i := 0; i < len(connMap); i++ {
		<-finished
	}

	close(exitInput)

	// sleep
	time.Sleep(300 * time.Millisecond)
}
