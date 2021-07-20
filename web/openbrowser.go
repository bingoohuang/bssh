package web

import (
	"fmt"
	gourl "net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/chzyer/readline"
)

// IsTerminal returns whether the UI is known to be tied to an
// interactive terminal (as opposed to being redirected to a file).
// from https://github.com/google/pprof/blob/master/pprof.go
func IsTerminal() bool {
	return readline.IsTerminal(syscall.Stdout)
}

// https://github.com/google/pprof/blob/master/internal/driver/webui.go
func OpenBrowser(url string) {
	// Construct URL.
	u, _ := gourl.Parse(url)
	if !IsTerminal() {
		fmt.Println(u.String())
		return
	}

	// Give server a little time to get ready.
	time.Sleep(time.Millisecond * 500)

	for _, b := range browsers() {
		args := strings.Split(b, " ")
		if len(args) == 0 {
			continue
		}
		viewer := exec.Command(args[0], append(args[1:], u.String())...)
		viewer.Stderr = os.Stderr
		if err := viewer.Start(); err == nil {
			return
		}
	}
	// No visualizer succeeded, so just print URL.
	fmt.Println(u.String())
}

// browsers returns a list of commands to attempt for web visualization.
// https://github.com/google/pprof/blob/master/internal/driver/commands.go
func browsers() []string {
	var cmds []string
	if userBrowser := os.Getenv("BROWSER"); userBrowser != "" {
		cmds = append(cmds, userBrowser)
	}
	switch runtime.GOOS {
	case "darwin":
		cmds = append(cmds, "/usr/bin/open")
	case "windows":
		cmds = append(cmds, "cmd /c start")
	default:
		// Commands opening browsers are prioritized over xdg-open, so browser()
		// command can be used on linux to open the .svg file generated by the -web
		// command (the .svg file includes embedded javascript so is best viewed in
		// a browser).
		cmds = append(cmds, []string{"chrome", "google-chrome", "chromium", "firefox", "sensible-browser"}...)
		if os.Getenv("DISPLAY") != "" {
			// xdg-open is only for use in a desktop environment.
			cmds = append(cmds, "xdg-open")
		}
	}
	return cmds
}