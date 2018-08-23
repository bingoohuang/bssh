package ssh

import (
	"bytes"
	"fmt"
	"os"
)

func (r *Run) cmd() {
	serverNameMaxLength := getServerNameMaxLength(r.ServerList)
	finished := make(chan bool)

	// print header
	r.printSelectServer()
	r.printRunCommand()
	r.printProxy()
	fmt.Println() // print newline

	for i, server := range r.ServerList {
		c := new(Connect)
		c.Server = server
		c.Conf = r.Conf
		c.IsTerm = r.IsTerm
		c.IsParallel = r.IsParallel
		serverListIndex := i

		// create session
		session, err := c.CreateSession()
		if err != nil {
			go func() {
				fmt.Fprintf(os.Stderr, "cannot connect session %v, %v\n", outColorStrings(serverListIndex, c.Server), err)
				finished <- true
			}()
			continue
		}

		session.Stdin = bytes.NewReader(r.StdinData)

		// run command
		outputChan := make(chan string)
		go func(outputChan chan string) {
			c.RunCmdGetOutput(session, r.ExecCmd, outputChan)
			close(outputChan)
		}(outputChan)

		// get command output
		go func(outputChan chan string) {
			for outputLine := range outputChan {
				if len(r.ServerList) > 1 {
					lineHeader := fmt.Sprintf("%-*s", serverNameMaxLength, c.Server)
					fmt.Println(outColorStrings(serverListIndex, lineHeader)+" :: ", outputLine)
				} else {
					fmt.Println(outputLine)
				}
			}
			finished <- true
		}(outputChan)
	}

	// wait all finish
	for i := 1; i <= len(r.ServerList); i++ {
		<-finished
	}

	return
}

func outColorStrings(num int, inStrings string) (str string) {
	// 1=Red,2=Yellow,3=Blue,4=Magenta,0=Cyan
	color := 31 + num%5
	str = fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, inStrings)
	return
}

func getServerNameMaxLength(serverList []string) (serverNameMaxLength int) {
	serverNameMaxLength = 0
	for _, serverName := range serverList {
		if serverNameMaxLength < len(serverName) {
			serverNameMaxLength = len(serverName)
		}
	}
	return
}
