// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package ssh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bingoohuang/bssh/misc"

	"github.com/bingoohuang/bssh/output"
	sshlib "github.com/blacknon/go-sshlib"
	"golang.org/x/crypto/ssh"
)

// TDXX(blacknon): 以下のBuild-in Commandを追加する
//     - %cd <PATH>         ... リモートのディレクトリを変更する(事前のチェックにsftpを使用か？)
//     - %lcd <PATH>        ... ローカルのディレクトリを変更する
//     - %save <num> <PATH> ... 指定したnumの履歴をPATHに記録する (v0.6.1)
//     - %set <args..>      ... 指定されたオプションを設定する(Optionsにて管理) (v0.6.1)
//     - %diff <num>        ... 指定されたnumの履歴をdiffする(multi diff)。できるかどうか要検討。 (v0.6.1以降)
//                              できれば、vimdiffのように横に差分表示させるようにしたいものだけど…？
//     - %get remote local  ... sftpプロトコルを利用して、ファイルやディレクトリを取得する (v0.6.1)
//     - %put local remote  ... sftpプロトコルを利用して、ファイルやディレクトリを配置する (v0.6.1)

// checkBuildInCommand return true if cmd is build-in command.
func checkBuildInCommand(cmd string) (isBuildInCmd bool) {
	// check build-in command
	switch cmd {
	case "exit", "quit", "clear": // build-in command
		isBuildInCmd = true

	case
		"%history",
		misc.PercentOut, "%outlist",
		"%save", "%set": // parsent build-in command.
		isBuildInCmd = true
	}

	return
}

// checkLocalCommand return bool, check is pshell build-in command or
// local machine command(%%command).
func checkLocalCommand(cmd string) (isLocalCmd bool) {
	// check local command regex
	regex := regexp.MustCompile(`^!.*`)

	// local command
	return regex.MatchString(cmd)
}

// checkLocalBuildInCommand check local or build-in command.
func checkLocalBuildInCommand(cmd string) (result bool) {
	result = checkBuildInCommand(cmd)
	if result {
		return result
	}

	// check local command
	result = checkLocalCommand(cmd)

	return result
}

// runBuildInCommand is run buildin or local machine command.
func (ps *pShell) run(pl pipeLine, in io.Reader, out *io.PipeWriter, ch chan<- bool, kill chan bool) {
	// get 1st element
	command := pl.Args[0]

	// check and exec build-in command
	switch command {
	// exit or quit
	case "exit", "quit":
		os.Exit(0)

	// clear
	case "clear":
		fmt.Printf("\033[H\033[2J")
		return

	// %history
	case "%history":
		ps.buildinHistory(out, ch)
		return

	// %outlist
	case "%outlist":
		ps.buildinOutlist(out, ch)
		return

	// %out [num]
	case misc.PercentOut:
		num := ps.Count - 1

		if len(pl.Args) > 1 {
			var err error

			num, err = strconv.Atoi(pl.Args[1])
			if err != nil {
				return
			}
		}

		ps.buildinOut(num, out, ch)

		return
	}

	// check and exec local command
	if regexp.MustCompile(`^!.*`).MatchString(command) {
		ps.executeLocalPipeLine(pl, in, out, ch, kill)
	} else {
		ps.executeRemotePipeLine(pl, in, out, ch, kill)
	}
}

// buildinHistory is printout history (shell history).
func (ps *pShell) buildinHistory(out *io.PipeWriter, ch chan<- bool) {
	stdout := setOutput(out)

	// read history file
	data, err := ps.GetHistoryFromFile()
	if err != nil {
		return
	}

	// print out history
	for _, h := range data {
		fmt.Fprintf(stdout, "%s: %s\n", h.Timestamp, h.Command)
	}

	// close out
	if _, ok := stdout.(*io.PipeWriter); ok {
		_ = out.CloseWithError(io.ErrClosedPipe)
	}

	// send exit
	ch <- true
}

// buildinOutlist is print exec history list.
func (ps *pShell) buildinOutlist(out *io.PipeWriter, ch chan<- bool) {
	stdout := setOutput(out)

	for i := 0; i < len(ps.History); i++ {
		h := ps.History[i]
		for _, hh := range h {
			fmt.Fprintf(stdout, "%3d : %s\n", i, hh.Command)
			break
		}
	}

	// close out
	if _, ok := stdout.(*io.PipeWriter); ok {
		_ = out.CloseWithError(io.ErrClosedPipe)
	}

	// send exit
	ch <- true
}

// buildinOut is print exec history at number
// example:
//     - %out
//     - %out <num>
func (ps *pShell) buildinOut(num int, out *io.PipeWriter, ch chan<- bool) {
	stdout := setOutput(out)
	histories := ps.History[num]

	i := 0
	for _, h := range histories {
		// if first, print out command
		if i == 0 {
			fmt.Fprintf(os.Stderr, "[History:%s ]\n", h.Command)
		}
		i++

		// print out result
		if len(histories) > 1 && stdout == os.Stdout && h.Output != nil {
			// set Output.Count
			bc := h.Output.Count
			h.Output.Count = num
			op := h.Output.GetPrompt()

			// TDXX(blacknon): Outputを利用させてOPROMPTを生成
			sc := bufio.NewScanner(strings.NewReader(h.Result))
			for sc.Scan() {
				_, _ = fmt.Fprintf(stdout, "%s %s\n", op, sc.Text())
			}

			// reset Output.Count
			h.Output.Count = bc
		} else {
			_, _ = fmt.Fprintf(stdout, h.Result)
		}
	}

	// close out
	if _, ok := stdout.(*io.PipeWriter); ok {
		_ = out.CloseWithError(io.ErrClosedPipe)
	}

	// send exit
	ch <- true
}

