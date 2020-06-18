// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package conf

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/bingoohuang/bssh/misc"

	"github.com/bingoohuang/bssh/common"
	"github.com/kevinburke/ssh_config"
)

// openOpenSSHConfig open the OpenSsh configuration file, return *ssh_config.Config.
func openOpenSSHConfig(path, command string) (cfg *ssh_config.Config, err error) {
	var rd io.Reader

	switch {
	case path != "": // 1st
		sshConfigFile := common.GetFullPath(path)
		rd, err = os.Open(sshConfigFile)
	case command != "": // 2nd
		var data []byte

		cmd := exec.Command("sh", "-c", command)
		data, err = cmd.Output()
		rd = bytes.NewReader(data)
	}

	// error check
	if err != nil {
		return
	}

	return ssh_config.Decode(rd)
}

// getOpenSSHConfig loads the specified OpenSsh configuration file and returns it in conf.ServerConfig format.
func getOpenSSHConfig(path, command string) (config map[string]ServerConfig, err error) {
	config = map[string]ServerConfig{}

	cfg, err := openOpenSSHConfig(path, command)
	if err != nil {
		return
	}

	// set name element
	ele := path
	if ele == "" {
		ele = "generate_sshconfig"
	}

	hostList := createHostList(cfg)

	for _, host := range hostList {
		serverName, serverConfig := createServerConfig(host, ele)
		config[serverName] = serverConfig
	}

	return config, err
}

func createServerConfig(host string, ele string) (string, ServerConfig) {
	serverConfig := ServerConfig{
		Addr:         ssh_config.Get(host, "HostName"),
		Port:         ssh_config.Get(host, "Port"),
		User:         ssh_config.Get(host, "User"),
		ProxyCommand: ssh_config.Get(host, "ProxyCommand"),
		PreCmd:       ssh_config.Get(host, "LocalCommand"),
		Note:         "from:" + ele,
	}

	if serverConfig.Addr == "" {
		serverConfig.Addr = host
	}

	key := ssh_config.Get(host, "IdentityFile")
	cert := ssh_config.Get(host, "Certificate")

	if cert != "" {
		serverConfig.Cert = cert
		serverConfig.CertKey = key
	} else {
		serverConfig.Key = key
	}

	pkcs11Provider := ssh_config.Get(host, "PKCS11Provider")
	if pkcs11Provider != "" {
		serverConfig.PKCS11Use = true
		serverConfig.PKCS11Provider = pkcs11Provider
	}

	x11 := ssh_config.Get(host, "ForwardX11")
	if x11 == misc.Yes {
		serverConfig.X11 = true
	}

	parseLocalPortForwarding(host, &serverConfig)

	parseRemotePortForwarding(host, &serverConfig)

	// Port forwarding (Dynamic forward)
	dynamicForward := ssh_config.Get(host, "DynamicForward")
	if dynamicForward != "" {
		serverConfig.DynamicPortForward = dynamicForward
	}

	serverName := ele + ":" + host

	return serverName, serverConfig
}

func parseRemotePortForwarding(host string, serverConfig *ServerConfig) {
	// Port forwarding (Remote forward)
	remoteForward := ssh_config.Get(host, "RemoteForward")
	if remoteForward == "" {
		return
	}

	array := strings.SplitN(remoteForward, " ", 2)

	if len(array) <= 1 {
		return
	}

	var e error

	_, e = strconv.Atoi(array[0])
	if e != nil { // localhost:8080
		serverConfig.PortForwardLocal = array[0]
	} else { // 8080
		serverConfig.PortForwardLocal = "localhost:" + array[0]
	}

	_, e = strconv.Atoi(array[1])
	if e != nil { // localhost:8080
		serverConfig.PortForwardRemote = array[1]
	} else { // 8080
		serverConfig.PortForwardRemote = "localhost:" + array[1]
	}
}

func parseLocalPortForwarding(host string, serverConfig *ServerConfig) {
	// Port forwarding (Local forward)
	localForward := ssh_config.Get(host, "LocalForward")
	if localForward == "" {
		return
	}

	array := strings.SplitN(localForward, " ", 2)

	if len(array) <= 1 {
		return
	}

	var e error

	_, e = strconv.Atoi(array[0])
	if e != nil { // localhost:8080
		serverConfig.PortForwardLocal = array[0]
	} else { // 8080
		serverConfig.PortForwardLocal = "localhost:" + array[0]
	}

	_, e = strconv.Atoi(array[1])
	if e != nil { // localhost:8080
		serverConfig.PortForwardRemote = array[1]
	} else { // 8080
		serverConfig.PortForwardRemote = "localhost:" + array[1]
	}
}

func createHostList(cfg *ssh_config.Config) []string {
	// Get Node names
	var hostList []string

	for _, h := range cfg.Hosts {
		// not supported wildcard host
		re := regexp.MustCompile(`\*`)

		for _, pattern := range h.Patterns {
			if !re.MatchString(pattern.String()) {
				hostList = append(hostList, pattern.String())
			}
		}
	}

	return hostList
}
