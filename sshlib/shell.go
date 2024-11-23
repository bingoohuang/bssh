// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sshlib

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"

	"go.uber.org/atomic"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func NewInitialPromptReadyChecker() *InitialPromptReadyChecker {
	return &InitialPromptReadyChecker{
		NotifyCh: make(chan struct{}),
	}
}

type InitialPromptReadyChecker struct {
	InitialPromptReady atomic.Bool
	NotifyCh           chan struct{}
}

func (t *InitialPromptReadyChecker) Wait(timeout time.Duration) bool {
	if t.InitialPromptReady.Load() {
		return true
	}
	select {
	case <-t.NotifyCh:
		return true
	case <-time.After(timeout):
		return t.InitialPromptReady.Load()
	}
}

func (t *InitialPromptReadyChecker) Read(p []byte) {
	if !t.InitialPromptReady.Load() && bytes.HasSuffix(p, []byte("# ")) || bytes.HasSuffix(p, []byte("$ ")) {
		t.InitialPromptReady.Store(true)

		select {
		case t.NotifyCh <- struct{}{}:
		default:
		}
	}
}

// ShellInitial connect login shell over ssh.
func (c *Connect) ShellInitial(session *ssh.Session, initialInput [][]byte,
	initialCmdSleep time.Duration, webPort int, hostInfoAutoEnabled bool,
	hostInfoScript string, hostInfoUpdater func(hostInfo string)) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, state)

	// setup
	checker := NewInitialPromptReadyChecker()
	pipeToStdin, ir, err := c.setupShell(session, webPort, hostInfoScript, checker.Read)
	if err != nil {
		return err
	}

	// Start shell
	if err := session.Shell(); err != nil {
		return err
	}

	// keep alive packet
	go c.SendKeepAlive(session)

	if initialCmdSleep == 0 {
		initialCmdSleep = 250 * time.Millisecond
	}
	checker.Wait(initialCmdSleep)

	if len(initialInput) > 0 {
		for i, initialCmd := range initialInput {
			if i > 0 {
				time.Sleep(initialCmdSleep)
			}
			if len(initialCmd) > 0 {
				_, _ = pipeToStdin.Write(initialCmd)
			}
		}
	}

	if hostInfoAutoEnabled {
		hostInfo, _ := ir.executeCmd(hostInfoScript, 15*time.Second)
		hostInfo = regexp.MustCompile(`[\r\n]+`).ReplaceAllString(hostInfo, "")
		if hostInfo != "" {
			fmt.Printf("主机信息: %s\n", hostInfo)
			hostInfoUpdater(hostInfo)
		}
		pipeToStdin.Write([]byte("\r"))
	}

	return session.Wait()
}

// Shell connects login shell over ssh.
func (c *Connect) Shell(session *ssh.Session) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, state)

	// setup
	if _, _, err := c.setupShell(session, 0, "", nil); err != nil {
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
	if _, _, err := c.setupShell(session, 0, "", nil); err != nil {
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

//type CatchWriter struct {
//	w io.Writer
//
//	lock     sync.RWMutex
//	catching bool
//	resp     string
//}
//
//func newCatchWriter(w io.Writer) *CatchWriter {
//	return &CatchWriter{
//		w: w,
//	}
//}
//
//func (t *CatchWriter) StartCatch() {
//	t.lock.Lock()
//	defer t.lock.Unlock()
//
//	t.resp = ""
//	t.catching = true
//}
//
//func (t *CatchWriter) StopCatch(startTag, endTag string) string {
//	for i := 0; i < 3; i++ {
//		time.Sleep(200 * time.Millisecond)
//		resp, ok := func() (string, bool) {
//			t.lock.Lock()
//			defer t.lock.Unlock()
//
//			if strings.Count(t.resp, startTag) < 2 || strings.Count(t.resp, endTag) < 2 {
//				return "", false
//			}
//
//			startPos := strings.LastIndex(t.resp, startTag)
//			endPos := strings.LastIndex(t.resp, endTag)
//			t.catching = false
//			return t.resp[startPos+len(startTag) : endPos], true
//		}()
//		if ok {
//			return resp
//		}
//	}
//
//	t.lock.Lock()
//	defer t.lock.Unlock()
//	t.catching = false
//
//	return ""
//}
//
//func (t *CatchWriter) Write(p []byte) (n int, err error) {
//	n, err = t.w.Write(p)
//
//	t.lock.RLock()
//	if t.catching {
//		t.resp += string(p[:n])
//	}
//	t.lock.RUnlock()
//
//	return
//}

func (c *Connect) setupShell(session *ssh.Session, webPort int, hostInfoScript string, shellReader func(p []byte)) (
	pipeToStdin *io.PipeWriter, ir *interruptReader, err error) {
	session.Stdin, session.Stdout, pipeToStdin, ir = c.interruptInput(webPort, hostInfoScript, shellReader)
	session.Stderr = os.Stderr

	if c.logging {
		if err := c.logger(session); err != nil {
			log.Println(err)
		}
	}
	// Request tty
	if err := RequestTty(session); err != nil {
		return nil, nil, err
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

	l := NewLogWrite(logfile, c.toggleLogging, c.logTimestamp, c.LogKeepAnsiCode)
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