// executePipeLineRemote is exec command in remote machine.
// Didn't know how to send data from Writer to Channel, so switch the function if * io.PipeWriter is Nil.
// nolint:funlen
func (ps *pShell) executeRemotePipeLine(pline pipeLine, in io.Reader, out *io.PipeWriter,
	ch chan<- bool, kill chan bool) {
	// join command
	command := strings.Join(pline.Args, " ")

	// set stdin/stdout
	stdin := setInput(in)
	stdout := setOutput(out)

	// create channels
	exit := make(chan bool)
	exitInput := make(chan bool) // Input finish channel

	writers := make([]io.WriteCloser, len(ps.Connects))
	sessions := make([]*ssh.Session, len(ps.Connects))

	// create session and writers
	m := new(sync.Mutex)

	for i, c := range ps.Connects {
		s, err := c.CreateSession()
		if err != nil {
			continue
		}

		// Request tty (Only when input is os.Stdin and output is os.Stdout).
		if stdin == os.Stdin && stdout == os.Stdout {
			_ = sshlib.RequestTty(s)
		}

		// set stdout
		var ow io.Writer

		ow = stdout
		if ow == os.Stdout {
			// create Output Writer
			c.Output.Count = ps.Count
			w := c.Output.NewWriter()

			// create pShellHistory Writer
			hw := ps.NewHistoryWriter(c.Output.Server, c.Output, m)

			ow = io.MultiWriter(w, hw)
			_ = w.CloseWithError(io.ErrClosedPipe)
			_ = hw.CloseWithError(io.ErrClosedPipe)
		}

		s.Stdout = ow

		// get and append stdin writer
		w, _ := s.StdinPipe()

		writers[i] = w
		sessions[i] = s
	}

	// multi input-writer
	switch stdin.(type) {
	case *os.File:
		// push input to parallel session
		// (Only when input is os.Stdin and output is os.Stdout).
		if stdout == os.Stdout {
			go output.PushInput(exitInput, writers)
		}
	case *io.PipeReader:
		go output.PushPipeWriter(exitInput, writers, stdin)
	}

	// run command
	for _, s := range sessions {
		session := s

		go func() {
			_ = session.Run(command)
			//_ = session.Close()
			exit <- true
		}()
	}

	go func() {
		<-kill

		for _, s := range sessions {
			_ = s.Signal(ssh.SIGINT)
			_ = s.Close()
		}
	}()

	wait(len(sessions), exit)

	// wait time (0.500 sec)
	time.Sleep(5000 * time.Millisecond) // nolint:gomnd

	// Print message `Please input enter` (Only when input is os.Stdin and output is os.Stdout).
	// Note: This necessary for using Blocking.IO.
	if stdin == os.Stdin && stdout == os.Stdout {
		_, _ = fmt.Fprintf(os.Stderr, "\n---\n%s\n", "Command exit. Please input Enter.")
		exitInput <- true
	}

	// send exit
	ch <- true

	// close out
	if _, ok := stdout.(*io.PipeWriter); ok {
		_ = out.CloseWithError(io.ErrClosedPipe)
	}
}

// executePipeLineLocal is exec command in local machine.
// TDXX(blacknon): 利用中のShellでの実行+functionや環境変数、aliasの引き継ぎを行えるように実装.
func (ps *pShell) executeLocalPipeLine(pline pipeLine, in io.Reader, out *io.PipeWriter,
	ch chan<- bool, kill chan bool) {
	// set stdin/stdout
	stdin := setInput(in)
	stdout := setOutput(out)

	// set HistoryResult
	var stdoutw io.Writer

	m := new(sync.Mutex)

	if stdout == os.Stdout {
		pw := ps.NewHistoryWriter("localhost", nil, m)

		defer func() { _ = pw.CloseWithError(io.ErrClosedPipe) }()

		stdoutw = io.MultiWriter(pw, stdout)
	} else {
		stdoutw = stdout
	}

	// delete command prefix(`!`)
	pline.Args[0] = regexp.MustCompile(`^!`).ReplaceAllString(pline.Args[0], "")

	cmd := exec.Command("sh", "-c", strings.Join(pline.Args, " ")) // nolint:gosec

	// set stdin, stdout, stderr
	cmd.Stdin = stdin
	if ps.Options.LocalCommandNotRecordResult {
		cmd.Stdout = stdout
	} else { // default
		cmd.Stdout = stdoutw
	}

	cmd.Stderr = os.Stderr

	// run command
	_ = cmd.Start()

	// get signal and kill
	p := cmd.Process

	go func() {
		<-kill

		_ = p.Kill()
	}()

	// wait command
	_ = cmd.Wait()

	// close out, or write pShellHistory
	if _, ok := stdout.(*io.PipeWriter); ok {
		_ = out.CloseWithError(io.ErrClosedPipe)
	}

	// send exit
	ch <- true
}

// wait ps.wait.
func wait(num int, ch <-chan bool) {
	for i := 0; i < num; i++ {
		<-ch
	}
}

// setInput ..
func setInput(in io.Reader) (stdin io.Reader) {
	if reflect.ValueOf(in).IsNil() {
		stdin = os.Stdin
	} else {
		stdin = in
	}

	return
}

// setOutput ...
func setOutput(out io.Writer) (stdout io.Writer) {
	if reflect.ValueOf(out).IsNil() {
		stdout = os.Stdout
	} else {
		stdout = out
	}

	return
}
