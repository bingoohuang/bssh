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
		return l.logfile.Write(p)
	}

	pos := bytes.IndexByte(p, '\n')
	if pos < 0 {
		cp := vtclean.Clean(string(p), false)
		return l.logfile.WriteString(cp)
	}

	cp := vtclean.Clean(string(p[:pos+1]), false)
	_, _ = l.logfile.WriteString(cp)
	timestamp := time.Now().Format("2006/01/02 15:04:05 ") // yyyy/mm/dd HH:MM:SS
	_, _ = l.logfile.Write([]byte(timestamp))
	if pos+1 < len(p) {
		cp = vtclean.Clean(string(p[pos+1:]), false)
		_, _ = l.logfile.WriteString(cp)
	}

	return len(p), nil
}
