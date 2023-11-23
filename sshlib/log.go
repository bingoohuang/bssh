package sshlib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lunixbochs/vtclean"
	"go.uber.org/atomic"
)

type logWriter struct {
	logfile       *os.File
	logTimestamp  bool
	toggleLogging *atomic.Bool

	// keep ansi code on terminal log.
	logKeepAnsiCode bool

	buf bytes.Buffer
}

func NewLogWrite(logfile *os.File, toggleLogging *atomic.Bool, logTimestamp, logKeepAnsiCode bool) io.Writer {
	w := &logWriter{
		logfile:         logfile,
		logTimestamp:    logTimestamp,
		toggleLogging:   toggleLogging,
		logKeepAnsiCode: logKeepAnsiCode,
	}

	if !logKeepAnsiCode {
		go w.cleanLog()
	}

	return w
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	if !l.toggleLogging.Load() {
		return len(p), nil
	}

	if !l.logKeepAnsiCode || l.logTimestamp {
		return l.buf.Write(p)
	}

	return l.logfile.Write(p)
}

func (l *logWriter) cleanLog() {
	var preLine []byte
	for {
		if l.buf.Len() > 0 {
			// get line
			line, err := l.buf.ReadBytes('\n')

			if err == io.EOF {
				preLine = append(preLine, line...)
				continue
			} else {
				printLine := string(append(preLine, line...))

				if l.logTimestamp {
					timestamp := time.Now().Format("2006/01/02 15:04:05 ") // yyyy/mm/dd HH:MM:SS
					printLine = timestamp + printLine
				}

				// remove ansi code.
				if !l.logKeepAnsiCode {
					// NOTE:
					//     In vtclean.Clean, the beginning of the line is deleted for some reason.
					//     for that reason, one character add at line head.
					printLine = "." + printLine
					printLine = vtclean.Clean(printLine, false)
				}

				fmt.Fprintf(l.logfile, printLine)
				preLine = []byte{}
			}
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
}
