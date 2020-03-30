package common

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/bingoohuang/gou/file"
)

// LastLogBean struct the bean of last log.
type LastLogBean struct {
	Args            []string
	ServerNames     []string
	CurrentLastMode bool
}

// lastLog records last command arguments and server name.
var lastLog LastLogBean // nolint

// SaveArgsLastLog saves the last log.
func SaveArgsLastLog() {
	if lastLog.CurrentLastMode {
		return
	}

	lastLog.Args = os.Args

	saveLastLogFile()
}

// SaveServerNameLastLog saves the last log.
func SaveServerNameLastLog(serverNames []string) {
	if lastLog.CurrentLastMode {
		return
	}

	lastLog.ServerNames = serverNames

	saveLastLogFile()
}

func saveLastLogFile() {
	bytes, _ := json.Marshal(lastLog)
	lastFile := ExpandHomeDir("~/.bssh.last")
	_ = ioutil.WriteFile(lastFile, bytes, 0644)
}

// ReadLastLog reads last Log
func ReadLastLog() (*LastLogBean, bool) {
	lastFile := ExpandHomeDir("~/.bssh.last")
	stat := file.Stat(lastFile)

	if stat == file.NotExists {
		return nil, false
	}

	confContent, err := ioutil.ReadFile(lastFile)

	if err != nil {
		return nil, false
	}

	if err := json.Unmarshal(confContent, &lastLog); err != nil {
		return nil, false
	}

	return &lastLog, true
}
