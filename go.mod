module github.com/bingoohuang/bssh

go 1.16

replace github.com/miekg/pkcs11 => github.com/blacknon/pkcs11 v1.0.4-0.20201018135904-6038e308f617

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/ScaleFT/sshkeys v0.0.0-20200327173127-6142f742bca5
	github.com/ThalesIgnite/crypto11 v1.2.4
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/bingoohuang/filestash v0.0.0-20210826063619-eaa9271da225
	github.com/bingoohuang/gg v0.0.0-20211227102539-8a437d3525d1
	github.com/bingoohuang/gonet v0.0.0-20200511075259-cef8ac6cd867
	github.com/bingoohuang/gossh v0.0.0-20220124024046-40e25ed6f93a
	github.com/bingoohuang/gou v0.0.0-20210727012756-4873089fc9df
	github.com/bingoohuang/linuxdash v0.0.0-20210726093226-eb284e2777e1
	github.com/blacknon/go-sshlib v0.1.3
	github.com/blacknon/textcol v0.0.1
	github.com/c-bata/go-prompt v0.2.6
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/dustin/go-humanize v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/juju/ratelimit v1.0.1
	github.com/kevinburke/ssh_config v1.1.0
	github.com/mattn/go-runewidth v0.0.12
	github.com/mattn/go-shellwords v1.0.12
	github.com/miekg/pkcs11 v1.0.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nsf/termbox-go v1.1.1
	github.com/pkg/sftp v1.13.4
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.9.0
	github.com/urfave/cli v1.22.5
	github.com/vbauerster/mpb v3.4.0+incompatible
	go.mongodb.org/mongo-driver v1.5.1 // indirect
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	mvdan.cc/sh v2.6.4+incompatible
)
