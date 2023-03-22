/*
check is a package used mainly for check processing required by bssh, content check of configuration file.
*/

package check

import (
	"fmt"
	"os"
	"strings"
)

// ExistServer returns true if inputServer exists in nameList.
func ExistServer(inputServer []string, nameList []string) bool {
	for _, nv := range nameList {
		for _, iv := range inputServer {
			if nv == iv {
				return true
			}
		}
	}

	return false
}

// ParseScpPath parses remote or local path string.
// Path string has a `:` delimiter.
// A prefix of path string is a scp location.
// A scp location is `local (l)` or `remote (r)`.
//
// arg examples:
//
//	Local path:
//	    local:/tmp/a.txt
//	    l:/tmp/a.txt
//	    /tmp/a.txt
//	Remote path:
//	    remote:/tmp/a.txt
//	    r:/tmp/a.txt
func ParseScpPath(arg string) (isRemote bool, path string) {
	argArray := strings.SplitN(arg, ":", 2)

	// check split count
	if len(argArray) < 2 {
		return false, argArray[0]
	}

	switch pathType := strings.ToLower(argArray[0]); pathType {
	case "local", "l": // local
		return false, argArray[1]
	case "remote", "r": // remote
		return true, replaceUserHome(argArray[1])
	default: // false
		_, _ = fmt.Fprintln(os.Stderr, "The format of the specified argument is incorrect.")
		os.Exit(1)
	}

	return false, ""
}

func replaceUserHome(s string) string {
	if strings.HasPrefix(s, "~") {
		return "." + s[1:]
	}

	return s
}

// EscapePath escapes characters (`\`, `;`, ` `).
func EscapePath(str string) (escapeStr string) {
	str = strings.Replace(str, "\\", "\\\\", -1)
	str = strings.Replace(str, ";", "\\;", -1)
	str = strings.Replace(str, " ", "\\ ", -1)

	return str
}

// TypeError validates from-remote, from-local, to-remote and host-counts.
func TypeError(isFromInRemote, isFromInLocal, isToRemote bool, countHosts int) {
	// from in local and remote
	if isFromInRemote && isFromInLocal {
		fmt.Fprintln(os.Stderr, "Can not set LOCAL and REMOTE to FROM path.")
		os.Exit(1)
	}

	// local only
	if !isFromInRemote && !isToRemote {
		fmt.Fprintln(os.Stderr, "It does not correspond LOCAL to LOCAL copy.")
		os.Exit(1)
	}

	// remote 2 remote and set host option
	if isFromInRemote && isToRemote && countHosts != 0 {
		fmt.Fprintln(os.Stderr, "In the case of REMOTE to REMOTE copy, it does not correspond to host option.")
		os.Exit(1)
	}
}
