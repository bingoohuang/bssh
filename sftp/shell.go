// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/misc"

	prompt "github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
	shellwords "github.com/mattn/go-shellwords"
)

// TDXX(blacknon): catコマンド相当の機能を追加する

// shell Shell mode function
func (r *RunSftp) shell() {
	// start message
	fmt.Println("Start lsftp...")

	// print select server
	r.Run.PrintSelectServer()

	// create go-prompt
	p := prompt.New(
		r.Executor,
		r.Completer,
		// prompt.OptionPrefix(pShellPrompt),
		prompt.OptionLivePrefix(r.CreatePrompt),
		prompt.OptionInputTextColor(prompt.Green),
		prompt.OptionPrefixTextColor(prompt.Blue),
		prompt.OptionMaxSuggestion(16),                                              // nolint gomnd
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator), // test
	)

	// start go-prompt
	p.Run()
}

// Executor sftp Shell mode function
func (r *RunSftp) Executor(command string) { // nolint funlen
	p := shellwords.NewParser()
	p.ParseEnv = true
	cmdline, _ := p.Parse(command)

	// switch command
	switch cmdline[0] {
	case "bye", "exit", "quit":
		os.Exit(0)
	case "help", "?":
	// case "cat":
	case "cd": // change remote directory
		r.cd(cmdline)
	case misc.Chgrp:
		r.chgrp(cmdline)
	case "chmod":
		r.chmod(cmdline)
	case misc.Chown:
		r.chown(cmdline)
	// case "copy":
	case "df":
		r.df(cmdline)
	case misc.Get:
		r.get(cmdline)
	case "lcd":
		r.lcd(cmdline)
	case misc.Lls:
		r.lls(cmdline)
	case misc.Lmkdir:
		r.lmkdir(cmdline)
	// case "ln":
	case "lpwd":
		r.lpwd()
	case "ls":
		r.ls(cmdline)
	// case "lumask":
	case misc.Mkdir:
		r.mkdir(cmdline)
	case misc.Put:
		r.put(cmdline)
	case "pwd":
		r.pwd()
	case misc.Rename:
		r.rename(cmdline)
	case "rm":
		r.rm(cmdline)
	case misc.Rmdir:
		r.rmdir(cmdline)
	case misc.Symlink:
		r.symlink(cmdline)
	// case "tree":
	// case "!": // ! or !command...
	case "": // none command...
	default:
		fmt.Println("Command Not Found...")
	}
}

