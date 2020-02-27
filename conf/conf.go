// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// Package conf is a package used to read configuration file (~/.bssh.toml).
package conf

import (
	"crypto/md5" // nolint
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bingoohuang/gou/pbe"
	"github.com/spf13/viper"

	"github.com/BurntSushi/toml"
	"github.com/bingoohuang/bssh/common"
)

// Config is Struct that stores the entire configuration file
type Config struct {
	Extra    ExtraConfig
	Log      LogConfig
	Shell    ShellConfig
	Include  map[string]IncludeConfig
	Includes IncludesConfig
	Common   ServerConfig
	Server   map[string]ServerConfig
	Proxy    map[string]ProxyConfig

	SSHConfig map[string]OpenSSHConfig

	grouping map[string]map[string]ServerConfig
}

// ExtraConfig store extra configs
type ExtraConfig struct {
	// Passphrase used to decrypt {PBE}xxx
	Passphrase string
	// DisableGrouping disable server names grouping
	DisableGrouping bool
	// DisableAutoEncryptPwd disable auto PBE passwords in config file.
	DisableAutoEncryptPwd bool
}

// LogConfig store the contents about the terminal log.
// The log file name is created in "YYYYmmdd_HHMMSS_servername.log" of the specified directory.
type LogConfig struct {
	// Enable terminal logging.
	Enable bool

	// Add a timestamp at the beginning of the terminal log line.
	Timestamp bool

	// Specifies the directory for creating terminal logs.
	Dir string `toml:"dirpath"`
}

// ShellConfig structure for storing bssh-shell settings.
type ShellConfig struct {
	// prompt
	Prompt  string `toml:"PROMPT"`  // bssh shell prompt
	OPrompt string `toml:"OPROMPT"` // bssh shell output prompt

	// message,title etc...
	Title string

	// history file
	HistoryFile string `toml:"histfile"`

	// pre | post command setting
	PreCmd  string `toml:"pre_cmd"`
	PostCmd string `toml:"post_cmd"`
}

// IncludeConfig specify the configuration file to include (ServerConfig only).
type IncludeConfig struct {
	Path string
}

// IncludesConfig specify the configuration file to include (ServerConfig only).
// Struct that can specify multiple files in array.
type IncludesConfig struct {
	// example:
	// 	path = [
	// 		 "~/.bssh.d/home.toml"
	// 		,"~/.bssh.d/cloud.toml"
	// 	]
	Path []string
}

// ServerConfig structure for holding SSH connection information
type ServerConfig struct {
	// templates, host:port user/pass
	Tmpl  string
	Group []string

	// Connect basic Setting
	Addr string
	Port string
	User string

	// Connect auth Setting
	Pass           string
	Passes         []string
	Key            string
	KeyCommand     string   `toml:"keycmd"`
	KeyCommandPass string   `toml:"keycmdpass"`
	KeyPass        string   `toml:"keypass"`
	Keys           []string `toml:"keys"` // "keypath::passphrase"
	Cert           string
	CertKey        string `toml:"certkey"`
	CertKeyPass    string `toml:"certkeypass"`

	CertPKCS11  bool `toml:"certpkcs11"`
	AgentAuth   bool `toml:"agentauth"`
	SSHAgentUse bool `toml:"ssh_agent"`
	PKCS11Use   bool `toml:"pkcs11"`
	// x11 forwarding setting
	X11 bool

	SSHAgentKeyPath []string `toml:"ssh_agent_key"` // "keypath::passphrase"

	PKCS11Provider string `toml:"pkcs11provider"` // PKCS11 Provider PATH
	PKCS11PIN      string `toml:"pkcs11pin"`      // PKCS11 PIN code

	// pre | post command setting
	PreCmd  string `toml:"pre_cmd"`
	PostCmd string `toml:"post_cmd"`

	// proxy setting
	ProxyType    string `toml:"proxy_type"`
	Proxy        string
	ProxyCommand string `toml:"proxy_cmd"` // OpenSSH type proxy setting

	// local rcfile setting
	LocalRcUse       string   `toml:"local_rc"` // yes|no (default: yes)
	LocalRcPath      []string `toml:"local_rc_file"`
	LocalRcDecodeCmd string   `toml:"local_rc_decode_cmd"`

	// local/remote port forwarding setting
	PortForwardMode   string `toml:"port_forward"`        // [`L`,`l`,`LOCAL`,`local`]|[`R`,`r`,`REMOTE`,`remote`]
	PortForwardLocal  string `toml:"port_forward_local"`  // port forward (local). "host:port"
	PortForwardRemote string `toml:"port_forward_remote"` // port forward (remote). "host:port"

	// Dynamic Port Forwarding setting
	DynamicPortForward string `toml:"dynamic_port_forward"` // ex.) "11080"
	Note               string

	// Connection Timeout second
	ConnectTimeout int `toml:"connect_timeout"`

	// Server Alive
	ServerAliveCountMax      int `toml:"alive_max"`
	ServerAliveCountInterval int `toml:"alive_interval"`
}

