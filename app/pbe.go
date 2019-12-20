package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/bingoohuang/gou/str"

	"github.com/bingoohuang/gou/pbe"
	"github.com/blacknon/lssh"
	"github.com/blacknon/lssh/conf"
	"github.com/blacknon/lssh/misc"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

// nolint
const lpbeAppHelpTemplate = `NAME:
    {{.Name}} - {{.Usage}}
USAGE:
    {{.HelpName}} {{if .VisibleFlags}}[options]{{end}} [commands...]
    {{if len .Authors}}
AUTHOR:
    {{range .Authors}}{{ . }}{{end}}
    {{end}}{{if .Commands}}
COMMANDS:
    {{range .Commands}}{{if not .HideHelp}}{{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
OPTIONS:
    {{range .VisibleFlags}}{{.}}
    {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
    {{.Copyright}}
    {{end}}{{if .Version}}
VERSION:
    {{.Version}}
    {{end}}
USAGE:
    # encrypt the passwords in the default ~/.lssh.conf.
	{{.Name}}
    
	# encrypt the passwords in the conf file located at filepath.
	{{.Name}} -F filepath...
 
    # decrypt the passwords in the default ~/.lssh.conf.
	{{.Name}} -r
    
	# decrypt the  passwords in the conf file located at filepath.
	{{.Name}} -r -F filepath...
`

// Lpbe pbe passwords in the conf file
func Lpbe() (app *cli.App) {
	cli.AppHelpTemplate = lpbeAppHelpTemplate

	app = cli.NewApp()
	app.Name = "lssh pbe"
	app.Usage = "{PBE} the clear passwords in the conf file"
	app.Copyright = misc.Copyright
	app.Version = lssh.AppVersion

	// Set options
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "file,F", Value: str.PickFirst(homedir.Expand("~/.lssh.conf")),
			Usage: "config `filepath`."},
		cli.StringFlag{Name: "passphrase,P", Usage: "passphrase used to PBE clear passwords"},
		cli.BoolFlag{Name: "reverse,r", Usage: "reverse action to decrypt pbe"},
		cli.BoolFlag{Name: "help,h", Usage: "print this help"},
	}
	app.EnableBashCompletion = true
	app.HideHelp = true
	app.Action = lpbeAction

	return app
}

// lpbeAction actions lpbe functions.
func lpbeAction(c *cli.Context) error {
	if c.Bool("help") {
		_ = cli.ShowAppHelp(c)

		os.Exit(0)
	}

	confpath := c.String("file")
	confContent, err := ioutil.ReadFile(confpath)

	if err != nil {
		fmt.Println(confpath, "read error", err.Error())

		os.Exit(1)
	}

	data := conf.ReadConf(confpath)

	passphrase := c.String("passphrase")
	if passphrase == "" {
		passphrase = data.Extra.Passphrase
	}

	viper.Set(pbe.PbePwd, passphrase)

	m := make(map[string]string)

	collectPasses(c.Bool("reverse"), data, m)

	strContent := string(confContent)
	for clear, encrypted := range m {
		strContent = strings.ReplaceAll(strContent, clear, encrypted)
	}

	stat, _ := os.Stat(confpath)

	err = ioutil.WriteFile(confpath, []byte(strContent), stat.Mode())
	if err != nil {
		fmt.Println(confpath, "write error", err.Error())

		os.Exit(1)
	}

	return nil
}

func collectPasses(reverse bool, data conf.Config, m map[string]string) {
	for _, sc := range data.Server {
		addPass(reverse, sc.Pass, m)

		for _, p := range sc.Passes {
			addPass(reverse, p, m)
		}
	}

	addPass(reverse, data.Common.Pass, m)

	for _, p := range data.Common.Passes {
		addPass(reverse, p, m)
	}

	for _, sc := range data.Proxy {
		addPass(reverse, sc.Pass, m)
	}

	for _, sc := range data.SSHConfig {
		addPass(reverse, sc.Pass, m)

		for _, p := range sc.Passes {
			addPass(reverse, p, m)
		}
	}
}

func addPass(reverse bool, pass string, m map[string]string) {
	isPBE := strings.HasPrefix(pass, "{PBE}")

	if reverse {
		if isPBE {
			if _, ok := m[pass]; !ok {
				m[pass], _ = pbe.Ebp(pass)
			}
		}

		return
	}

	if isPBE || pass == "" || pass == "na" {
		return
	}

	if _, ok := m[pass]; !ok {
		m[pass], _ = pbe.Pbe(pass)
	}
}
