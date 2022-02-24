// Package conf is a package used to read configuration file (~/.bssh.toml).
package conf

import (
	"crypto/md5" // nolint
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bingoohuang/gossh/pkg/hostparse"

	_ "embed"

	"github.com/jedib0t/go-pretty/table"

	"github.com/bingoohuang/gou/str"

	"github.com/bingoohuang/gou/pbe"
	"github.com/spf13/viper"

	"github.com/BurntSushi/toml"
	"github.com/bingoohuang/bssh/common"
)

// Config is Struct that stores the entire configuration file.
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

	// DisableAutoEncryptPwd disable auto PBE passwords in config file.
	DisableAutoEncryptPwd bool
	Passphrase            string
	Hosts                 []string
	tempHostsFile         string
	tempHosts             map[string]bool
}

// ExtraConfig store extra configs.
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

// ServerConfig structure for holding SSH connection information.
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

	InitialCmd string `toml:"initial_cmd"`
	WebPort    int    `toml:"web_port"`
	ID         string `toml:"id"`

	Raw string // to register the raw template config, like `user:pass@host:port`
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

// ReadConf load configuration file and return Config structure.
func ReadConf(confPath string) (config Config) {
	confPath = common.ExpandHomeDir(confPath)
	checkConfPath(confPath)

	config.Server = map[string]ServerConfig{}
	config.SSHConfig = map[string]OpenSSHConfig{}

	// Read config file
	if _, err := toml.DecodeFile(confPath, &config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	config.loadTempHosts(confPath)

	viper.Set(pbe.PbePwd, str.EmptyThen(config.Extra.Passphrase, config.Passphrase))

	// reduce common setting (in .bssh.toml servers)
	config.parseConfigServers(config.Server, config.Common)

	for i, server := range config.Hosts {
		tmpls := hostparse.Parse(server)
		for j, tmpl := range tmpls {
			sc := ServerConfig{}
			createServerConfigFromHost(tmpl, &sc)

			if sc.ID == "" {
				sc.ID = generateKey(len(tmpls), len(config.Hosts), i, j)
			}
			config.Server[sc.ID] = sc
		}
	}

	// Read Openssh configs
	if len(config.SSHConfig) == 0 {
		if v, err := getOpenSSHConfig("~/.ssh/config", ""); err == nil {
			config.parseConfigServers(v, config.Common)
		}
	} else {
		for _, sshConfig := range config.SSHConfig {
			setCommon := ServerConfigDeduct(config.Common, sshConfig.ServerConfig)

			if v, err := getOpenSSHConfig(sshConfig.Path, sshConfig.Command); err == nil {
				config.parseConfigServers(v, setCommon)
			}
		}
	}

	config.appendIncludePaths()
	config.readIncludeFiles()

	// Check Config Parameter
	CheckFormatServerConf(config)
	config.parseGroups()

	return config
}

//go:embed conf.toml
var initBsshToml []byte

func checkConfPath(confPath string) {
	if common.IsExist(confPath) {
		return
	}

	fmt.Printf("Config file(%s) not found, auto create one, please edit later.\n", confPath)
	fmt.Println("or directly run `bssh -H user:pass@192.168.1.30:8022`")

	_ = os.MkdirAll(filepath.Dir(confPath), 0755)
	_ = ioutil.WriteFile(confPath, initBsshToml, 0600)

	os.Exit(0)
}

func generateKey(tmplsNum, hostsNum, i int, j int) string {
	if tmplsNum > 1 {
		return fmt.Sprintf("host-%s-%s", pad(i+1, hostsNum), pad(j+1, tmplsNum))
	}

	return fmt.Sprintf("host-%s", pad(i+1, hostsNum))
}

func pad(i int, size int) string {
	return fmt.Sprintf(fmt.Sprintf("%%0%dd", len(strconv.Itoa(size))), i)
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
		keyString := strings.Join([]string{strconv.FormatInt(unixTime, 10), includePath}, "_")

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
		setCommon := ServerConfigDeduct(cf.Common, includeConf.Common)

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
		setValue := ServerConfigDeduct(setCommon, value)
		cf.Server[key] = setValue

		if value.Tmpl != "" {
			delete(cf.Server, key)

			tmplHosts := hostparse.Parse(setValue.Tmpl)
			tmplConfigs = append(tmplConfigs, tmplConfig{k: key, c: setValue, t: tmplHosts})
		}
	}

	cf.tmplServers(tmplConfigs)
}

