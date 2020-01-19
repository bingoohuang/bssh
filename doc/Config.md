The default PATH of the configuration file is ~ /.bssh.toml. This is a sample at configuration file.

### Basic Setting

#### .bssh.toml

```
[server.ServerName1]
addr = "192.168.0.101"                 # server address
port = "22"                            # port number(Default:22)
user = "user"                          # connect user
pass = "password"                      # connect password
note = "this is a test. password auth" # note text

[server.ServerName2]
addr = "192.168.0.102"                 # server address
user = "user"                          # connect user
key = "/path/to/key"                   # connect private key path
note = "this is a test. key auth"      # note text

[server.ServerName3]
addr = "192.168.0.103"                 # server address
user = "user"                          # connect user
key = "/path/to/key"                   # connect private key path
keypass = "password"                   # private key passphrase
note = "this is a test. key auth"      # note text
```

### Common settings

You can set common server settings. You can set common settings for each include file (overwrite the entire common setting with include file).

```
[common]
port = "10022"
user = "user"

[server.ServerName1] # user port=10022,user=user
addr = "192.168.0.101"
user = "password"
note = "this is a test. password auth"

[server.ServerName2] # user port=10022,user=user
addr = "192.168.0.102"
key = "/path/to/key"
note = "this is a test. key auth"

[server.ServerName3]
addr = "192.168.0.103"
port = "22"                            # overwrite common port number
user = "test"                          # overwrite common user
key = "/path/to/key"
keypass = "password"
note = "this is a test. key auth"
```

### Include server config file

Include config file settings and path. (only common,server config)

#### .bssh.toml

```
# [include.include1] # When writing individually
# path = "~/.bssh.toml.include1"

[includes]
path = [
     "~/.bssh.toml.include1"
    ,"~/.bssh.toml.include2"
]
```

#### .bssh.toml.include1

```
[server.ServerName1]
addr = "192.168.0.101"
port = "22"
user = "user"
user = "password"
note = "this is a test. password auth"
```

### Logging terminal log

You can record the terminal log. The following variables can be specified in the log file path directory. Log file name is in the format "YYYYmmdd_HHMMss_ServerName.log".

* \<Date\> ... YYYYMMDD
* \<Hostname\> ... ServerName

#### .bssh.toml
```
[log]
enable = true       # bool logging
timestamp = true    # add timestamp line head
dirpath = "/path/to/<Date>_<Hostname>/logdir"  
```

### [ssh,http,socks5] Proxy server settings

You can connect via http, socks 5, ssh proxy. Supported multiple proxy. (html, socks5 only 1st proxy).

#### .bssh.toml

```
[server.sshProxyServer]
addr = "192.168.100.200"
key  = "/path/to/private_key"
note = "proxy server"

[server.overProxyServer]         # via "sshProxyServer"
addr = "192.168.10.10"
key  = "/path/to/private_key"
note = "connect use ssh proxy"
proxy = "sshProxyServer"

[server.overProxyServer2]        # via "sshProxyServer" > "overProxyServer"
addr = "192.168.10.100"
key  = "/path/to/private_key"
note = "connect use ssh proxy(multiple)"
proxy = "overProxyServer"

[server.overHttpProxy]           # via "HttpProxy"
addr = "over-http-proxy.com"
key  = "/path/to/private_key"
note = "connect use http proxy"
proxy = "HttpProxy"
proxy_type = "http"

[server.overSocks5Proxy]         # via Socks5Proxy
addr = "192.168.10.101"
key  = "/path/to/private_key"
note = "connect use socks5 proxy"
proxy = "Socks5Proxy"
proxy_type = "socks5"

[proxy.HttpProxy]
addr = "example.com"
port = "8080"

[proxy.Socks5Proxy]
addr = "example.com"
port = "54321"
```

### Exec command ssh {pre,post} connect

Commands specified before and after ssh connection can be executed locally.

```
[server.LocalCommand_ServerName]
addr = "192.168.100.103"
key  = "/path/to/private_key"
note = "Before/After run local command"
pre_cmd = "(option) exec command before ssh connect."
post_cmd = "(option) exec command after ssh disconnected."
```

### Use local bashrc file (v0.5.1-)

If bash is used as the login shell of the ssh connection user, you can connect using the local bashrc file.

```
[server.UseLocalBashrc_ServerName]
addr = "192.168.100.104"
key  = "/path/to/private_key"
note = "Use local bashrc files."

# bool use localrc
local_rc = 'yes'
# bashrc files (array)
local_rc_file = [
     "~/dotfiles/.bashrc"
    ,"~/dotfiles/bash_prompt"
    ,"~/dotfiles/sh_alias"
    ,"~/dotfiles/sh_export"
    ,"~/dotfiles/sh_function"
]
```

### Use Ssh Agent (v0.5.2-)

You can use ssh-agent.

```
[server.UseSshAgent]
addr = "192.168.0.111"
port = "22"
user = "user"
pass = "password"
note = "use ssh agent"

# bool ssh-agent
ssh_agent = true
ssh_agent_key = [
     "~/.ssh/id_rsa"
    ,"~/.ssh/encrypt_key::passphase"
]
```

### Use PKCS11 Auth (v0.5.3-)

You can use PKCS11 authentication by specifying PATH of libraries such as OpenSC.

```
[server.UsePKCS11Auth]
addr = "192.168.0.101"
port = "22"
user = "user"
pkcs11provider = "/usr/local/lib/opensc-pkcs11.so"
pkcs11pin = "123456" # option
pkcs11 = true # pkcs11 use flag
note = "use pkcs11 auth"
```

### Use Cert Auth (v0.5.4-)

You can use Cert authentication.

```
[server.UseCertAuth]
addr = "192.168.0.101"
port = "22"
user = "user"
cert = "/path/to/cert"
certkey = "/path/to/cert-key"
# certkeypass = "certkey-password" 
note = "use cert auth"
```


### Set port forwarding (v0.5.2-)

You can configure port-forwarding. It will be overwritten if specified by the command option.

```
[server.UsePosrForwarding]
addr = "192.168.0.112"
port = "22"
user = "user"
pass = "password"
note = "use port forwarding"
port_forward_local = "localhost:8080"
port_forward_remote = "localhost:80"
```

### (Sample) Change terminal profile(or terminal background,front color)

In a typical terminal emulator, you can change the terminal background color and text color using the OSC escape sequence. iTerm 2 can also specify a profile.

```
[server.ChangeTerminalColor1]
addr = "192.168.0.101"
port = "22"
user = "user"
pass = "password"
note = "Change terminal color (Gnome Terminal, Terminator etc...)"
pre_cmd = 'echo -ne "\e]10;#ffffff\a\e]11;#003000\a"#background: green, character: white'
post_cmd = 'echo -ne "\e]10;#ffffff\a\e]11;#000000\a"#background: black, character: white'

[server.ChangeTerminalColor2]
addr = "192.168.0.101"
port = "22"
user = "user"
pass = "password"
note = "Change terminal color (iTerm2)"
pre_cmd = 'printf "\033]50;SetProfile=SshProfile\a"' # ssh theme
post_cmd = 'printf "\033]50;SetProfile=Default\a"'   # local theme
```