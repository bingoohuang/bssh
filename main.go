package main

import (
	"fmt"
	"os"
	"os/user"
	"sort"

	arg "github.com/alexflint/go-arg"
	"github.com/blacknon/lssh/check"
	"github.com/blacknon/lssh/conf"
	"github.com/blacknon/lssh/list"
	"github.com/blacknon/lssh/ssh"
)

// Command Option
type CommandOption struct {
	Host     string   `arg:"-H,help:Connect servername"`
	List     bool     `arg:"-l,help:print server list"`
	File     string   `arg:"-f,help:config file path"`
	Terminal bool     `arg:"-T,help:Run specified command at terminal"`
	Command  []string `arg:"positional,help:Remote Server exec command."`
}

// Version Setting
func (CommandOption) Version() string {
	return "lssh v0.2"
}

func main() {
	// Exec Before Check
	check.OsCheck()
	check.DefCommandExistCheck()

	// Set default value
	usr, _ := user.Current()
	defaultConfPath := usr.HomeDir + "/.lssh.conf"

	// get Command Option
	var args struct {
		CommandOption
	}

	// Default Value
	args.File = defaultConfPath
	arg.MustParse(&args)

	// set option value
	connectHost := args.Host
	listFlag := args.List
	configFile := args.File
	terminalExec := args.Terminal
	execRemoteCmd := args.Command

	// Get List
	listConf := conf.ConfigCheckRead(configFile)

	// Get Server Name List (and sort List)
	nameList := conf.GetNameList(listConf)
	sort.Strings(nameList)

	// if --list option
	if listFlag == true {
		fmt.Fprintf(os.Stderr, "lssh Server List:\n")
		for v := range nameList {
			fmt.Fprintf(os.Stderr, "  %s\n", nameList[v])
		}
		os.Exit(0)
	}

	selectServer := ""
	if connectHost != "" {
		if check.CheckInputServerExit(connectHost, nameList) == false {
			fmt.Fprintln(os.Stderr, "Input Server not found from list.")
			os.Exit(1)
		} else {
			selectServer = connectHost
		}
	} else {
		// View List And Get Select Line
		selectServer = list.DrawList(nameList, listConf)
		if selectServer == "ServerName" {
			fmt.Fprintln(os.Stderr, "Server not selected.")
			os.Exit(1)
		}
	}

	// Exec Connect ssh
	if len(execRemoteCmd) != 0 {
		// Connect SSH Terminal
		os.Exit(ssh.ConnectSshCommand(selectServer, listConf, terminalExec, execRemoteCmd...))
	} else {
		// Exec SSH Command Only
		os.Exit(ssh.ConnectSshTerminal(selectServer, listConf))
	}
}
