module github.com/bingoohuang/bssh

go 1.22

replace (
	github.com/ThalesIgnite/crypto11 v1.2.5 => github.com/blacknon/crypto11 v1.2.6
	golang.org/x/crypto => github.com/goldstd/crypto v0.0.0-20240620011023-5817ff2c8f02
//golang.org/x/crypto => /Volumes/e2t/Github/crypto
)

require (
	github.com/BurntSushi/toml v1.4.0
	github.com/ScaleFT/sshkeys v1.2.0
	github.com/ThalesIgnite/crypto11 v1.2.5
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/bingoohuang/filestash v0.0.0-20240613081414-e66bed70f730
	github.com/bingoohuang/gg v0.0.0-20240813092226-31c91c0d930e
	github.com/bingoohuang/gonet v0.0.0-20230804022419-67aac8effd70
	github.com/bingoohuang/gossh v0.0.0-20240314074636-b579cd982160
	github.com/bingoohuang/gou v0.0.0-20210727012756-4873089fc9df
	github.com/bingoohuang/linuxdash v0.0.0-20210726093226-eb284e2777e1
	github.com/blacknon/textcol v0.0.1
	github.com/c-bata/go-prompt v0.2.6
	github.com/cheggaaa/pb/v3 v3.1.5
	github.com/dustin/go-humanize v1.0.1
	github.com/gorilla/mux v1.8.1
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/juju/ratelimit v1.0.2
	github.com/kevinburke/ssh_config v1.2.0
	github.com/lunixbochs/vtclean v1.0.0
	github.com/manifoldco/promptui v0.9.0
	github.com/mattn/go-runewidth v0.0.16
	github.com/mattn/go-shellwords v1.0.12
	github.com/miekg/pkcs11 v1.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/term v0.5.0
	github.com/nsf/termbox-go v1.1.1
	github.com/pkg/sftp v1.13.6
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.9.0
	github.com/thoas/go-funk v0.9.3
	github.com/urfave/cli v1.22.15
	github.com/vbauerster/mpb v3.4.0+incompatible
	go.uber.org/atomic v1.11.0
	golang.org/x/crypto v0.26.0
	golang.org/x/net v0.28.0
	golang.org/x/sys v0.24.0
	golang.org/x/term v0.23.0
	mvdan.cc/sh v2.6.4+incompatible
)

require (
	cloud.google.com/go/auth v0.8.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.4 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/averagesecurityguy/random v0.0.0-20210803154528-d84c3ae3b767 // indirect
	github.com/bingoohuang/gor v0.0.0-20230310012915-2ad15da4d290 // indirect
	github.com/bingoohuang/strcase v0.0.0-20200312105414-ac2c85cfc85d // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/creack/pty v1.1.23 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dchest/bcrypt_pbkdf v0.0.0-20150205184540-83f37f9c154a // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/howeyc/gopass v0.0.0-20210920133722-c8aef6fb66ef // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/mattn/go-tty v0.0.7 // indirect
	github.com/mickael-kerjean/net v0.0.0-20191120063050-2457c043ba06 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/term v1.2.0-beta.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/src-d/gcfg v1.4.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/thales-e-security/pool v0.0.2 // indirect
	github.com/tidwall/gjson v1.17.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go.mongodb.org/mongo-driver v1.16.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240808152545-0cdaa3abc0fa // indirect
	golang.org/x/oauth2 v0.22.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/api v0.192.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240812133136-8ffd90a71988 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.2 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
