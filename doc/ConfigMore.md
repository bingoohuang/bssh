### configure multiple 

.lssh.conf

```
[server.ServerName]
tmpl = "192.168.0.(101-103):22 user/password"       # 101,102,103 server address
note = "this is a test. password auth"              # note text
```

the above conf will be expanded to the followings：

```
[server.ServerName-1]
addr = "192.168.0.101"                 # server address
port = "22"                            # port number(Default:22)
user = "user"                          # connect user
pass = "password"                      # connect password
note = "this is a test. password auth" # note text

[server.ServerName-2]
addr = "192.168.0.102"                 # server address
port = "22"                            # port number(Default:22)
user = "user"                          # connect user
pass = "password"                      # connect password
note = "this is a test. password auth" # note text

[server.ServerName-3]
addr = "192.168.0.103"                 # server address
port = "22"                            # port number(Default:22)
user = "user"                          # connect user
pass = "password"                      # connect password
note = "this is a test. password auth" # note text
```

### grouping servers (when there is more than ~20 servers, grouping is a good chosen)

.lssh.conf

```
[server.zonea]
grouping = ["zonea"]
tmpl = "192.168.0.(101-103):22 user/password"       # 101,102,103 server address
note = "this is a test. password auth"              # note text

[server.zonea]
grouping = ["zoneb"]
tmpl = "192.168.0.(101-103):22 user/password"       # 101,102,103 server address
note = "this is a test. password auth"              # note text
```

the above conf has more than one groups (zonea and zoneb), the lssh will show the group list first,
after the grouping selected, the narrowed server list in the selected grouping will show then.