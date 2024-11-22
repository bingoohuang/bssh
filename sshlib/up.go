package sshlib

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bingoohuang/ngg/tsid"
	"github.com/cheggaaa/pb/v3"
)

func (i *interruptReader) up(file string) {
	stat, err := os.Stat(file)
	if err != nil {
		log.Printf("stat error: %v", err)
		return
	}

	f, err := os.Open(file)
	if err != nil {
		log.Printf("open error: %v", err)
		return
	}
	defer f.Close()

	prefix := fmt.Sprintf("/tmp/%s.%s", tsid.Fast().ToString(), filepath.Base(file))
	os.Stdout.Write([]byte(fmt.Sprintf("start to upload local %s to remote %s\n",
		file, prefix)))

	// create bar
	bar := pb.New(int(stat.Size()))
	// refresh info every second (default 200ms)
	bar.SetRefreshRate(time.Second)
	// force set io.Writer, by default it's os.Stderr
	bar.SetWriter(os.Stdout)
	// bar will format numbers as bytes (B, KiB, MiB, etc)
	bar.Set(pb.Bytes, true)
	bar.Start()

	bs := make([]byte, 20480)
	count := 0
	for idx := 1; ; idx++ {
		n, err := f.Read(bs)
		bs = bs[:n]
		if err != nil && errors.Is(err, io.EOF) {
			break
		}
		bar.Add(n)

		count++
		tmpfile := fmt.Sprintf("%s.%d", prefix, idx)
		content := base64.StdEncoding.EncodeToString(bs)
		t := tsid.Fast().ToString()
		cmd := fmt.Sprintf("echo open:%s; echo %s | base64 -d > %s ; md5sum %s; echo close:%s\r",
			t, content, tmpfile, tmpfile, t)
		i.directWriter.Write([]byte(cmd))
		i.notifyC <- NotifyCmd{Type: NotifyTypeTag, Value: t}
		localMd5 := Md5Hash(bs)
		rsp := <-i.notifyRspC
		if field0(rsp) != localMd5 {
			bar.Finish()
			log.Printf("write failed")
			return
		}
	}

	bar.Finish()

	t := tsid.Fast().ToString()
	cmd := fmt.Sprintf("echo open:%s; cat %s.{1..%d} > %s; rm -fr %s.{1..%d}; echo close:%s\r",
		t, prefix, count, prefix, prefix, count, t)
	i.directWriter.Write([]byte(cmd))
	i.notifyC <- NotifyCmd{Type: NotifyTypeTag, Value: t}
	<-i.notifyRspC
}

func Md5Hash(raw []byte) string {
	m := md5.Sum(raw)
	return hex.EncodeToString(m[:])
}
