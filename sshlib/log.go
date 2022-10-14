package sshlib

import (
	"bytes"
	"os"
	"time"

	"github.com/lunixbochs/vtclean"
	"go.uber.org/atomic"
)

type logWriter struct {
	logfile       *os.File
	logTimestamp  bool
	toggleLogging *atomic.Bool
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	if !l.toggleLogging.Load() {
		return len(p), nil
	}

	if !l.logTimestamp {
		return l.write(p)
	}

	pos := bytes.IndexByte(p, '\n')
	if pos < 0 {
		return l.write(p)
	}

	_, _ = l.write(p[:pos+1])
	timestamp := time.Now().Format("2006/01/02 15:04:05 ") // yyyy/mm/dd HH:MM:SS
	_, _ = l.logfile.Write([]byte(timestamp))
	if pos+1 < len(p) {
		_, _ = l.write(p[pos+1:])
	}

	return len(p), nil
}

func (l *logWriter) write(p []byte) (n int, err error) {
	c := vtclean.Clean(string(p), false)
	return l.logfile.WriteString(c)
}
