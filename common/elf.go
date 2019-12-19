package common

import homedir "github.com/mitchellh/go-homedir"

// ExpandHomeDir expands the ~ in the path if it is available.
func ExpandHomeDir(f string) string {
	if s, err := homedir.Expand(f); err == nil {
		return s
	}

	return f
}

// Contains tells if s contains element e.
func Contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}

	return false
}