// ProxyConfig struct that stores Proxy server settings connected via http and socks5.
type ProxyConfig struct {
	Addr      string
	Port      string
	User      string
	Pass      string
	Proxy     string
	ProxyType string `toml:"proxy_type"`
	Note      string
}

// OpenSSHConfig to read OpenSSH configuration file.
//
// WARN: This struct is not use...
type OpenSSHConfig struct {
	Path    string // This is preferred
	Command string
	ServerConfig
}

// ReadConf load configuration file and return Config structure
// TDXX(blacknon): リファクタリング！(v0.6.1) 外出しや処理のまとめなど
func ReadConf(confPath string) (config Config) {
	confPath = common.ExpandHomeDir(confPath)

	if !common.IsExist(confPath) {
		fmt.Printf("Config file(%s) Not Found.\nPlease create file.\n\n", confPath)
		fmt.Printf("sample: %s\n", "https://raw.githubusercontent.com/bingoohuang/bssh/master/example/config.toml")
		os.Exit(1) // nolint gomnd
	}

	config.Server = map[string]ServerConfig{}
	config.SSHConfig = map[string]OpenSSHConfig{}

	// Read config file
	if _, err := toml.DecodeFile(confPath, &config); err != nil {
		fmt.Println(err)
		os.Exit(1) // nolint gomnd
	}

	viper.Set(pbe.PbePwd, config.Extra.Passphrase)

	// reduce common setting (in .bssh.toml servers)
	config.parseConfigServers(config.Server, config.Common)

	// Read Openssh configs
	if len(config.SSHConfig) == 0 {
		if v, err := getOpenSSHConfig("~/.ssh/config", ""); err == nil {
			config.parseConfigServers(v, config.Common)
		}
	} else {
		for _, sshConfig := range config.SSHConfig {
			setCommon := serverConfigReduct(config.Common, sshConfig.ServerConfig)

			if v, err := getOpenSSHConfig(sshConfig.Path, sshConfig.Command); err == nil {
				config.parseConfigServers(v, setCommon)
			}
		}
	}

	config.appendIncludePaths()
	config.readIncludeFiles()

	// Check Config Parameter
	if !checkFormatServerConf(config) {
		os.Exit(1) // nolint gomnd
	}

	config.parseGroups()

	return config
}

func (cf *Config) appendIncludePaths() {
	// for append includes to include.path
	if len(cf.Includes.Path) == 0 {
		return
	}

	if cf.Include == nil {
		cf.Include = map[string]IncludeConfig{}
	}

	for _, includePath := range cf.Includes.Path {
		unixTime := time.Now().Unix()
		keyString := strings.Join([]string{string(unixTime), includePath}, "_")

		hasher := md5.New() // nolint
		_, _ = hasher.Write([]byte(keyString))
		key := hex.EncodeToString(hasher.Sum(nil))

		// append config.Include[key]
		cf.Include[key] = IncludeConfig{common.ExpandHomeDir(includePath)}
	}
}

