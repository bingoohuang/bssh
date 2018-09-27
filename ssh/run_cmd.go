package ssh

import (
	"bytes"
	"fmt"
	"os"

	"github.com/blacknon/lssh/common"
)

func (r *Run) cmd() {
	finished := make(chan bool)

	// print header
	r.printSelectServer()
	r.printRunCommand()
	r.printProxy()
	fmt.Println() // print newline

	for i, server := range r.ServerList {
		count := i

		c := new(Connect)
		c.Server = server
		c.Conf = r.Conf
		c.IsTerm = r.IsTerm
		c.IsParallel = r.IsParallel

		// run command
		outputChan := make(chan string)
		go r.cmdRun(c, i, outputChan)

		// print command output
		if r.IsParallel {
			go func() {
				r.cmdPrintOutput(c, count, outputChan)
				finished <- true
			}()
		} else {
			r.cmdPrintOutput(c, count, outputChan)
		}
	}

	// wait all finish
	if r.IsParallel {
		for i := 1; i <= len(r.ServerList); i++ {
			<-finished
		}
	}

	return
}

func (r *Run) cmdRun(conn *Connect, serverListIndex int, outputChan chan string) {
	// create session
	session, err := conn.CreateSession()
	if err != nil {
		go func() {
			fmt.Fprintf(os.Stderr, "cannot connect session %v, %v\n", outColorStrings(serverListIndex, conn.Server), err)
		}()
		close(outputChan)
		return
	}

	// set stdin
	session.Stdin = bytes.NewReader(r.StdinData)

	// run command and get output data to outputChan
	conn.RunCmdWithOutput(session, r.ExecCmd, outputChan)
	close(outputChan)
}

func (r *Run) cmdPrintOutput(conn *Connect, serverListIndex int, outputChan chan string) {
	serverNameMaxLength := common.GetMaxLength(r.ServerList)

	for outputLine := range outputChan {
		if len(r.ServerList) > 1 {
			lineHeader := fmt.Sprintf("%-*s", serverNameMaxLength, conn.Server)
			fmt.Println(outColorStrings(serverListIndex, lineHeader)+" :: ", outputLine)
		} else {
			fmt.Println(outputLine)
		}
	}
}

func outColorStrings(num int, inStrings string) (str string) {
	// 1=Red,2=Yellow,3=Blue,4=Magenta,0=Cyan
	color := 31 + num%5
	str = fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, inStrings)
	return
}
