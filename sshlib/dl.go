package sshlib

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bingoohuang/ngg/ss"
	"github.com/bingoohuang/ngg/tsid"
	"github.com/cheggaaa/pb/v3"
)

func (i *interruptReader) dl(file string) {
	fileSize, err := i.lsSize(file)
	if err != nil {
		log.Printf("ls error: %v", err)
		return
	}

	md5sum := i.md5sum(file)

	base := filepath.Base(file)
	tempFile, err := os.CreateTemp("/tmp", "*."+base)
	if err != nil {
		log.Printf("create temp file: %v", err)
		return
	}
	defer tempFile.Close()

	os.Stdout.Write([]byte(fmt.Sprintf("start to download remote %s to local %s\n",
		file, tempFile.Name())))

	// create bar
	bar := pb.New(int(fileSize))
	// refresh info every second (default 200ms)
	bar.SetRefreshRate(time.Second)
	// force set io.Writer, by default it's os.Stderr
	bar.SetWriter(os.Stdout)
	// bar will format numbers as bytes (B, KiB, MiB, etc)
	bar.Set(pb.Bytes, true)
	bar.Start()

	pr, pw := io.Pipe()
	decoder := base64.NewDecoder(base64.StdEncoding, pr)

	h := md5.New()
	br := &PbReader{Reader: decoder, bar: bar}

	go func() {
		if _, err := io.Copy(io.MultiWriter(tempFile, h), br); err != nil && errors.Is(err, io.EOF) {
			log.Printf("copy file error: %v", err)
		}
	}()

	for skip := 0; ; skip++ {
		if i.dlPart(file, skip, pw) {
			break
		}
	}

	bar.Finish()

	dlMd5 := fmt.Sprintf("%x", h.Sum(nil))
	if dlMd5 != md5sum {
		os.Stdout.Write([]byte("downloaded failed"))
	}
}

// PbReader counts the bytes read through it.
type PbReader struct {
	io.Reader
	bar *pb.ProgressBar
}

func (r *PbReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if n > 0 {
		r.bar.Add(n)
	}

	return
}

func (i *interruptReader) dlPart(file string, skip int, pw *io.PipeWriter) bool {
	rsp, _ := i.executeCmd(fmt.Sprintf(
		"dd if=%s bs=%d count=1 skip=%d 2>/dev/null | base64 -w 0 && echo", file, 102400, skip), 3*time.Second)
	ok := rsp == ""
	if ok {
		pw.Close()
	} else {
		pw.Write([]byte(rsp))
	}

	return ok
}

func (i *interruptReader) md5sum(file string) string {
	rsp, _ := i.executeCmd(fmt.Sprintf("md5sum %s", file), 3*time.Second)
	return field0(rsp)
}

func (i *interruptReader) lsSize(file string) (size int64, err error) {
	rsp, _ := i.executeCmd(fmt.Sprintf("ls -l %s 2>&1", file), 3*time.Second)
	f := strings.Fields(rsp)
	if len(f) >= 4 {
		size, _ = ss.Parse[int64](f[4])
	}

	if size > 0 {
		return size, nil
	}

	return 0, errors.New(rsp)
}

func (i *interruptReader) executeCmd(cmd string, timeout time.Duration) (string, error) {
	tag := tsid.Fast().ToString()
	i.directWriter.Write([]byte(fmt.Sprintf("echo open:%s; %s; echo close:%s\r", tag, cmd, tag)))
	i.notifyC <- NotifyCmd{
		Type:  NotifyTypeTag,
		Value: tag,
	}
	select {
	case <-time.After(timeout):
		return "", errors.New("timeout")
	case rsp := <-i.notifyRspC:
		return rsp, nil
	}
}

func field0(s string) string {
	f := strings.Fields(s)
	if len(f) > 0 {
		return f[0]
	}

	return ""
}