// CheckFormatServerConf checkes format of server config.
//
// Note: Checking Addr, User and authentications
// having a value. No checking a validity of each fields.
//
// See also: checkFormatServerConfAuth function.
func CheckFormatServerConf(c Config) (isFormat bool) {
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

		if !CheckFormatServerConfAuth(v) {
			fmt.Printf("%s: Authentication information is not set.\n", k)

			isFormat = false
		}
	}

	return
}

// CheckFormatServerConfAuth checkes format of server config authentication.
//
// Note: Checking Pass, Key, Cert, AgentAuth, PKCS11Use, PKCS11Provider, Keys or
// Passes having a value. No checking a validity of each fields.
func CheckFormatServerConfAuth(c ServerConfig) (isFormat bool) {
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

// ServerConfigDeduct returns a new server config that set perConfig field to
// childConfig empty filed.
func ServerConfigDeduct(perConfig, childConfig ServerConfig) ServerConfig {
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

	sort.Strings(nameList)

	return nameList
}

// IsDirectServer tells that the server is a direct server address like user:pass@host:port.
func IsDirectServer(server string) bool {
	return strings.Index(server, "@") > 0
}

// ParseDirectServer parses a direct server address.
func ParseDirectServer(server string) (ServerConfig, bool) {
	// LastIndex of "@" will allow that password contains "@"
	atPos := strings.LastIndex(server, "@")
	sc := ServerConfig{}

	if atPos < 0 {
		return sc, false
	}

	left := server[:atPos]
	right := server[atPos+1:]

	sc.User, sc.Pass = splitBySep(left, []string{":", "/"})
	commaPos := strings.Index(right, ":")

	if commaPos == -1 {
		sc.Addr = right
		sc.Port = "22"
	} else {
		sc.Addr = right[:commaPos]
		sc.Port = right[commaPos+1:]
	}

	return sc, true
}

// EnsureSearchHost searches the host name by glob pattern.
func (cf *Config) EnsureSearchHost(host string) (string, []string) {
	if IsDirectServer(host) {
		return host, nil
	}

	matches1 := cf.globMatch(host)
	if len(matches1) == 1 {
		return matches1[0], nil
	}

	matches2 := cf.containsMatch(host)
	if len(matches2) == 1 {
		return matches2[0], nil
	}

	matches := make([]string, 0, len(matches1)+len(matches2))

	matches = append(matches, matches1...)
	matches = append(matches, matches2...)

	if len(matches) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "host %s not found from list.\n", host)
		return "", cf.GetNameSortedList()
	}

	_, _ = fmt.Fprintf(os.Stderr, "host %s found multiple hosts.\n", host)
	return "", matches
}

func (cf *Config) containsMatch(host string) []string {
	result1 := cf.matchesFn(host, func(host, serverName string, _ ServerConfig) bool {
		return strings.Contains(serverName, host)
	})
	if len(result1) == 1 {
		return result1
	}

	result2 := cf.matchesFn(host, func(host, _ string, v ServerConfig) bool {
		return strings.Contains(v.User+"@"+v.Addr+":"+v.Port, host)
	})
	if len(result2) == 1 {
		return result2
	}

	result3 := cf.matchesFn(host, func(host, _ string, v ServerConfig) bool {
		return strings.Contains(v.Note, host)
	})
	if len(result3) == 1 {
		return result3
	}

	matches := make([]string, 0, len(result1)+len(result2)+len(result3))

	matches = append(matches, result1...)
	matches = append(matches, result2...)
	matches = append(matches, result3...)

	return Unique(matches)
}

