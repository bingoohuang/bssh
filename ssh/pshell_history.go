// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bingoohuang/bssh/output"
	"github.com/bingoohuang/ngg/ss"
)

type pShellHistory struct {
	Timestamp string
	Command   string
	Result    string
	Output    *output.Output
}

func (ps *pShell) NewHistoryWriter(server string, output *output.Output, m sync.Locker) *io.PipeWriter {
	// craete pShellHistory struct
	psh := &pShellHistory{
		Command:   ps.latestCommand,
		Timestamp: time.Now().Format("2006/01/02_15:04:05 "), // "yyyy/mm/dd_HH:MM:SS "
		Output:    output,
	}

	// create io.PipeReader, io.PipeWriter
	r, w := io.Pipe()

	// output Struct
	go ps.pShellHistoryPrint(psh, server, r, m)

	// return io.PipeWriter
	return w
}

func (ps *pShell) pShellHistoryPrint(psh *pShellHistory, server string, r io.Reader, m sync.Locker) {
	count := ps.Count

	var result string

	sc := bufio.NewScanner(r)

	for {
		for sc.Scan() {
			text := sc.Text()
			result = result + text + "\n"
		}

		if errors.Is(sc.Err(), io.ErrClosedPipe) {
			break
		}

		<-time.After(50 * time.Millisecond)
	}

	// Add Result
	psh.Result = result

	// Add History
	m.Lock()
	ps.History[count][server] = psh
	m.Unlock()
}

// GetHistoryFromFile return []History from historyfile.
func (ps *pShell) GetHistoryFromFile() (data []pShellHistory, err error) {
	histfile := ss.ExpandHome(ps.HistoryFile)

	// Open history file
	file, err := os.OpenFile(histfile, os.O_RDONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		line := sc.Text()
		text := strings.SplitN(line, " ", 2)

		if len(text) < 2 {
			continue
		}

		d := pShellHistory{
			Timestamp: text[0],
			Command:   text[1],
			Result:    "",
		}

		data = append(data, d)
	}

	return data, err
}

// PutHistoryFile put history text to s.HistoryFile
// ex.) write history(history file format)
//
//	YYYY-mm-dd_HH:MM:SS command...
//	YYYY-mm-dd_HH:MM:SS command...
//	...
func (ps *pShell) PutHistoryFile(cmd string) (err error) {
	histfile := ss.ExpandHome(ps.HistoryFile)

	// Open history file
	file, err := os.OpenFile(histfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer file.Close()

	// Get Time
	timestamp := time.Now().Format("2006/01/02_15:04:05 ") // "yyyy/mm/dd_HH:MM:SS "

	fmt.Fprintln(file, timestamp+cmd)

	return
}
