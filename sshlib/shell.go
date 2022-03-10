// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sshlib

import (
	"io"
	"log"
	"os"
	"time"

	"go.uber.org/atomic"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// ShellInitial connect login shell over ssh.
func (c *Connect) ShellInitial(session *ssh.Session, initialInput [][]byte, webPort int) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return
	}
	defer terminal.Restore(fd, state)

	var w io.WriteCloser
	if len(initialInput) > 0 {
		w, _ = session.StdinPipe()
		go func() {
			_, _ = io.Copy(w, os.Stdin)
		}()
	}

	// setup
	err = c.setupShell(session, webPort)
	if err != nil {
		return
	}

	// Start shell
	err = session.Shell()
	if err != nil {
		return
	}

	// keep alive packet
	go c.SendKeepAlive(session)

	if w != nil {
		for _, initialCmd := range initialInput {
			time.Sleep(100 * time.Millisecond)
			w.Write(initialCmd)
		}
	}

	err = session.Wait()
	if err != nil {
		return
	}

	return
}

// Shell connect login shell over ssh.
func (c *Connect) Shell(session *ssh.Session) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return
	}
	defer terminal.Restore(fd, state)

	// setup
	err = c.setupShell(session, 0)
	if err != nil {
		return
	}

	// Start shell
	err = session.Shell()
	if err != nil {
		return
	}

	// keep alive packet
	go c.SendKeepAlive(session)

	err = session.Wait()
	if err != nil {
		return
	}

	return
}

// Shell connect command shell over ssh.
// Used to start a shell with a specified command.
func (c *Connect) CmdShell(session *ssh.Session, command string) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return
	}
	defer terminal.Restore(fd, state)

	// setup
	err = c.setupShell(session, 0)
	if err != nil {
		return
	}

	// Start shell
	err = session.Start(command)
	if err != nil {
		return
	}

	// keep alive packet
	go c.SendKeepAlive(session)

	err = session.Wait()
	if err != nil {
		return
	}

	return
}

func (c *Connect) setupShell(session *ssh.Session, webPort int) (err error) {
	// set FD
	session.Stderr = os.Stderr

	if webPort > 0 {
		session.Stdin, session.Stdout = c.interruptInput(webPort)
	} else {
		session.Stdin = os.Stdin
		session.Stdout = os.Stdout
	}

	// Logging
	if c.logging {
		err = c.logger(session)
		if err != nil {
			log.Println(err)
		}
	}
	err = nil

	// Request tty
	if err := RequestTty(session); err != nil {
		return err
	}

	// x11 forwarding
	if c.ForwardX11 {
		err = c.X11Forward(session)
		if err != nil {
			log.Println(err)
		}
	}
	err = nil

	// ssh agent forwarding
	if c.ForwardAgent {
		c.ForwardSshAgent(session)
	}

	return
}

// SetLog set up terminal log logging.
// This only happens in Connect.Shell().
func (c *Connect) SetLog(path string, timestamp bool) {
	c.logging = true
	c.logFile = path
	c.logTimestamp = timestamp
	c.toggleLogging = atomic.NewBool(true)
}

// ToggleLogging set up terminal log logging.
// This only happens in Connect.Shell().
func (c *Connect) ToggleLogging(toggle bool) {
	c.toggleLogging.Store(toggle)
}

// logger is logging terminal log to c.logFile
func (c *Connect) logger(session *ssh.Session) (err error) {
	logfile, err := os.OpenFile(c.logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return
	}

	l := &logWriter{logfile: logfile, logTimestamp: c.logTimestamp, toggleLogging: c.toggleLogging}
	session.Stdout = io.MultiWriter(session.Stdout, l)
	session.Stderr = io.MultiWriter(session.Stderr, l)
	return nil
}
