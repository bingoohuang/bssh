package ssh

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/blacknon/lssh/conf"
	"github.com/shavac/gexpect"
)

func tmpFileName() string {
	var n uint64
	binary.Read(rand.Reader, binary.LittleEndian, &n)
	return strconv.FormatUint(n, 36) + ".lssh.tmp"
}

// OS ssh wrapper(terminal connect)
func ConnectSshTerminal(connectServer string, confList conf.Config) int {
	// Get ssh config value
	connectHost := confList.Server[connectServer].User + "@" + confList.Server[connectServer].Addr
	connectPort := "22"
	if confList.Server[connectServer].Port != "" {
		connectPort = confList.Server[connectServer].Port
	}
	connectPass := confList.Server[connectServer].Pass
	connectKey := confList.Server[connectServer].Key

	// ssh command Args
	// "/usr/bin/ssh -o 'StrictHostKeyChecking no' -o 'NumberOfPasswordPrompts 1' connectUser@connectAddr -p connectPort"
	sshCmd := "/usr/bin/ssh -o 'StrictHostKeyChecking no' -o 'NumberOfPasswordPrompts 1' " + connectHost + " -p " + connectPort
	if connectKey != "" {
		// "/usr/bin/ssh -o 'StrictHostKeyChecking no' -o 'NumberOfPasswordPrompts 1' -i connectKey connectUser@connectAddr -p connectPort"
		sshCmd = "/usr/bin/ssh -o 'StrictHostKeyChecking no' -o 'NumberOfPasswordPrompts 1' -i " + connectKey + " " + connectHost + " -p " + connectPort
	}

	// log Enable
	execCmd := ""
	if confList.Log.Enable == true {
		logDirPath := confList.Log.Dir
		execOS := runtime.GOOS
		execCmd = "/usr/bin/script"

		// ~ replace User current Directory
		usr, _ := user.Current()
		logDirPath = strings.Replace(logDirPath, "~", usr.HomeDir, 1)

		// mkdir logDIr
		if err := os.MkdirAll(logDirPath, 0755); err != nil {
			fmt.Println(err)
			return 1
		}

		// Golang time.format YYYYmmdd_HHMMSS = "20060102_150405".(https://golang.org/src/time/format.go)
		logFile := time.Now().Format("20060102_150405") + "_" + connectServer + ".log"
		logFilePATH := logDirPath + "/" + logFile
		awkCmd := ">(awk '{print strftime(\"%F %T \") $0}{fflush() }'>>" + logFilePATH + ")"

		// OS check
		if execOS == "linux" || execOS == "android" {
			execCmd = "/usr/bin/script -qf -c \"" + sshCmd + "\" " + awkCmd
		} else {
			execCmd = "/usr/bin/script -qF " + awkCmd + " " + sshCmd
		}

	} else {
		execCmd = sshCmd
	}

	// exec ssh command
	child, _ := gexpect.NewSubProcess("/bin/bash", "-c", execCmd)

	if err := child.Start(); err != nil {
		fmt.Println(err)
		return 1
	}
	defer child.Close()

	// Password Input
	if connectPass != "" {
		passwordPrompt := "word:"
		if idx, _ := child.ExpectTimeout(20*time.Second, regexp.MustCompile(passwordPrompt)); idx >= 0 {
			child.SendLine(connectPass)

		} else {
			fmt.Println("ssh connect timeout.")
			return 1
		}
	}

	// timeout
	child.InteractTimeout(2419200 * time.Second)
	return 0
}

// remote ssh server exec command only
func ConnectSshCommand(connectServerList []string, confList conf.Config, terminalMode bool, execRemoteCmd ...string) int {
	// Create tmp file
	stdinTemp, err := ioutil.TempFile("", tmpFileName())
	if err != nil {
		panic(err)
	}
	defer os.Remove(stdinTemp.Name())

	// Stdin only pipes
	if terminal.IsTerminal(syscall.Stdin) == false {
		io.Copy(stdinTemp, os.Stdin)
	}

	// for command exec
	for _, connectServer := range connectServerList {
		connectPort := "22"
		if confList.Server[connectServer].Port != "" {
			connectPort = confList.Server[connectServer].Port
		}
		connectPass := confList.Server[connectServer].Pass
		connectKey := confList.Server[connectServer].Key

		// Set ssh client config
		config := &ssh.ClientConfig{}
		if connectKey != "" {
			// Read PublicKey
			buffer, err := ioutil.ReadFile(connectKey)
			if err != nil {
				return 1
			}
			key, err := ssh.ParsePrivateKey(buffer)
			if err != nil {
				return 1
			}

			// Create ssh client config for KeyAuth
			config = &ssh.ClientConfig{
				User: confList.Server[connectServer].User,
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(key)},
				Timeout: 60 * time.Second,
			}
		} else {
			// Create ssh client config for PasswordAuth
			config = &ssh.ClientConfig{
				User: confList.Server[connectServer].User,
				Auth: []ssh.AuthMethod{
					ssh.Password(connectPass)},
				Timeout: 60 * time.Second,
			}
		}

		connectHostPort := confList.Server[connectServer].Addr + ":" + connectPort

		conn, err := ssh.Dial("tcp", connectHostPort, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot connect %v: %v", connectHostPort, err)
			return 1
		}
		defer conn.Close()

		session, err := conn.NewSession()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot open new session: %v", err)
			return 1
		}
		defer session.Close()

		go func() {
			time.Sleep(2419200 * time.Second)
			conn.Close()
		}()

		var stdoutBuf bytes.Buffer

		session.Stdout = &stdoutBuf
		session.Stderr = &stdoutBuf
		if terminalMode == true {
			modes := ssh.TerminalModes{
				ssh.ECHO:          0,
				ssh.TTY_OP_ISPEED: 14400,
				ssh.TTY_OP_OSPEED: 14400,
			}

			if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
				session.Close()
				fmt.Errorf("request for pseudo terminal failed: %s", err)
			}
		}

		// stdin tmp file Open.
		stdinTempRead, _ := os.OpenFile(stdinTemp.Name(), os.O_RDONLY, 0644)
		session.Stdin = stdinTempRead

		execRemoteCmdString := strings.Join(execRemoteCmd, " ")

		err = session.Run(execRemoteCmdString)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			if ee, ok := err.(*ssh.ExitError); ok {
				return ee.ExitStatus()
			}
		}

		// Get stdout
		stdoutBufArray := regexp.MustCompile("\r\n|\n\r|\n|\r").Split(stdoutBuf.String(), -1)
		for i, v := range stdoutBufArray {
			if i == len(stdoutBufArray)-1 {
				break
			}
			if len(connectServerList) > 1 {
				// Add server name line head.
				fmt.Println(connectServer+":", v)
			} else {
				fmt.Println(v)
			}
		}

	}
	return 0
}
