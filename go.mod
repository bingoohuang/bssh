module github.com/bingoohuang/bssh

go 1.16

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/ScaleFT/sshkeys v0.0.0-20200327173127-6142f742bca5
	github.com/ThalesIgnite/crypto11 v1.2.4
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/bingoohuang/filestash v0.2.2-0.20210720070429-4806ed46ab4d
	github.com/bingoohuang/gonet v0.0.0-20200511075259-cef8ac6cd867
	github.com/bingoohuang/gossh v1.0.2-0.20210327012853-fda8e601e987
	github.com/bingoohuang/gou v0.0.0-20200714112627-3254bbe11221
	github.com/bingoohuang/linuxdash v0.0.0-20210720081020-5a35e261bd9c
	github.com/blacknon/go-sshlib v0.1.3
	github.com/blacknon/textcol v0.0.1
	github.com/c-bata/go-prompt v0.2.5
	github.com/chzyer/logex v1.1.10 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/chzyer/test v0.0.0-20180213035817-a1ea475d72b1 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/gorilla/mux v1.7.3
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/juju/ratelimit v1.0.1
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351
	github.com/mattn/go-runewidth v0.0.10
	github.com/mattn/go-shellwords v1.0.11
	github.com/miekg/pkcs11 v1.0.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nsf/termbox-go v0.0.0-20210114135735-d04385b850e8
	github.com/pkg/sftp v1.12.0
	github.com/sirupsen/logrus v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.7.0
	github.com/urfave/cli v1.22.5
	github.com/vbauerster/mpb v3.4.0+incompatible
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
	mvdan.cc/sh v2.6.4+incompatible
)

replace github.com/miekg/pkcs11 => github.com/blacknon/pkcs11 v1.0.4-0.20201018135904-6038e308f617
