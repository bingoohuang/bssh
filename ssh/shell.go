// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package ssh

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bingoohuang/bssh/conf"

	"github.com/bingoohuang/bssh/misc"

	"github.com/bingoohuang/bssh/common"
	sshlib "github.com/blacknon/go-sshlib"
	"golang.org/x/crypto/ssh"
)

// run shell
// nolint:funlen
func (r *Run) shell() (err error) {
	server := r.ServerList[0]
	config, ok := r.Conf.Server[server]
	isTempHost := !ok
	if isTempHost {
		config = r.parseDirectServer(server)
	}

	// check count AuthMethod
	if len(r.serverAuthMethodMap[server]) == 0 {
		msg := fmt.Sprintf("Error: %s has No AuthMethod.\n", server)

		return errors.New(msg) // nolint:goerr113
	}

	r.overwritePortForwardConfig(&config)
	r.overwriteBashrcConfig(&config)

	// header
	r.PrintSelectServer()
	r.printPortForward(config.PortForwardMode, config.PortForwardLocal, config.PortForwardRemote)
	r.printDynamicPortForward(config.DynamicPortForward)
	r.printProxy(server)

	if config.LocalRcUse == misc.Yes {
		fmt.Fprintf(os.Stderr, "Information   :This connect use local bashrc.\n")
	}

	// Create sshlib.Connect (Connect Proxy loop)
	connect, err := r.CreateSSHConnect(server)
	if err != nil {
		return err
	}

	// Create session
	session, err := connect.CreateSession()
	if err != nil {
		return err
	}

	if isTempHost {
		r.Conf.WriteTempHosts(server)
	}

	r.sshAgent(&config, connect, session)

	err = r.portForwarding(&config, connect)

	if config.DynamicPortForward != "" { // Dynamic Port Forwarding
		go func() {
			if err := connect.TCPDynamicForward("localhost", config.DynamicPortForward); err != nil {
				fmt.Println(err)
			}
		}()
	}

	// switch check Not-execute flag
	// TDXX(blacknon): Backgroundフラグを実装したら追加
	switch {
	case r.IsNone:
		r.noneExecute()

	default:
		// run pre local command
		if config.PreCmd != "" {
			execLocalCommand(config.PreCmd)
		}

		// defer run post local command
		if config.PostCmd != "" {
			defer execLocalCommand(config.PostCmd)
		}

		// if terminal log enable
		logConf := r.Conf.Log
		if logConf.Enable {
			logPath := r.getLogPath(server)
			connect.SetLog(logPath, logConf.Timestamp)
		}

		// TDXX(blacknon): local rc file add
		if config.LocalRcUse == misc.Yes {
			err = localrcShell(connect, session, config.LocalRcPath, config.LocalRcDecodeCmd)
		} else {
			err = connect.Shell(session)
		}
	}

	return err
}

func (r *Run) parseDirectServer(server string) conf.ServerConfig {
	sc, ok := conf.ParseDirectServer(server)
	if ok {
		r.Conf.Server[server] = sc
		r.registerAuthMapPassword(server, sc.Pass)
	}

	return r.Conf.Server[server]
}

func (r *Run) sshAgent(config *conf.ServerConfig, connect *sshlib.Connect, session *ssh.Session) {
	// ssh-agent
	if config.SSHAgentUse {
		connect.Agent = r.agent
		connect.ForwardSshAgent(session)
	}
}

func (r *Run) overwriteBashrcConfig(config *conf.ServerConfig) {
	// OverWrite local bashrc use
	if r.IsBashrc {
		config.LocalRcUse = misc.Yes
	}

	// OverWrite local bashrc not use
	if r.IsNotBashrc {
		config.LocalRcUse = "no"
	}
}

func (r *Run) overwritePortForwardConfig(config *conf.ServerConfig) {
	// OverWrite port forward mode
	if r.PortForwardMode != "" {
		config.PortForwardMode = r.PortForwardMode
	}

	// OverWrite port forwarding address
	if r.PortForwardLocal != "" && r.PortForwardRemote != "" {
		config.PortForwardLocal = r.PortForwardLocal
		config.PortForwardRemote = r.PortForwardRemote
	}

	// OverWrite dynamic port forwarding
	if r.DynamicPortForward != "" {
		config.DynamicPortForward = r.DynamicPortForward
	}
}

func (r *Run) portForwarding(config *conf.ServerConfig, connect *sshlib.Connect) (err error) {
	// Local/Remote Port Forwarding
	if config.PortForwardLocal != "" && config.PortForwardRemote != "" {
		// port forwarding
		switch config.PortForwardMode {
		case "L", "":
			err = connect.TCPLocalForward(config.PortForwardLocal, config.PortForwardRemote)
		case "R":
			err = connect.TCPRemoteForward(config.PortForwardLocal, config.PortForwardRemote)
		}

		if err != nil {
			fmt.Println(err)
		}
	}

	return err
}

// getLogPath return log file path.
func (r *Run) getLogPath(server string) (logPath string) {
	// check regex
	// if ~/.ssh/config, in ":"
	reg := regexp.MustCompile(`:`)

	if reg.MatchString(server) {
		slice := strings.SplitN(server, ":", 2)
		server = slice[1]
	}

	dir, err := r.getLogDirPath(server)
	if err != nil {
		log.Println(err)
	}

	file := time.Now().Format("20060102_150405") + "_" + server + ".log"
	logPath = filepath.Join(dir, file)

	return logPath
}

// getLogDirPath return log directory path.
func (r *Run) getLogDirPath(server string) (dir string, err error) {
	logConf := r.Conf.Log

	// expansion variable
	dir = common.ExpandHomeDir(logConf.Dir)
	dir = strings.Replace(dir, "<Date>", time.Now().Format("20060102"), 1)
	dir = strings.Replace(dir, "<Hostname>", server, 1)

	// create directory
	err = os.MkdirAll(dir, 0700)

	return dir, err
}

// runLocalRcShell connect to remote shell using local bashrc.
func localrcShell(connect *sshlib.Connect, session *ssh.Session, localrcPath []string, decoder string) (err error) {
	// set default bashrc
	if len(localrcPath) == 0 {
		localrcPath = []string{"~/.bashrc"}
	}

	// get bashrc base64 data
	rcData, err := common.GetFilesBase64(localrcPath)
	if err != nil {
		return err
	}

	// command
	s := "bash --noprofile --rcfile<(echo %s|((base64 --help|grep -q coreutils)&&base64 -d<(cat)||base64 -D<(cat) ))"
	cmd := fmt.Sprintf(s, rcData)

	// decode command
	if decoder != "" {
		cmd = fmt.Sprintf("bash --noprofile --rcfile <(echo %s | %s)", rcData, decoder)
	}

	return connect.CmdShell(session, cmd)
}

// noneExecute is not execute command and shell.
func (r *Run) noneExecute() {
	for range time.After(500 * time.Millisecond) { // nolint:gomnd

	}
}
