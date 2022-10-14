// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// NOTE:
// The file in which code for the sort function used mainly with the lsftp ls command is written.

package sftp

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/misc"
	"github.com/blacknon/textcol"
	"github.com/dustin/go-humanize"
	"github.com/pkg/sftp"
	"github.com/urfave/cli"
)

// sftpLs ...
type sftpLs struct {
	Client *Connect
	Files  []os.FileInfo
	Passwd string
	Groups string
}

// ClientReadFile read file from client.
func ClientReadFile(client *Connect, file string) (string, error) {
	f, err := client.Connect.Open(file)
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// getRemoteLsData ...
func (r *RunSftp) getRemoteLsData(client *Connect, path string) (lsdata sftpLs, err error) {
	// get symlink
	p, err := client.Connect.ReadLink(path)
	if err == nil {
		path = p
	}

	lstat, err := client.Connect.Lstat(path)
	if err != nil {
		return lsdata, err
	}

	// get path data
	var data []os.FileInfo
	if lstat.IsDir() {
		// get directory list data
		data, err = client.Connect.ReadDir(path)
		if err != nil {
			return lsdata, err
		}
	} else {
		data = []os.FileInfo{lstat}
	}

	passwd, err := ClientReadFile(client, "/etc/passwd")
	if err != nil {
		return lsdata, err
	}

	groups, err := ClientReadFile(client, "/etc/group")
	if err != nil {
		return lsdata, err
	}

	return sftpLs{Client: client, Files: data, Passwd: passwd, Groups: groups}, err
}

// ls exec and print out remote ls data.
func (r *RunSftp) ls(args []string) {
	app := cli.NewApp()
	app.CustomAppHelpTemplate = helptext
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "1", Usage: "list one file per line"},
		cli.BoolFlag{Name: "a", Usage: "do not ignore entries starting with"},
		cli.BoolFlag{Name: "f", Usage: "do not sort"},
		cli.BoolFlag{Name: "h", Usage: "with -l, print sizes like 1K 234M 2G etc."},
		cli.BoolFlag{Name: "l", Usage: "use a long listing format"},
		cli.BoolFlag{Name: "n", Usage: "list numeric user and group IDs"},
		cli.BoolFlag{Name: "r", Usage: "reverse order while sorting"},
		cli.BoolFlag{Name: "S", Usage: "sort by file size, largest first"},
		cli.BoolFlag{Name: "t", Usage: "sort by modification time, newest first"},
	}
	app.Name = "ls"
	app.Usage = "bssh ftp build-in command: ls [remote machine ls]"
	app.ArgsUsage = misc.Path
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = r.lsAction
	args = common.ParseArgs(app.Flags, args)

	_ = app.Run(args)
}

func (r *RunSftp) lsAction(c *cli.Context) error {
	// argpath
	argpath := c.Args().First()

	// get directory files data
	exit := make(chan bool)
	lsdata := map[string]sftpLs{}
	m := new(sync.Mutex)

	for s, cl := range r.Client {
		server, client := s, cl

		go r.doLs(lsdata, c, exit, m, client, server, argpath)
	}

	// wait get directory data
	for range r.Client {
		<-exit
	}

	switch {
	case c.Bool("l"): // long list format
		r.longList(lsdata, c)

	case c.Bool("1"): // list 1 file per line
		// for list
		for server, data := range lsdata {
			data.Client.Output.Create(server)
			w := data.Client.Output.NewWriter()

			for _, f := range data.Files {
				name := f.Name()
				fmt.Fprintf(w, "%s\n", name)
			}
		}

	default: // default
		for server, data := range lsdata {
			// get header width
			data.Client.Output.Create(server)
			w := data.Client.Output.NewWriter()
			headerWidth := len(data.Client.Output.Prompt)

			var item []string
			for _, f := range data.Files {
				item = append(item, f.Name())
			}

			textcol.Output = w
			textcol.Padding = headerWidth
			textcol.PrintColumns(&item, 2)
		}
	}

	return nil
}

func (r *RunSftp) longList(lsdata map[string]sftpLs, c *cli.Context) {
	// set tabwriter
	tabw := new(tabwriter.Writer)
	tabw.Init(os.Stdout, 0, 1, 1, ' ', 0)

	// get maxSizeWidth
	var maxSizeWidth int

	var sizestr string

	for _, data := range lsdata {
		for _, f := range data.Files {
			if c.Bool("h") {
				sizestr = humanize.Bytes(uint64(f.Size()))
			} else {
				sizestr = strconv.FormatUint(uint64(f.Size()), 10)
			}

			// set sizestr max length
			if maxSizeWidth < len(sizestr) {
				maxSizeWidth = len(sizestr)
			}
		}
	}

	// print list ls
	for server, data := range lsdata {
		// get prompt
		data.Client.Output.Create(server)
		prompt := data.Client.Output.GetPrompt()

		// for get data
		for _, f := range lsdata[server].Files {
			r.listFile(f, c, lsdata, server, maxSizeWidth, tabw, prompt)
		}
	}

	tabw.Flush()
}

func (r *RunSftp) listFile(f os.FileInfo, c *cli.Context, lsdata map[string]sftpLs,
	server string, maxSizeWidth int, tabw io.Writer, prompt string,
) {
	sys := f.Sys()

	// TDXX(blacknon): count hardlink (2列目)の取得方法がわからないため、わかったら追加。
	var uid, gid uint32

	var size uint64

	var user, group, timestr, sizestr string

	if stat, ok := sys.(*sftp.FileStat); ok {
		uid = stat.UID
		gid = stat.GID
		size = stat.Size
		timestamp := time.Unix(int64(stat.Mtime), 0)
		timestr = timestamp.Format("2006 01-02 15:04:05")
	}

	// Switch with or without -n option.
	if c.Bool("n") {
		user = strconv.FormatUint(uint64(uid), 10)
		group = strconv.FormatUint(uint64(gid), 10)
	} else {
		user, _ = common.GetNameFromID(lsdata[server].Passwd, uid)
		group, _ = common.GetNameFromID(lsdata[server].Groups, gid)
	}

	// Switch with or without -h option.
	if c.Bool("h") {
		sizestr = humanize.Bytes(size)
	} else {
		sizestr = strconv.FormatUint(size, 10)
	}

	// set data
	data := new(sftpLsData)
	data.Mode = f.Mode().String()
	data.User = user
	data.Group = group
	data.Size = sizestr
	data.Time = timestr
	data.Path = f.Name()

	if len(lsdata) == 1 {
		// set print format
		format := "%s\t%s\t%s\t%" + strconv.Itoa(maxSizeWidth) + "s\t%s\t%s\n"

		// write data
		fmt.Fprintf(tabw, format, data.Mode, data.User, data.Group, data.Size, data.Time, data.Path)
	} else {
		// set print format
		format := "%s\t%s\t%s\t%s\t%" + strconv.Itoa(maxSizeWidth) + "s\t%s\t%s\n"

		// write data
		fmt.Fprintf(tabw, format, prompt, data.Mode, data.User, data.Group, data.Size, data.Time, data.Path)
	}
}
