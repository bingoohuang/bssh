// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package ssh

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/gou/pbe"
	sshlib "github.com/blacknon/go-sshlib"

	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/bssh/misc"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// TDXX(blacknon): 自動再接続機能の追加(v0.6.1)
//     autosshのように、接続が切れた際に自動的に再接続を試みる動作をさせたい
//     パラメータでの有効・無効指定が必要になる。

// TDXX(blacknon): リバースでのsshfsの追加(v0.6.1以降？)
//     lsshfs実装後になるか？ssh接続時に、指定したフォルダにローカルの内容をマウントさせて読み取らせる。
//     うまくやれれば、ローカルのスクリプトなどをそのままマウントさせて実行させたりできるかもしれない。
//     Socketかなにかでトンネルさせて、あとは指定したディレクトリ配下をそのままFUSEでファイルシステムとして利用できるように書けばいける…？
//
//     【参考】
//         - https://github.com/rom1v/rsshfs
//         - https://github.com/hanwen/go-fuse
//         - https://gitlab.com/dns2utf8/revfs/

// Run running info.
type Run struct {
	ServerList []string
	Conf       conf.Config

	// Mode value in
	//     - shell
	//     - cmd
	//     - pshell
	Mode string

	// tty use (-t option)
	IsTerm bool

	// parallel connect (-p option)
	IsParallel bool

	// not run (-N option)
	IsNone bool

	// x11 forwarding (-X option)
	X11 bool

	// use or not-use local bashrc.
	// IsNotBashrc takes precedence.
	IsBashrc    bool
	IsNotBashrc bool

	// enable/disable print header in command mode
	EnableHeader  bool
	DisableHeader bool

	// StdinData from pipe flag
	isStdinPipe bool

	// local/remote Port Forwarding
	PortForwardMode   string // L or R
	PortForwardLocal  string
	PortForwardRemote string

	// Dynamic Port Forwarding
	// set localhost port num (ex. 11080).
	DynamicPortForward string

	// Exec command
	ExecCmd []string

	// Agent is ssh-agent.
	// In agent.Agent or agent.ExtendedAgent.
	agent interface{}

	// AuthMethodMap is
	// map of AuthMethod summarized in Run overall
	authMethodMap map[AuthKey][]ssh.AuthMethod

	// ServerAuthMethodMap is
	// Map of AuthMethod used by target server
	serverAuthMethodMap map[string][]ssh.AuthMethod

	decodedPasswordMap map[string]bool
	confFile           string
}

// NewRun news a Run struct.
func NewRun(confFile string) *Run {
	r := &Run{
		decodedPasswordMap: make(map[string]bool),
		confFile:           confFile,
	}

	return r
}

// AuthKey define auth key\
type AuthKey struct {
	// auth type:
	//   - password
	//   - agent
	//   - key
	//   - cert
	//   - pkcs11
	Type string

	// auth type value:
	//   - key(path)
	//     ex.) ~/.ssh/id_rsa
	//   - cert(path)
	//     ex.) ~/.ssh/id_rsa.crt
	//   - pkcs11(libpath)
	//     ex.) /usr/local/lib/opensc-pkcs11.so
	Value string
}

// PathSet ...
type PathSet struct {
	Base      string
	PathSlice []string
}

const (
	// AuthKeyPassword auth by Password
	AuthKeyPassword = "password"
	// AuthKeyKey auth by key
	AuthKeyKey = "key"
	// AuthKeyCert  auth by cert
	AuthKeyCert = "cert"
	// AuthKeyPkcs11 auth by pkcs11
	AuthKeyPkcs11 = "pkcs11"
)

// Start ssh connect
func (r *Run) Start() {
	var err error

	// Get stdin data(pipe)
	// TDXX(blacknon): os.StdinをReadAllで全部読み込んでから処理する方式だと、ストリームで処理出来ない
	//                 (全部読み込み終わるまで待ってしまう)ので、Reader/Writerによるストリーム処理に切り替える(v0.6.0)
	//                 => flagとして検知させて、あとはpushPipeWriterにos.Stdinを渡すことで対処する
	if runtime.GOOS != "windows" {
		stdin := 0
		if !terminal.IsTerminal(stdin) {
			r.isStdinPipe = true
		}
	}

	r.CreateAuthMethodMap()

	switch {
	case len(r.ExecCmd) > 0 && r.Mode == "cmd":
		r.cmd()
	case r.Mode == "shell":
		err = r.shell()
	case r.Mode == "pshell":
		err = r.pshell()
	default:
		return
	}

	if err != nil {
		fmt.Println(err)
	}
}

// PrintSelectServer is printout select server.
// use ssh login header.
func (r *Run) PrintSelectServer() {
	serverListStr := strings.Join(r.ServerList, ",")
	fmt.Fprintf(os.Stderr, "Select Server :%s\n", serverListStr)
}

// printRunCommand is printout run command.
// use ssh command run header.
func (r *Run) printRunCommand() {
	runCmdStr := strings.Join(r.ExecCmd, " ")
	fmt.Fprintf(os.Stderr, "Run Command   :%s\n", runCmdStr)
}

