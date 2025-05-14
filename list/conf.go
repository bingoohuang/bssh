package list

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/ngg/ss"
	"github.com/mozillazg/go-pinyin"
)

// ShowServersView shows view for servers.
func ShowServersView(cf *conf.Config, prompt string, names []string, isMulti bool) []string {
	group := showGroupsView(cf)

	// View List And Get Select Line
	l := new(Info)
	l.Prompt = prompt
	l.NameList = cf.FilterNamesByGroup(group, names)
	hostInfoEnabled := cf.HostInfoEnabled.Get()
	if hostInfoEnabled {
		l.SetTitle([]string{"ServerName", "Connect Info # Note", "Host Info"})
	} else {
		l.SetTitle([]string{"ServerName", "Connect Info # Note"})
	}
	l.RowFn = func(name string) string {
		s := cf.Server[name]
		note := s.Note

		chinesePinyinInitials := GetChinesePinyinInitials(note)
		if chinesePinyinInitials != "" {
			note += "(" + chinesePinyinInitials + ")"
		}
		if s.PassPbeEncrypted {
			note = "*" + note
		}

		hostInfo := cf.HostInfo[name]
		row := name +
			"\t" + s.User + "@" + s.Addr + ss.If(s.Port != "", ":"+s.Port, "") +
			" # " + strings.TrimSpace(note)
		if hostInfoEnabled {
			row += "\t" + strings.TrimSpace(hostInfo.Info)
		}

		return row
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
	if !cf.Extra.Grouping.Get() || len(cf.GetGrouping()) <= 1 {
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

// GetChinesePinyinInitials 将字符串中的汉字为拼音首字母。
func GetChinesePinyinInitials(text string) string {
	a := pinyin.NewArgs()
	a.Style = pinyin.FirstLetter // 设置为首字母风格

	result := ""
	for _, r := range text {
		char := string(r)
		isChinese, _ := regexp.MatchString(`^\p{Han}+$`, char) // 检查是否是汉字
		if isChinese {
			pinyinResult := pinyin.Pinyin(char, a)
			if len(pinyinResult) > 0 && len(pinyinResult[0]) > 0 {
				result += pinyinResult[0][0] // 取第一个字的第一个字母
			}
		}
	}
	return result
}