// Completer sftp Shell mode function
func (r *RunSftp) Completer(t prompt.Document) []prompt.Suggest {
	// result
	var suggest []prompt.Suggest

	// Get cursor left
	left := t.CurrentLineBeforeCursor()

	// Get cursor char(string)
	char := ""
	if len(left) > 0 {
		char = string(left[len(left)-1])
	}

	cmdline := strings.Split(left, " ")
	if len(cmdline) == 1 { // nolint gomnd
		suggest = []prompt.Suggest{
			{Text: "bye", Description: "Quit lsftp"},
			// {Text: "cat", Description: "Open file"},
			{Text: "cd", Description: "Change remote directory to 'path'"},
			{Text: misc.Chgrp, Description: "Change group of file 'path' to 'grp'"},
			{Text: misc.Chown, Description: "Change owner of file 'path' to 'own'"},
			// {Text: "copy", Description: "Copy to file from 'remote' or 'local' to 'remote' or 'local'"},
			{Text: "df", Description: "Display statistics for current directory or filesystem containing 'path'"},
			{Text: "exit", Description: "Quit lsftp"},
			{Text: misc.Get, Description: "Download file"},
			// {Text: "reget", Description: "Resume download file"},
			// {Text: "reput", Description: "Resume upload file"},
			{Text: "help", Description: "Display this help text"},
			{Text: "lcd", Description: "Change local directory to 'path'"},
			{Text: misc.Lls, Description: "Display local directory listing"},
			{Text: misc.Lmkdir, Description: "Create local directory"},
			// {Text: "ln", Description: "Link remote file (-s for symlink)"},
			{Text: "lpwd", Description: "Print local working directory"},
			{Text: "ls", Description: "Display remote directory listing"},
			// {Text: "lumask", Description: "Set local umask to 'umask'"},
			{Text: misc.Mkdir, Description: "Create remote directory"},
			// {Text: "progress", Description: "Toggle display of progress meter"},
			{Text: misc.Put, Description: "Upload file"},
			{Text: "pwd", Description: "Display remote working directory"},
			{Text: "quit", Description: "Quit sftp"},
			{Text: misc.Rename, Description: "Rename remote file"},
			{Text: "rm", Description: "Delete remote file"},
			{Text: misc.Rmdir, Description: "Remove remote directory"},
			{Text: misc.Symlink, Description: "Create symbolic link"},
			// {Text: "tree", Description: "Tree view remote directory"},
			// {Text: "!command", Description: "Execute 'command' in local shell"},
			{Text: "!", Description: "Escape to local shell"},
			{Text: "?", Description: "Display this help text"},
		}
	} else { // command pattern
		switch cmdline[0] {
		case "cd":
			return r.PathComplete(true, 1, t)
		case misc.Chgrp:
			// TDXX(blacknon): そのうち追加 ver0.6.1
		case misc.Chown:
			// TDXX(blacknon): そのうち追加 ver0.6.1
		case "df":
			suggest = []prompt.Suggest{
				{Text: "-h", Description: "print sizes in powers of 1024 (e.g., 1023M)"},
				{Text: "-i", Description: "list inode information instead of block usage"},
			}
			return prompt.FilterHasPrefix(suggest, t.GetWordBeforeCursor(), false)
		case misc.Get:
			// TDXX(blacknon): オプションを追加したら引数の数から減らす処理が必要
			switch strings.Count(t.CurrentLineBeforeCursor(), " ") {
			case 1: // nolint gomnd remote
				return r.PathComplete(true, 1, t)
			case 2: // nolint gomnd local
				return r.PathComplete(false, 2, t)
			}

		case "lcd":
			return r.PathComplete(false, 1, t)
		case misc.Lls:
			// switch options or path
			switch {
			case common.Contains([]string{"-"}, char):
				suggest = []prompt.Suggest{
					{Text: "-1", Description: "list one file per line"},
					{Text: "-a", Description: "do not ignore entries starting with"},
					{Text: "-f", Description: "do not sort"},
					{Text: "-h", Description: "with -l, print sizes like 1K 234M 2G etc."},
					{Text: "-l", Description: "use a long listing format"},
					{Text: "-n", Description: "list numeric user and group IDs"},
					{Text: "-r", Description: "reverse order while sorting"},
					{Text: "-S", Description: "sort by file size, largest first"},
					{Text: "-t", Description: "sort by modification time, newest first"},
				}
				return prompt.FilterHasPrefix(suggest, t.GetWordBeforeCursor(), false)

			default:
				return r.PathComplete(false, 1, t)
			}
		case misc.Lmkdir:
			switch {
			case common.Contains([]string{"-"}, char):
				suggest = []prompt.Suggest{
					{Text: "-p", Description: "no error if existing, make parent directories as needed"},
				}
				return prompt.FilterHasPrefix(suggest, t.GetWordBeforeCursor(), false)

			default:
				return r.PathComplete(false, 1, t)
			}

		// case "ln":
		case "lpwd":
		case "ls":
			// switch options or path
			switch {
			case common.Contains([]string{"-"}, char):
				suggest = []prompt.Suggest{
					{Text: "-1", Description: "list one file per line"},
					{Text: "-a", Description: "do not ignore entries starting with"},
					{Text: "-f", Description: "do not sort"},
					{Text: "-h", Description: "with -l, print sizes like 1K 234M 2G etc."},
					{Text: "-l", Description: "use a long listing format"},
					{Text: "-n", Description: "list numeric user and group IDs"},
					{Text: "-r", Description: "reverse order while sorting"},
					{Text: "-S", Description: "sort by file size, largest first"},
					{Text: "-t", Description: "sort by modification time, newest first"},
				}
				return prompt.FilterHasPrefix(suggest, t.GetWordBeforeCursor(), false)

			default:
				return r.PathComplete(true, 1, t)
			}

		// case "lumask":
		case misc.Mkdir:
			switch {
			case common.Contains([]string{"-"}, char):
				suggest = []prompt.Suggest{
					{Text: "-p", Description: "no error if existing, make parent directories as needed"},
				}

			default:
				return r.PathComplete(true, 1, t)
			}

		case misc.Put:
			// TDXX(blacknon): オプションを追加したら引数の数から減らす処理が必要
			// TDXX（blacknon）：添加选项后，有必要减少参数数量
			switch strings.Count(t.CurrentLineBeforeCursor(), " ") {
			case 1: // nolint gomnd local
				return r.PathComplete(false, 1, t)
			case 2: // nolint gomnd remote
				return r.PathComplete(true, 2, t)
			}
		case "pwd":
		case "quit":
		case misc.Rename:
			return r.PathComplete(true, 1, t)
		case "rm":
			return r.PathComplete(true, 1, t)
		case misc.Rmdir:
			return r.PathComplete(true, 1, t)
		case misc.Symlink:
			// TDXX(blacknon): そのうち追加 ver0.6.1
		// case "tree":

		default:
		}
	}

	// return prompt.FilterHasPrefix(suggest, t.GetWordBeforeCursor(), true)
	return prompt.FilterHasPrefix(suggest, t.GetWordBeforeCursor(), false)
}

