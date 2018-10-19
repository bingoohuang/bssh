package check

import (
	"fmt"
	"os"
	"strings"
)

// @brief
//
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

// @brief:
//    parse lscp args path
func ParseScpPath(arg string) (isRemote bool, path string) {
	argArray := strings.SplitN(arg, ":", 2)

	// check split count
	if len(argArray) < 2 {
		isRemote = false
		path = argArray[0]
		return
	}

	pathType := strings.ToLower(argArray[0])
	switch pathType {
	// local
	case "local", "l":
		isRemote = false
		path = argArray[1]

	// remote
	case "remote", "r":
		isRemote = true
		path = argArray[1]

	// false
	default:
		isRemote = false
		path = ""

		// error
		fmt.Fprintln(os.Stderr, "The format of the specified argument is incorrect.")
		os.Exit(1)
	}

	return
}

// @brief
//    check type.
func CheckTypeError(isFromInRemote, isFromInLocal, isToRemote bool, countHosts int) {
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
