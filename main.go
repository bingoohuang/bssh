package main

import (
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"

	"github.com/blacknon/lssh/check"
	"github.com/blacknon/lssh/conf"
	"github.com/blacknon/lssh/list"
	"github.com/blacknon/lssh/ssh"
)

func main() {
	// Exec Before Check
	check.OsCheck()
	check.CommandExistCheck()

	// Default Config file
	confFilePath := "~/.lssh.conf"

	// Get ConfigFile Path
	usr, _ := user.Current()
	configFile := strings.Replace(confFilePath, "~", usr.HomeDir, 1)

	// Get List
	listConf := conf.ConfigCheckRead(configFile)

	// Get Server Name List (and sort List)
	nameList := conf.GetNameList(listConf)
	sort.Strings(nameList)

	// View List And Get Select Line
	selectServer := list.DrawList(nameList, listConf)

	if selectServer == "ServerName" {
		fmt.Println("Server not selected.")
		os.Exit(0)
	}

	// Connect SSH
	ssh.ConnectSsh(selectServer, listConf)
}
