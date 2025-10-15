package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/mitchellh/go-homedir"
)

func OpenBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("openbrowser: %v\n", err)
	}
}

func ExpandFile(filePath string) string {
	fp, err := homedir.Expand(filePath)
	if err != nil {
		log.Fatalf("expand key path %q error: %v", filePath, err)
	}

	if IsSymlinkFile(fp) {
		linkedFile, err := filepath.EvalSymlinks(fp)
		if err != nil {
			log.Fatalf("eval symlinks %q error: %v", fp, err)
		}
		fp = linkedFile
	}

	if _, err := os.Stat(fp); err != nil {
		log.Fatalf("stat key path %q error: %v", fp, err)
	}

	return fp
}

func IsSymlinkFile(name string) bool {
	st, err := os.Lstat(name)
	if err != nil {
		return false
	}

	return st.Mode()&os.ModeSymlink != 0
}
