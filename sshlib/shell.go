// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sshlib

import (
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/term"

	"go.uber.org/atomic"

	"golang.org/x/crypto/ssh"
)

// ShellInitial connect login shell over ssh.
func (c *Connect) ShellInitial(session *ssh.Session, initialInput [][]byte, webPort int) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, state)

	// setup
	pipeToStdin, err := c.setupShell(session, webPort)
	if err != nil {
		return err
	}

	// Start shell
	if err := session.Shell(); err != nil {
		return err
	}

	// keep alive packet
	go c.SendKeepAlive(session)
	for _, initialCmd := range initialInput {
		time.Sleep(100 * time.Millisecond)
		_, _ = pipeToStdin.Write(initialCmd)
	}

	return session.Wait()
}

// Shell connects login shell over ssh.
func (c *Connect) Shell(session *ssh.Session) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return
	}
	defer term.Restore(fd, state)

	// setup
	if _, err := c.setupShell(session, 0); err != nil {
		return err
	}

	// Start shell
	if err := session.Shell(); err != nil {
		return err
	}

	// keep alive packet
	go c.SendKeepAlive(session)

	return session.Wait()
}

// CmdShell connect command shell over ssh.
// Used to start a shell with a specified command.
func (c *Connect) CmdShell(session *ssh.Session, command string) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return
	}
	defer term.Restore(fd, state)

	// setup
	if _, err := c.setupShell(session, 0); err != nil {
		return err
	}

	// Start shell
	if err := session.Start(command); err != nil {
		return err
	}

	// keep alive packet
	go c.SendKeepAlive(session)

	return session.Wait()
}

func (c *Connect) setupShell(session *ssh.Session, webPort int) (pipeToStdin *io.PipeWriter, err error) {
	session.Stdin, session.Stdout, pipeToStdin = c.interruptInput(webPort)
	session.Stderr = os.Stderr

	if c.logging {
		err = c.logger(session)
		if err != nil {
			log.Println(err)
		}
	}
	err = nil

	// Request tty
	if err := RequestTty(session); err != nil {
		return nil, err
	}

	// x11 forwarding
	if c.ForwardX11 {
		if err := c.X11Forward(session); err != nil {
			log.Println(err)
		}
	}

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
	session.Stdout = withLogWriters(session.Stdout, l)
	session.Stderr = withLogWriters(session.Stderr, l)
	return nil
}

func withLogWriters(writers ...io.Writer) io.Writer {
	allWriters := make([]io.Writer, 0, len(writers))
	for _, w := range writers {
		if mw, ok := w.(*logWriters); ok {
			allWriters = append(allWriters, mw.writers...)
		} else {
			allWriters = append(allWriters, w)
		}
	}
	return &logWriters{allWriters}
}

type logWriters struct {
	writers []io.Writer
}

func (t *logWriters) Write(p []byte) (n int, err error) {
	for i, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if i == 0 && n != len(p) { // only check the first writer response
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}
