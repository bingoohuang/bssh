package sshlib

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bingoohuang/filestash"
	"github.com/bingoohuang/gossh/pkg/gossh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func (c *Connect) interruptInput(webPort int) (*io.PipeReader, *io.PipeWriter) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	notifyC := make(chan NotifyCmd)
	notifyRspC := make(chan string)
	go func() {
		r := newInterruptWriter(r2, notifyC, notifyRspC)
		if _, err := io.Copy(os.Stdout, r); err != nil && errors.Is(err, io.EOF) {
			return
		}
	}()

	go func() {
		r := newInterruptReader(webPort, notifyC, notifyRspC, w1, c)
		if _, err := io.Copy(w1, r); err != nil && errors.Is(err, io.EOF) {
			return
		}
	}()

	return r1, w2
}

func newInterruptReader(port int, notifyC chan NotifyCmd, notifyRspC chan string, directWriter *io.PipeWriter, connect *Connect) *interruptReader {
	return &interruptReader{
		r:            os.Stdin,
		port:         port,
		directWriter: directWriter,
		notifyC:      notifyC,
		notifyRspC:   notifyRspC,
		connect:      connect,
	}
}

type interruptWriter struct {
	r          io.Reader
	notifyC    chan NotifyCmd
	result     chan string
	notifyTag  string
	buf        bytes.Buffer
	notifyRspC chan string
}

func (i *interruptWriter) Read(p []byte) (n int, err error) {
	n, err = i.r.Read(p)
	if n == 0 {
		return 0, err
	}

	if i.notifyTag != "" {
		i.buf.Write(p[:n])
		if bytes.Contains(i.buf.Bytes(), []byte("close:"+i.notifyTag+"\r\n")) {
			rsp, closeFound := clearTag(i.notifyTag, i.buf.Bytes())
			if closeFound {
				i.notifyRspC <- rsp
				i.buf.Reset()
				i.notifyTag = ""
			}
		}
		return 0, err
	}

	select {
	case notify := <-i.notifyC:
		i.notifyTag = notify.Value
		i.buf.Reset()
		i.buf.Write(p[:n])
		return 0, err
	default:
	}

	return n, err
}

func clearTag(tag string, b []byte) (string, bool) {
	openTag := []byte("open:" + tag + "\r\n")
	openPos := bytes.Index(b, openTag)
	if openPos < 0 {
		return "", false
	}

	closeTag := []byte("close:" + tag)
	closePos := bytes.Index(b[openPos:], closeTag)
	if closePos < 0 {
		return "", false
	}

	s := string(b[openPos+len(openTag) : openPos+closePos])
	return strings.TrimSpace(s), true
}

func newInterruptWriter(r io.Reader, notifyC chan NotifyCmd, notifyRspC chan string) io.Reader {
	return &interruptWriter{
		r:          r,
		notifyC:    notifyC,
		notifyRspC: notifyRspC,
	}
}

type NotifyType int

const (
	NotifyTypeTag NotifyType = iota
)

type NotifyCmd struct {
	Type  NotifyType
	Value string
}

type interruptReader struct {
	r            io.Reader
	port         int
	buf          bytes.Buffer
	directWriter *io.PipeWriter
	notifyC      chan NotifyCmd
	notifyRspC   chan string
	connect      *Connect

	LastKeyCtrK     bool
	LastKeyCtrKTime time.Time
}

func (i *interruptReader) Read(p []byte) (n int, err error) {
	if GetEnvSshEnv() == 1 {
		n, err = i.r.Read(p)
		if n == 0 {
			return 0, err
		}

		isKeyCtrK := n == 1 && p[0] == gossh.KeyCtrlK
		now := time.Now()
		defer func() {
			i.LastKeyCtrK = isKeyCtrK
			i.LastKeyCtrKTime = now
		}()
		if !isKeyCtrK || !i.LastKeyCtrK || now.Sub(i.LastKeyCtrKTime) > time.Second {
			return n, nil
		}
		os.Stdout.Write([]byte(">> "))
	}

Next:
	screen := struct {
		io.Reader
		io.Writer
	}{Reader: os.Stdin, Writer: os.Stdout}
	term := terminal.NewTerminal(screen, "")
	line, err := term.ReadLine()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Printf("read line, error: %v", err)
		}
		i.directWriter.Write([]byte("\r"))
		return 0, err
	}

	cmdFields := strings.Fields(line)
	i.connect.ToggleLogging(false)
	defer i.connect.ToggleLogging(true)

	if len(cmdFields) == 1 && strings.EqualFold(cmdFields[0], "%dash") {
		go filestash.OpenBrowser(fmt.Sprintf("http://127.0.0.1:%d/dash", i.port))
	} else if len(cmdFields) == 1 && strings.EqualFold(cmdFields[0], "%web") {
		go filestash.OpenBrowser(fmt.Sprintf("http://127.0.0.1:%d", i.port))
	} else if len(cmdFields) == 2 && strings.EqualFold(cmdFields[0], "%up") {
		i.up(cmdFields[1])
	} else if len(cmdFields) == 2 && strings.EqualFold(cmdFields[0], "%dl") {
		i.dl(cmdFields[1])

		// 参考 https://github.com/M09Ic/rscp
		// 		if opt.upload blockSize = 20480
		//		if opt.download  blockSize = 102400
		// 下载 cmd := fmt.Sprintf("dd if=%s bs=%d count=1 skip=%d 2>/dev/null | base64 -w 0 && echo", remotefile, blockSize, off)
		// 上传 cmd := fmt.Sprintf("echo %s | base64 -d > %s && md5sum %s", content, tmpfile, tmpfile)
		// 合并文件: cd %s && cat %s > %s
	} else {
		i.directWriter.Write([]byte(line))
	}

	i.directWriter.Write([]byte("\r"))
	if GetEnvSshEnv() == 0 {
		goto Next
	}
	return 0, err
}