// Unique returns unique items in a slice
func Unique(slice []string) []string {
	us := make([]string, 0, len(slice))
	um := make(map[string]struct{})
	for _, v := range slice {
		if _, ok := um[v]; !ok {
			um[v] = struct{}{}
			us = append(us, v)
		}
	}

	return us
}

func (cf *Config) globMatch(host string) []string {
	result1 := cf.matchesFn(host, func(host, serverName string, _ ServerConfig) bool {
		ok, _ := filepath.Match(host, serverName)

		return ok
	})
	if len(result1) == 1 {
		return result1
	}

	result2 := cf.matchesFn(host, func(host, _ string, v ServerConfig) bool {
		ok, _ := filepath.Match(host, v.User+"@"+v.Addr+":"+v.Port)

		return ok
	})
	if len(result2) == 1 {
		return result2
	}

	result3 := cf.matchesFn(host, func(host, _ string, v ServerConfig) bool {
		ok, _ := filepath.Match(host, v.Note)

		return ok
	})
	if len(result3) == 1 {
		return result3
	}

	matches := make([]string, 0)

	matches = append(matches, result1...)
	matches = append(matches, result2...)
	matches = append(matches, result3...)

	return matches
}

func (cf *Config) matchesFn(host string, f func(host, serverName string, _ ServerConfig) bool) []string {
	matches := make([]string, 0)

	for k, v := range cf.Server {
		if f(host, k, v) {
			matches = append(matches, k)
		}
	}

	return matches
}

// GetNameSortedList return a list of server names from the Config structure.
func (cf *Config) GetNameSortedList() (nameList []string) {
	nameList = cf.GetNameList()
	sort.Strings(nameList)

	return nameList
}

func (cf *Config) IsDisableAutoEncryptPwd() bool {
	return cf.Extra.DisableAutoEncryptPwd || cf.DisableAutoEncryptPwd
}

func (cf *Config) loadTempHosts(confPath string) {
	tempHostsFile := strings.TrimSuffix(confPath, ".toml") + ".hosts"
	cf.tempHostsFile = tempHostsFile
	cf.tempHosts = make(map[string]bool)

	if !common.IsExist(tempHostsFile) {
		return
	}

	file, _ := ioutil.ReadFile(tempHostsFile)
	for _, line := range strings.Split(string(file), "\n") {
		hostLine := strings.TrimSpace(line)
		if hostLine != "" && !strings.HasPrefix(hostLine, "#") {
			if s, _ := pbe.Ebp(hostLine); s != "" {
				cf.tempHosts[hostLine] = true
			}
		}
	}

	for k := range cf.tempHosts {
		cf.Hosts = append(cf.Hosts, k)
	}
}

// WriteTempHosts writes a new host to temporary file.
func (cf *Config) WriteTempHosts(tempHost string) {
	if _, ok := cf.tempHosts[tempHost]; ok {
		return
	}

	cf.tempHosts[tempHost] = true

	s, _ := pbe.Pbe(tempHost)
	if err := AppendFile(cf.tempHostsFile, s); err != nil {
		fmt.Println(err)
	}
}

// PrintServerList prints server list which has names.
func (cf *Config) PrintServerList(names []string, printTitle bool) {
	if printTitle {
		_, _ = fmt.Fprintf(os.Stdout, "bssh Server List:\n")
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Server Name", "Connect Info", "Note"})

	for i, name := range names {
		v := cf.Server[name]
		t.AppendRow(table.Row{i + 1, name, v.User + "@" + v.Addr + ":" + v.Port, v.Note})
	}

	t.Render()
}

func AppendFile(file, line string) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := f.WriteString(line + "\n"); err != nil {
		return err
	}

	return nil
}
