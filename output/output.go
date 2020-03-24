// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package output

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/conf"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

// Output struct. command execute and bssh-shell mode output data.
type Output struct {
	// Template variable value (in unimplemented).
	//     - ${COUNT}  ... Count value(int)
	//     - ${SERVER} ... Server Name
	//     - ${ADDR}   ... Address
	//     - ${USER}   ... User Name
	//     - ${PORT}   ... Port
	//     - ${DATE}   ... Date(YYYY/mm/dd)
	//     - ${YEAR}   ... Year(YYYY)
	//     - ${MONTH}  ... Month(mm)
	//     - ${DAY}    ... Day(dd)
	//     - ${TIME}   ... Time(HH:MM:SS)
	//     - ${HOUR}   ... Hour(HH)
	//     - ${MINUTE} ... Minute(MM)
	//     - ${SECOND} ... Second(SS)
	Templete string

	// prompt is Output prompt.
	Prompt string

	// target server name. ${SERVER}
	Server string

	// Count value. ${COUNT}
	Count int

	// Selected Server list
	ServerList []string

	// ServerConfig
	Conf conf.ServerConfig

	// Progress bar
	// TDXX(blacknon): プログレスバーを出力させるための項目を追加
	Progress   *mpb.Progress
	ProgressWG *sync.WaitGroup

	// Enable/Disable print header
	EnableHeader  bool
	DisableHeader bool

	// Auto Colorize flag
	// TDXX(blacknon): colormodeに応じて、パイプ経由だった場合は色分けしないなどの対応ができるように条件分岐する(v0.6.1)
	AutoColor bool
}

// Create template, set variable value.
func (o *Output) Create(server string) {
	// TDXX(blacknon): Replaceでの処理ではなく、Text templateを作ってそちらで処理させる(置換処理だと脆弱性がありそうなので)
	o.Server = server

	// get max length at server name
	length := common.GetMaxLength(o.ServerList)
	addL := length - len(server)

	// get color num
	n := common.GetOrderNumber(server, o.ServerList)
	colorServerName := OutColorStrings(n, server)

	// set templete
	p := o.Templete

	// server info
	p = strings.Replace(p, "${SERVER}", fmt.Sprintf("%-*s", len(colorServerName)+addL, colorServerName), -1)
	p = strings.Replace(p, "${ADDR}", o.Conf.Addr, -1)
	p = strings.Replace(p, "${USER}", o.Conf.User, -1)
	p = strings.Replace(p, "${PORT}", o.Conf.Port, -1)

	o.Prompt = p
}

// GetPrompt update variable value
func (o *Output) GetPrompt() (p string) {
	// replace variable value
	p = strings.Replace(o.Prompt, "${COUNT}", strconv.Itoa(o.Count), -1)
	return
}

// NewWriter return io.WriteCloser at Output printer.
func (o *Output) NewWriter() (writer *io.PipeWriter) {
	// create io.PipeReader, io.PipeWriter
	r, w := io.Pipe()

	// run output.Printer()
	go o.Printer(r)

	// return writer
	return w
}

// Printer output stdout from reader.
func (o *Output) Printer(reader io.Reader) {
	sc := bufio.NewScanner(reader)

	for {
		for sc.Scan() {
			text := sc.Text()

			if (len(o.ServerList) > 1 && !o.DisableHeader) || o.EnableHeader {
				oPrompt := o.GetPrompt()
				fmt.Printf("%s %s\n", oPrompt, text)
			} else {
				fmt.Printf("%s\n", text)
			}
		}

		if sc.Err() == io.ErrClosedPipe {
			break
		}

		<-time.After(50 * time.Millisecond) // nolint gomnd
	}
}

// ProgressPrinter ...
func (o *Output) ProgressPrinter(size int64, reader io.Reader, path string) {
	// print header
	oPrompt := ""
	name := decor.Name(oPrompt)

	if len(o.ServerList) > 1 { // nolint gomnd
		oPrompt = o.GetPrompt()
		name = decor.Name(oPrompt, decor.WC{W: len(path) + 1, C: decor.DSyncWidth}) // nolint gomnd
	}

	// trim space
	path = strings.TrimSpace(path)

	// set progress
	bar := o.Progress.AddBar(size,
		mpb.BarClearOnComplete(),
		mpb.PrependDecorators(name,
			decor.OnComplete(decor.Name(path, decor.WCSyncSpaceR), fmt.Sprintf("%s done!", path)),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WC{W: 5}), ""), // nolint gomnd
			decor.Elapsed(decor.ET_STYLE_HHMMSS, decor.WC{W: 10}),  // nolint gomnd
		),
	)

	sum := 0
	startTime := time.Now()

	// print out progress
	defer o.ProgressWG.Done()

	for {
		// read byte (1mb)
		b := make([]byte, 1048576)
		s, err := reader.Read(b)

		sum += s

		bar.IncrBy(s, time.Since(startTime))

		// check exit
		if err == io.EOF {
			bar.SetTotal(size, true)
			break
		}
	}
}

// OutColorStrings ...
func OutColorStrings(num int, inStrings string) (str string) {
	// 1=Red,2=Yellow,3=Blue,4=Magenta,0=Cyan
	color := 31 + num%5 // nolint gomnd

	str = fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, inStrings)

	return
}

// PushPipeWriter is PipeReader to []io.WriteCloser.
func PushPipeWriter(isExit <-chan bool, output []io.WriteCloser, input io.Reader) {
	rd := bufio.NewReader(input)
loop:
	for {
		buf := make([]byte, 1024)
		size, err := rd.Read(buf)

		if size > 0 {
			d := buf[:size]

			for _, w := range output {
				_, _ = w.Write(d)
			}
		}

		switch err {
		case nil:
			continue
		case io.ErrClosedPipe, io.EOF:
			break loop
		}

		select {
		case <-isExit:
			break loop
		case <-time.After(10 * time.Millisecond): // nolint gomnd
			continue
		}
	}

	for _, w := range output {
		_ = w.Close()
	}
}

// PushInput sends input to ssh Session Stdin
func PushInput(isExit <-chan bool, output []io.WriteCloser) {
	rd := bufio.NewReader(os.Stdin)
loop:
	for {
		data, _ := rd.ReadBytes('\n')
		if len(data) > 0 {
			for _, w := range output {
				_, _ = w.Write(data)
			}
		}

		select {
		case <-isExit:
			break loop
		case <-time.After(10 * time.Millisecond): // nolint gomnd
			continue
		}
	}

	for _, w := range output {
		_ = w.Close()
	}
}
