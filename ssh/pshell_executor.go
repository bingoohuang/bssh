// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package ssh

import (
	"fmt"
	"io"
	"strings"
)

// PipeSet is pipe in/out set struct.
type PipeSet struct {
	in  *io.PipeReader
	out *io.PipeWriter
}

// Executor run ssh command in parallel-shell.
func (ps *pShell) Executor(command string) {
	// trim space
	command = strings.TrimSpace(command)

	// parse command
	pslice, _ := parsePipeLine(command)
	if len(pslice) == 0 {
		return
	}

	// set latest command
	ps.latestCommand = command

	// register history
	_ = ps.PutHistoryFile(command)

	// exec pipeline
	ps.parseExecutor(pslice)
}

// parseExecutor assemble and execute the parsed command line.
// TDXX(blacknon): 現状はパイプにしか対応していないので、`&&`や`||`にも対応できるよう変更する(v0.6.1)
// nolint funlen
func (ps *pShell) parseExecutor(pslice [][]pipeLine) {
	// Create History
	ps.History[ps.Count] = map[string]*pShellHistory{}

	for _, pline := range pslice {
		// count pipe num
		pnum := countPipeSet(pline, "|")

		// create pipe set
		pipes := createPipeSet(pnum)

		// join pipe set
		pline = joinPipeLine(pline)

		// printout run command
		fmt.Printf("[Command:%s ]\n", joinPipeLineSlice(pline))

		// pipe counter
		var n int

		// create channel
		ch := make(chan bool)
		defer close(ch)

		kill := make(chan bool)
		defer close(kill)

		for i, p := range pline {
			// declare nextPipeLine
			var bp pipeLine

			// declare in,out
			var in *io.PipeReader

			var out *io.PipeWriter

			// get next pipe line
			if i > 0 {
				bp = pline[i-1]
			}

			// set stdin
			// If the before delimiter is a pipe, set the stdin before io.PipeReader.
			if bp.Operator == "|" {
				in = pipes[n-1].in
			}

			// set stdout
			// If the delimiter is a pipe, set the stdout output a io.PipeWriter.
			if p.Operator == "|" {
				out = pipes[n].out

				// add pipe num
				n++
			}

			// exec pipeline
			go ps.run(p, in, out, ch, kill)
		}

		// get and send kill
		killExit := make(chan bool)
		defer close(killExit)

		go func() {
			select {
			case <-ps.Signal:
				for i := 0; i < len(pline); i++ {
					kill <- true
				}
			case <-killExit:
				return
			}
		}()

		// wait channel
		wait(len(pline), ch)
	}

	// add ps.Count
	// (Does not count if only the built-in command is executed)
	isBuildInOnly := true

	for _, pline := range pslice {
		if len(pline) > 1 { // nolint gomnd
			isBuildInOnly = false
			break
		}

		if !checkBuildInCommand(pline[0].Args[0]) {
			isBuildInOnly = false
			break
		}
	}

	if !isBuildInOnly {
		ps.Count++
	}
}

// countPipeSet count delimiter in pslice.
func countPipeSet(pline []pipeLine, del string) (count int) {
	for _, p := range pline {
		if p.Operator == del {
			count++
		}
	}

	return count
}

// createPipeSet return Returns []*PipeSet used by the process.
func createPipeSet(count int) (pipes []*PipeSet) {
	pipes = make([]*PipeSet, count)

	for i := 0; i < count; i++ {
		in, out := io.Pipe()
		pipes[i] = &PipeSet{in: in, out: out}
	}

	return pipes
}
