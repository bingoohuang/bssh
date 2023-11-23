// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// NOTE:
// The file in which code for the sort function used mainly with the lsftp ls command is written.

package sftp

import (
	"fmt"
	"io/ioutil"
	"os"
	pkguser "os/user"
	"strconv"
	"text/tabwriter"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/misc"
	"github.com/bingoohuang/gou/mat"
	"github.com/blacknon/textcol"
	"github.com/dustin/go-humanize"
	"github.com/thoas/go-funk"
	"github.com/urfave/cli"
)

// lls exec and print out local ls data.
func (r *RunSftp) lls(args []string) {
	// create app
	app := cli.NewApp()
	// app.UseShortOptionHandling = true

	// set help message
	app.CustomAppHelpTemplate = helptext

	// set parameter
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
	app.Name = misc.Lls
	app.Usage = "bssh ftp build-in command: lls [local machine ls]"
	app.ArgsUsage = misc.Path
	app.HideHelp = true
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Action = llsAction

	// parse short options
	args = common.ParseArgs(app.Flags, args)
	_ = app.Run(args)
}

func llsAction(c *cli.Context) error {
	// argpath
	argpath := c.Args().First()
	if argpath == "" {
		argpath = "./"
	}

	stat, err := os.Stat(argpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return nil
	}

	// check is directory
	var data []os.FileInfo

	if stat.IsDir() {
		if data, err = ioutil.ReadDir(argpath); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return nil
		}
	} else {
		data = append(data, stat)
	}

	switch {
	case c.Bool("l"): // long list format
		longList(data)
	case c.Bool("1"): // list 1 file per line
		funk.ForEach(data, func(f os.FileInfo) { fmt.Println(f.Name()) })
	default: // default
		item := funk.Map(data, func(f os.FileInfo) string { return f.Name() }).([]string)

		textcol.Output = os.Stdout
		textcol.Padding = 0
		textcol.PrintColumns(&item, 2)
	}

	return nil
}

func longList(data []os.FileInfo) {
	// set tabwriter
	tabw := new(tabwriter.Writer)
	tabw.Init(os.Stdout, 0, 1, 1, ' ', 0)

	maxSizeWidth, maxUserWidth, maxGroupWidth := 0, 0, 0
	lsData := make([]*sftpLsData, len(data))

	for i, f := range data {
		uid, gid, size, _ := SyscallStat(f)

		user := strconv.FormatUint(uint64(uid), 10)
		group := strconv.FormatUint(uint64(gid), 10)

		userData, _ := pkguser.LookupId(user)
		user += `(` + userData.Username + `)`
		maxUserWidth = mat.MaxInt(maxUserWidth, len(user))

		groupData, _ := pkguser.LookupGroupId(group)
		group += `(` + groupData.Name + `)`
		maxGroupWidth = mat.MaxInt(maxGroupWidth, len(group))

		sizestr := strconv.FormatUint(uint64(size), 10) + `(` + humanize.Bytes(uint64(size)) + `)`
		maxSizeWidth = mat.MaxInt(maxSizeWidth, len(sizestr))

		// set data
		lsData[i] = &sftpLsData{
			Mode: f.Mode().String(), User: user, Group: group, Size: sizestr,
			Time: f.ModTime().Format("2006 01-02 15:04:05"), Path: f.Name(),
		}
	}

	format := "%s\t%" + strconv.Itoa(maxUserWidth) + "s%" + strconv.Itoa(maxGroupWidth) +
		"s%" + strconv.Itoa(maxSizeWidth) + "s\t%s\t%s\n"

	for _, f := range lsData {
		fmt.Fprintf(tabw, format, f.Mode, f.User, f.Group, f.Size, f.Time, f.Path)
	}

	tabw.Flush()
}
