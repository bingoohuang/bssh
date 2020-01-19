[![TravisCI](https://travis-ci.org/bingoohuang/bssh.svg?branch=master)](https://travis-ci.org/bingoohuang/bssh)
[![Go Report Card](https://goreportcard.com/badge/github.com/bingoohuang/bssh)](https://goreportcard.com/report/github.com/bingoohuang/bssh)

# bssh

TUI list select ssh/scp/sftp client tools.

## Description

command to read a prepared list in advance and connect ssh/scp/sftp the selected host. List file is set in yaml format. When selecting a host, you can filter by keywords. Can execute commands concurrently to multiple hosts. Supported multiple ssh proxy, http/socks5 proxy, x11 forward, and port forwarding.

## Features

* Config Templating, grouping, passwords PEB encryption and direct last sub-command to reuse last chosen servers.
* List selection type ssh client.
* Pure Go.
* Commands can be executed by ssh connection in parallel.
* Supported ssh multiple proxy, http/socks5 proxy.
* Supported ssh-agent.
* Supported Port forward, x11 forward.
* Can use bashrc of local machine at ssh connection destination.

## Demo

## Install

### compile

compile go file(tested go1.12.4).

    go get -u github.com/bingoohuang/bssh/cmd/bssh

    # copy sample config. create `~/.bssh.toml`.
    test -f ~/.bssh.toml||curl -s https://raw.githubusercontent.com/bingoohuang/bssh/master/example/config.toml -o ~/.bssh.toml

or

    git clone https://github.com/bingoohuang/bssh
    cd bssh
    make && sudo make install

    # copy sample config. create `~/.bssh.toml`.
    test -f ~/.bssh.toml||curl -s https://raw.githubusercontent.com/bingoohuang/bssh/master/example/config.toml -o ~/.bssh.toml

## Config

Please edit "~/.bssh.toml".\
For details see [Config](doc/Config.md).

## Usage

### bssh

run command.

    bssh


option(bssh)

	NAME:
	    bssh - TUI list select and parallel ssh client command.
	USAGE:
	    bssh [options] [commands...]
	
	OPTIONS:
	    --host servername, -H servername            connect servername.
	    --file filepath, -F filepath                config filepath. (default: "/Users/blacknon/.bssh.toml")
	    -L [bind_address:]port:remote_address:port  Local port forward mode.Specify a [bind_address:]port:remote_address:port.
	    -R [bind_address:]port:remote_address:port  Remote port forward mode.Specify a [bind_address:]port:remote_address:port.
	    -D port                                     Dynamic port forward mode(Socks5). Specify a port.
	    -w                                          Displays the server header when in command execution mode.
	    -W                                          Not displays the server header when in command execution mode.
	    --not-execute, -N                           not execute remote command and shell.
	    --x11, -X                                   x11 forwarding(forward to ${DISPLAY}).
	    --term, -t                                  run specified command at terminal.
	    --parallel, -p                              run command parallel node(tail -F etc...).
	    --localrc                                   use local bashrc shell.
	    --not-localrc                               not use local bashrc shell.
	    --pshell, -s                                use parallel-shell(pshell) (alpha).
	    --list, -l                                  print server list from config.
	    --help, -h                                  print this help
	    --version, -v                               print the version
	
	COPYRIGHT:
	    blacknon(blacknon@orebibou.com)
	
	VERSION:
	    0.6.0
	
	USAGE:
	    # connect ssh
	    bssh
	
	    # parallel run command in select server over ssh
	    bssh -p command...
	
	    # parallel run command in select server over ssh, do it interactively.
	    bssh -s


### bssh scp

run command.

    bssh scp from... to

option(lscp)
	
	NAME:
	    lscp scp - TUI list select and parallel scp client command.
	USAGE:
	    lscp scp [options] (local|remote):from_path... (local|remote):to_path
	
	OPTIONS:
	    --host value, -H value  connect servernames
	    --list, -l              print server list from config
	    --file value, -F value  config file path (default: "/Users/blacknon/.bssh.toml")
	    --permission, -p        copy file permission
	    --help, -h              print this help
	    --version, -v           print the version
	
	COPYRIGHT:
	    blacknon(blacknon@orebibou.com)
	
	VERSION:
	    0.6.0
	
	USAGE:
	    # local to remote scp
	    bssh scp /path/to/local... remote:/path/to/remote
	
	    # remote to local scp
	    bssh scp remote:/path/to/remote... /path/to/local
	
	    # remote to remote scp
	    bssh scp remote:/path/to/remote... remote:/path/to/local


### bssh ftp

run command.

    bssh ftp

option(bssh ftp)

	NAME:
	    bssh ftp - TUI list select and parallel sftp client command.
	USAGE:
	    bssh ftp [options]
	
	OPTIONS:
	    --file value, -F value  config file path (default: "/Users/blacknon/.bssh.toml")
	    --help, -h              print this help
	    --version, -v           print the version
	
	COPYRIGHT:
	    blacknon(blacknon@orebibou.com)
	
	VERSION:
	    0.6.0
	
	USAGE:
	  # start bssh ftp shell
	  bssh ftp


If you specify a command as an argument, you can select multiple hosts. Select host <kbd>Tab</kbd>, select all displayed hosts <kbd>Ctrl</kbd> + <kbd>a</kbd>.


### 1. [bssh] connect terminal
<details>

You can connect to the terminal like a normal ssh command (OpenSSH).

<p align="center">
<img src="./images/1-1.gif" />
</p>


You can connect using a local bashrc file (if ssh login shell is bash).

<p align="center">
<img src="./images/1-2.gif" />
</p>

`~/.bssh.toml` example.

    [server.localrc]
	addr = "192.168.100.104"
	key  = "/path/to/private_key"
	note = "Use local bashrc files."
	local_rc = 'yes'
	local_rc_file = [
         "~/dotfiles/.bashrc"
        ,"~/dotfiles/bash_prompt"
        ,"~/dotfiles/sh_alias"
        ,"~/dotfiles/sh_export"
        ,"~/dotfiles/sh_function"
	]


You can execute commands before and after ssh connection.\
You can also change the color of each host's terminal by combining it with the OSC escape sequence.

if iTerm2, you can also change the profile.



`~/.bssh.toml` example.

    [server.iTerm2_sample]
	addr = "192.168.100.103"
	key  = "/path/to/private_key"
	note = "Before/After run local command"
	pre_cmd = 'printf "\033]50;SetProfile=Theme\a"'    # ssh theme
    post_cmd = 'printf "\033]50;SetProfile=Default\a"' # local theme
	note = "(option) exec command after ssh disconnected."

    [server.GnomeTerminal_sample]
	addr = "192.168.100.103"
	key  = "/path/to/private_key"
	note = "Before/After run local command"
	pre_cmd = 'printf "\e]10;#ffffff\a\e]11;#503000\a"'  # ssh color
    post_cmd = 'printf "\e]10;#ffffff\a\e]11;#000000\a"' # local color
	note = "(option) exec command after ssh disconnected."


A terminal log can be recorded by writing a configuration file.

`~/.bssh.toml` example.

	[log]
	enable = true
	timestamp = true
	dirpath = "~/log/bssh/<Date>/<Hostname>"


There are other parameters corresponding to ClientAliveInterval and ClientAliveCountMax.

    [server.alivecount]
	addr = "192.168.100.101"
	key  = "/path/to/private_key"
	note = "alive count max."
	alive_max = 3 # ServerAliveCountMax
	alive_interval = 60 # ServerAliveCountInterval


</details>

### 2. [bssh] run command (parallel)
<details>

It is possible to execute by specifying command in argument.\
Parallel execution can be performed by adding the `-p` option.


	# exec command over ssh.
	bssh <command...>

	# exec command over ssh, parallel.
	bssh -p <command>


In parallel connection mode (`-p` option), Stdin can be sent to each host.\

<p align="center">
<img src="./images/2-2.gif" />
</p>


Can be piped to send Stdin.

	# You can pass values ​​in a pipe
	command... | bssh <command...>


</details>

### 3. [bssh] Execute commands interactively (parallel shell)
<details>

You can send commands to multiple servers interactively.

	# parallel shell connect
	bssh -s


You can also combine remote and local commands.

	remote_command | !local_command


</details>

### 4. [bssh] scp (local=>remote(multi), remote(multi)=>local, remote=>remote(multi))
<details>

You can do scp by selecting a list with the command lscp.\
You can select multiple connection destinations. This program use sftp protocol.


`local => remote(multiple)`

    # lscp local => remote(multiple)
    lscp /path/to/local... r:/path/to/remote


`remote(multiple) => local`

    # lscp remote(multiple) => local
    lscp r:/path/to/remote... /path/to/local


`remote => remote(multiple)`

    # lscp remote => remote(multiple)
    lscp r:/path/to/remote... r:/path/to/local


</details>

### 5. [bssh ftp] sftp (local=>remote(multi), remote(multi)=>local)
<details>

You can do sftp by selecting a list with the command lstp.\
You can select multiple connection destinations.


`bssh ftp`


</details>


### 5. include ~/.ssh/config file.
<details>

Load and use `~/.ssh/config` by default.\
`ProxyCommand` can also be used.

Alternatively, you can specify and read the path as follows: In addition to the path, ServerConfig items can be specified and applied collectively.

	[sshconfig.default]
	path = "~/.ssh/config"
	pre_cmd = 'printf "\033]50;SetProfile=local\a"'
	post_cmd = 'printf "\033]50;SetProfile=Default\a"'

</details>

### 6. include other ServerConfig file.
<details>

You can include server settings in another file.\
`common` settings can be specified for each file that you went out.

`~/.bssh.toml` example.

	[includes]
	path = [
    	 "~/.bssh.d/home.conf"
    	,"~/.bssh.d/cloud.conf"
	]

`~/.bssh.d/home.conf` example.

	[common]
	pre_cmd = 'printf "\033]50;SetProfile=dq\a"'       # iterm2 ssh theme
	post_cmd = 'printf "\033]50;SetProfile=Default\a"' # iterm2 local theme
	ssh_agent_key = ["~/.ssh/id_rsa"]
	ssh_agent = false
	user = "user"
	key = "~/.ssh/id_rsa"
	pkcs11provider = "/usr/local/lib/opensc-pkcs11.so"
	
	[server.Server1]
	addr = "172.16.200.1"
	note = "TEST Server1"
	local_rc = "yes"
	
	[server.Server2]
	addr = "172.16.200.2"
	note = "TEST Server2"
	local_rc = "yes"

The priority of setting values ​​is as follows.

`[server.hogehoge]` > `[common] at Include file` > `[common] at ~/.bssh.toml`


</details>

### 7. Supported Proxy
<details>

Supports multiple proxy.

* http
* socks5
* ssh

Besides this, you can also specify ProxyCommand like OpenSSH.

`http` proxy example.

	[proxy.HttpProxy]
	addr = "example.com"
	port = "8080"

	[server.overHttpProxy]
	addr = "over-http-proxy.com"
	key  = "/path/to/private_key"
	note = "connect use http proxy"
	proxy = "HttpProxy"
	proxy_type = "http"


`socks5` proxy example.

	[proxy.Socks5Proxy]
	addr = "example.com"
	port = "54321"

	[server.overSocks5Proxy]
	addr = "192.168.10.101"
	key  = "/path/to/private_key"
	note = "connect use socks5 proxy"
	proxy = "Socks5Proxy"
	proxy_type = "socks5"


`ssh` proxy example.

	[server.sshProxyServer]
	addr = "192.168.100.200"
	key  = "/path/to/private_key"
	note = "proxy server"
	
	[server.overProxyServer]
	addr = "192.168.10.10"
	key  = "/path/to/private_key"
	note = "connect use ssh proxy"
	proxy = "sshProxyServer"
	
	[server.overProxyServer2]
	addr = "192.168.10.100"
	key  = "/path/to/private_key"
	note = "connect use ssh proxy(multiple)"
	proxy = "overProxyServer"


`ProxyCommand` proxy example.

	[server.ProxyCommand]
	addr = "192.168.10.20"
	key  = "/path/to/private_key"
	note = "connect use ssh proxy(multiple)"
	proxy_cmd = "ssh -W %h:%p proxy"


</details>


### 8. Available authentication method
<details>

* Password auth
* Publickey auth
* Certificate auth
* PKCS11 auth
* Ssh-Agent auth

`password` auth example.

	[server.PasswordAuth]
	addr = "password_auth.local"
	user = "user"
	pass = "Password"
	note = "password auth server"


`publickey` auth example.

	[server.PublicKeyAuth]
	addr = "pubkey_auth.local"
	user = "user"
	key = "~/path/to/key"
	note = "Public key auth server"

	[server.PublicKeyAuth_with_passwd]
	addr = "password_auth.local"
	user = "user"
	key = "~/path/to/key"
	keypass = "passphrase"
	note = "Public key auth server with passphrase"


`cert` auth example.\
(pkcs11 key is not supported in the current version.)

	[server.CertAuth]
	addr = "cert_auth.local"
	user = "user"
	cert = "~/path/to/cert"
	certkey = "~/path/to/certkey"
	note = "Certificate auth server"

	[server.CertAuth_with_passwd]
	addr = "cert_auth.local"
	user = "user"
	cert = "~/path/to/cert"
	certkey = "~/path/to/certkey"
	certkeypass = "passphrase"
	note = "Certificate auth server with passphrase"


`pkcs11` auth example.

	[server.PKCS11Auth]
	addr = "pkcs11_auth.local"
	user = "user"
	pkcs11provider = "/usr/local/lib/opensc-pkcs11.so"
	pkcs11 = true
	note = "PKCS11 auth server"

	[server.PKCS11Auth_with_PIN]
	addr = "pkcs11_auth.local"
	user = "user"
	pkcs11provider = "/usr/local/lib/opensc-pkcs11.so"
	pkcs11 = true
	pkcs11pin = "123456"
	note = "PKCS11 auth server"


`ssh-agent` auth example.

	[server.SshAgentAuth]
	addr = "agent_auth.local"
	user = "user"
	agentauth = true # auth ssh-agent
	note = "ssh-agent auth server"

</details>


### 9. Port forwarding

<details>

Supported Local/Remote/Dynamic port forwarding.\
You can specify from the command line or from the configuration file.

#### command line option

    bssh -L 8080:localhost:80 # local port forwarding
    bssh -R 80:localhost:8080 # remote port forwarding
    bssh -D 10080             # dynamic port forwarding


#### config file

	[server.LocalPortForward]
	addr = "localforward.local"
	user = "user"
	agentauth = true
	port_forward_local = "localhost:8080"
	port_forward_remote = "localhost:80"
	note = "local port forwawrd example"

	[server.RemotePortForward]
	addr = "remoteforward.local"
	user = "user"
	agentauth = true
	port_forward = "REMOTE"
	port_forward_local = "localhost:80"
	port_forward_remote = "localhost:8080"
	note = "remote port forwawrd example"

If OpenSsh config is loaded, it will be loaded as it is.


</details>


## Licence

A short snippet describing the license [MIT](https://github.com/bingoohuang/bssh/blob/master/LICENSE.md).

## Author

[blacknon](https://github.com/blacknon)
