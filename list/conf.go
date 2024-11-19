package list

import (
	"fmt"
	"os"

	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/ngg/ss"
)

// ShowServersView shows view for servers.
func ShowServersView(cf *conf.Config, prompt string, names []string, isMulti bool) []string {
	group := showGroupsView(cf)

	// View List And Get Select Line
	l := new(Info)
	l.Prompt = prompt
	l.NameList = cf.FilterNamesByGroup(group, names)
	l.SetTitle([]string{"ServerName", "Connect Information", "Note"})
	l.RowFn = func(name string) string {
		s := cf.Server[name]
		note := s.Note
		if s.PassPbeEncrypted {
			note = "*" + note
		}

		return name + "\t" + s.User + "@" + s.Addr + ss.If(s.Port != "", ":"+s.Port, "") + "\t" + note
	}
	l.MultiFlag = isMulti

	l.View()
	selected := l.SelectName

	if selected[0] == "ServerName" {
		fmt.Fprintln(os.Stderr, "Server not selected.")
		os.Exit(1)
	}

	return selected
}

// showGroupsView shows view for groups.
func showGroupsView(cf *conf.Config) string {
	if cf.Extra.DisableGrouping || len(cf.GetGrouping()) <= 1 {
		return ""
	}

	// View List And Get Select Line
	l := new(Info)
	l.Prompt = "group>>"
	l.NameList = cf.GroupsNames()
	l.SetTitle([]string{"GroupName"})
	l.RowFn = func(name string) string { return name }
	l.MultiFlag = false

	l.View()
	selected := l.SelectName

	if selected[0] == "GroupName" {
		fmt.Fprintln(os.Stderr, "Group not selected.")
		os.Exit(1)
	}

	return selected[0]
}