func (cf *Config) readIncludeFiles() {
	if len(cf.Include) == 0 {
		return
	}

	for _, v := range cf.Include {
		var includeConf Config

		// user path
		path := common.ExpandHomeDir(v.Path)

		// Read include config file
		_, err := toml.DecodeFile(path, &includeConf)
		if err != nil {
			panic(err)
		}

		// reduce common setting
		setCommon := serverConfigReduct(cf.Common, includeConf.Common)

		// map init
		if len(cf.Server) == 0 {
			cf.Server = map[string]ServerConfig{}
		}

		// add include file serverconf
		cf.parseConfigServers(includeConf.Server, setCommon)
	}
}

func (cf *Config) parseConfigServers(configServers map[string]ServerConfig, setCommon ServerConfig) {
	tmplConfigs := make([]tmplConfig, 0)

	for key, value := range configServers {
		setValue := serverConfigReduct(setCommon, value)
		cf.Server[key] = setValue

		if value.Tmpl != "" {
			delete(cf.Server, key)

			tmplConfigs = append(tmplConfigs, tmplConfig{
				k: key, c: setValue, t: ParseTmpl(setValue.Tmpl)})
		}
	}

	cf.tmplServers(tmplConfigs)
}

// checkFormatServerConf checkes format of server config.
//
// Note: Checking Addr, User and authentications
// having a value. No checking a validity of each fields.
//
// See also: checkFormatServerConfAuth function.
func checkFormatServerConf(c Config) (isFormat bool) {
	isFormat = true

	for k, v := range c.Server {
		// Address Set Check
		if v.Addr == "" {
			fmt.Printf("%s: 'addr' is not set.\n", k)

			isFormat = false
		}

		// User Set Check
		if v.User == "" {
			fmt.Printf("%s: 'user' is not set.\n", k)

			isFormat = false
		}

		if !checkFormatServerConfAuth(v) {
			fmt.Printf("%s: Authentication information is not set.\n", k)

			isFormat = false
		}
	}

	return
}

// checkFormatServerConfAuth checkes format of server config authentication.
//
// Note: Checking Pass, Key, Cert, AgentAuth, PKCS11Use, PKCS11Provider, Keys or
// Passes having a value. No checking a validity of each fields.
func checkFormatServerConfAuth(c ServerConfig) (isFormat bool) {
	isFormat = false
	if c.Pass != "" || c.Key != "" || c.Cert != "" {
		isFormat = true
	}

	if c.AgentAuth {
		isFormat = true
	}

	if c.PKCS11Use {
		_, err := os.Stat(c.PKCS11Provider)
		if err == nil {
			isFormat = true
		}
	}

	if len(c.Keys) > 0 || len(c.Passes) > 0 {
		isFormat = true
	}

	return
}

// serverConfigReduct returns a new server config that set perConfig field to
// childConfig empty filed.
func serverConfigReduct(perConfig, childConfig ServerConfig) ServerConfig {
	result := ServerConfig{}

	// struct to map
	perConfigMap, _ := common.StructToMap(&perConfig)
	childConfigMap, _ := common.StructToMap(&childConfig)

	resultMap := common.MapReduce(perConfigMap, childConfigMap)
	_ = common.MapToStruct(resultMap, &result)

	return result
}

// GetNameList return a list of server names from the Config structure.
func (cf *Config) GetNameList() (nameList []string) {
	for k := range cf.Server {
		nameList = append(nameList, k)
	}

	return nameList
}

// GetNameSortedList return a list of server names from the Config structure.
func (cf *Config) GetNameSortedList() (nameList []string) {
	nameList = cf.GetNameList()
	sort.Strings(nameList)

	return nameList
}