// PathComplete ...
func (r *RunSftp) PathComplete(remote bool, num int, t prompt.Document) []prompt.Suggest {
	var suggest []prompt.Suggest

	// Get cursor left
	left := t.CurrentLineBeforeCursor()

	// Get cursor char(string)
	char := ""
	if len(left) > 0 {
		char = string(left[len(left)-1])
	}

	// get last slash place
	word := t.GetWordBeforeCursor()
	sp := strings.LastIndex(word, "/")

	if len(word) > 0 {
		word = word[sp+1:]
	}

	if remote {
		switch {
		case common.Contains([]string{"/"}, char): // char is slash or
			r.GetRemoteComplete(t.GetWordBeforeCursor())
		case common.Contains([]string{" "}, char) && strings.Count(t.CurrentLineBeforeCursor(), " ") == num:
			r.GetRemoteComplete(t.GetWordBeforeCursor())
		}

		suggest = r.RemoteComplete
	} else {
		switch {
		case common.Contains([]string{"/"}, char): // char is slash or
			r.GetLocalComplete(t.GetWordBeforeCursor())
		case common.Contains([]string{" "}, char) && strings.Count(t.CurrentLineBeforeCursor(), " ") == num:
			r.GetLocalComplete(t.GetWordBeforeCursor())
		}

		suggest = r.LocalComplete
	}

	return prompt.FilterHasPrefix(suggest, word, false)
}

// GetRemoteComplete ...
func (r *RunSftp) GetRemoteComplete(path string) {
	// create map
	m := map[string][]string{}
	exit := make(chan bool)

	// create sync mutex
	sm := new(sync.Mutex)

	// connect client...
	for s, c := range r.Client {
		server, client := s, c

		go func() {
			defer func() { exit <- true }()

			rpath, err := r.prepareRemotePath(path, client)
			if err != nil {
				return
			}

			// get path list
			globlist, err := client.Connect.Glob(rpath)
			if err != nil {
				return
			}

			// set glob list
			for _, p := range globlist {
				p = filepath.Base(p)

				sm.Lock()
				m[p] = append(m[p], server)
				sm.Unlock()
			}
		}()
	}

	for range r.Client {
		<-exit
	}

	// create suggest slice
	p := make([]prompt.Suggest, 0, len(m))
	// create suggest
	for path, hosts := range m {
		// join hosts
		h := strings.Join(hosts, ",")

		p = append(p, prompt.Suggest{Text: path, Description: "remote path. from:" + h})
	}

	// sort
	sort.SliceStable(p, func(i, j int) bool { return p[i].Text < p[j].Text })

	// set suggest to struct
	r.RemoteComplete = p
}

func (r *RunSftp) prepareRemotePath(path string, client *Connect) (string, error) {
	rpath := ""

	switch {
	case filepath.IsAbs(path):
		rpath = path
	case strings.HasPrefix(path, "~/"):
		rpath = filepath.Join(client.Pwd, path[2:])
	default:
		rpath = filepath.Join(client.Pwd, path)
	}

	stat, err := client.Connect.Stat(rpath)
	if err != nil {
		return "", err
	}

	if stat.IsDir() {
		rpath += "/*"
	} else {
		rpath += "*"
	}

	return rpath, nil
}

// GetLocalComplete ...
func (r *RunSftp) GetLocalComplete(path string) {
	path = common.ExpandHomeDir(path)
	stat, err := os.Lstat(path)

	if err != nil {
		return
	}

	// dir check
	var lpath string
	if stat.IsDir() {
		lpath = filepath.Join(path, "*")
	} else {
		lpath = path + "*"
	}

	// get globlist
	globlist, err := filepath.Glob(lpath)
	if err != nil {
		return
	}

	// create suggest slice
	p := make([]prompt.Suggest, len(globlist))
	// set path
	for i, lp := range globlist {
		lp = filepath.Base(lp)
		p[i] = prompt.Suggest{
			Text:        lp,
			Description: "local path.",
		}
	}

	r.LocalComplete = p
}

// CreatePrompt creates prompt.
func (r *RunSftp) CreatePrompt() (p string, result bool) {
	p = "bssh ftp>> "
	return p, true
}
