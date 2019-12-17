package common

import homedir "github.com/mitchellh/go-homedir"

// ExpandHomeDir expands the ~ in the path if it is available.
func ExpandHomeDir(f string) string {
	if s, err := homedir.Expand(f); err == nil {
		return s
	}

	return f
}