// printPortForward is printout port forwarding.
// use ssh command run header. only use shell().
func (r *Run) printPortForward(m, forwardLocal, forwardRemote string) {
	if forwardLocal != "" && forwardRemote != "" {
		var mode, arrow string

		switch m {
		case "L", "":
			mode = "LOCAL "
			arrow = " =>"
		case "R":
			mode = "REMOTE"
			arrow = "<= "
		}

		fmt.Fprintf(os.Stderr, "Port Forward  :%s\n", mode)
		fmt.Fprintf(os.Stderr, "               local[%s] %s remote[%s]\n", forwardLocal, arrow, forwardRemote)
	}
}

// printPortForward is printout port forwarding.
// use ssh command run header. only use shell().
func (r *Run) printDynamicPortForward(port string) {
	if port != "" {
		fmt.Fprintf(os.Stderr, "DynamicForward:%s\n", port)
		fmt.Fprintf(os.Stderr, "               %s\n", "connect Socks5.")
	}
}

// printProxy is printout proxy route.
// use ssh command run header. only use shell().
func (r *Run) printProxy(server string) {
	array := []string{}

	proxyRoute, err := getProxyRoute(server, r.Conf)
	if err != nil || len(proxyRoute) == 0 {
		return
	}

	// set localhost
	localhost := "localhost"

	// set target host
	targethost := server

	// add localhost
	array = append(array, localhost)

	for _, pxy := range proxyRoute {
		// separator
		var sep string
		if pxy.Type == misc.Command {
			sep = ":"
		} else {
			sep = "://"
		}

		// setup string
		str := "[" + pxy.Type + sep + pxy.Name
		if pxy.Port != "" {
			str += ":" + pxy.Port
		}

		str += "]"

		array = append(array, str)
	}

	// add target
	array = append(array, targethost)

	// print header
	header := strings.Join(array, " => ")
	fmt.Fprintf(os.Stderr, "Proxy         :%s\n", header)
}

func (r *Run) registerAutoEncryptPwd(oldPwd string) {
	if r.Conf.Extra.DisableAutoEncryptPwd || strings.HasPrefix(oldPwd, `{PBE}`) {
		return
	}

	if _, ok := r.decodedPasswordMap[oldPwd]; ok {
		return
	}

	r.decodedPasswordMap[oldPwd] = true
}

func readConfContent(confPath string) (string, error) {
	confPath = common.ExpandHomeDir(confPath)
	bytes, err := ioutil.ReadFile(confPath)

	return string(bytes), err
}

func writeConfContent(confPath, content string) error {
	confPath = common.ExpandHomeDir(confPath)
	stat, err := os.Stat(confPath)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(confPath, []byte(content), stat.Mode())
}

func (r *Run) autoEncryptPwd() {
	if r.Conf.Extra.DisableAutoEncryptPwd || len(r.decodedPasswordMap) == 0 {
		return
	}

	content, err := readConfContent(r.confFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read conf file %s, error %v\n", r.confFile, err)
		return
	}

	newContent := content

	for pwd := range r.decodedPasswordMap {
		newPwd, err := pbe.Pbe(pwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read conf file %s, error %v\n", r.confFile, err)
			return
		}

		r := regexp.QuoteMeta(pwd)
		reg := regexp.MustCompile(r)
		newContent = reg.ReplaceAllString(newContent, newPwd)
	}

	if newContent != content {
		if err := writeConfContent(r.confFile, newContent); err != nil {
			fmt.Fprintf(os.Stderr, "write conf file %s, error %v\n", r.confFile, err)
			return
		}
	}

	for pwd := range r.decodedPasswordMap {
		r.decodedPasswordMap[pwd] = false
	}
}

func (r *Run) createAuthMethodMapForServer(server string) {
	// get server config
	config := r.Conf.Server[server]

	// Password
	r.registerAuthMapPassword(server, config.Pass)

	// Multiple Password
	for _, pass := range config.Passes {
		r.registerAuthMapPassword(server, pass)
	}

	// PublicKey
	if err := r.registerAuthMapPublicKey(server, config.Key, config.KeyPass); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	// Multiple PublicKeys
	for _, key := range config.Keys {
		pair := strings.SplitN(key, "::", 2)
		keyName := pair[0]
		keyPass := ""

		if len(pair) > 1 { // nolint gomnd
			keyPass = pair[1]
		}

		if err := r.registerAuthMapPublicKey(server, keyName, keyPass); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
	}

	// Public Key Command
	// TDXX(blacknon): keyCommandの追加
	if err := r.registerAuthMapPublicKeyCommand(server, config.KeyCommand, config.KeyCommandPass); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	// Certificate
	if config.Cert != "" {
		keySigner, err := sshlib.CreateSignerPublicKeyPrompt(config.CertKey, config.CertKeyPass)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		if err := r.registerAuthMapCertificate(server, config.Cert, keySigner); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}

	// PKCS11
	if config.PKCS11Use {
		if err := r.registerAuthMapPKCS11(server, config.PKCS11Provider, config.PKCS11PIN); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// runCmdLocal exec command local machine.
// Mainly used in r.shell().
func execLocalCommand(cmd string) {
	out, _ := exec.Command("sh", "-c", cmd).CombinedOutput()
	fmt.Print(string(out))
}
