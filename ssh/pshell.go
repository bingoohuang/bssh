package ssh

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bingoohuang/bssh/conf"
	"github.com/bingoohuang/bssh/output"
	"github.com/bingoohuang/bssh/sshlib"
	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

// TDXX(blacknon): 接続が切れた場合の再接続処理、および再接続ができなかった場合のsliceからの削除対応の追加(v0.6.1)
// TDXX(blacknon): pShellのログ(実行コマンド及び出力結果)をログとしてファイルに記録する機能の追加(v0.6.1)
// TDXX(blacknon): グループ化(`()`で囲んだりする)や三項演算子への対応(v0.6.1)
// TDXX(blacknon): `サーバ名:command...` で、指定したサーバでのみコマンドを実行させる機能の追加(v0.6.1)

// Pshell is Parallel-Shell struct.
type pShell struct {
	Signal        chan os.Signal
	Count         int
	ServerList    []string
	Connects      []*psConnect
	PROMPT        string
	History       map[int]map[string]*pShellHistory
	HistoryFile   string
	latestCommand string
	CmdComplete   []prompt.Suggest
	PathComplete  []prompt.Suggest
	Options       pShellOption
}

// pShellOption is optitons pshell.
type pShellOption struct {
	// local command実行時の結果をHistoryResultに記録しない(os.Stdoutに直接出す)
	LocalCommandNotRecordResult bool

	// trueの場合、リモートマシンでパイプライン処理をする際にパイプ経由でもOPROMPTを付与して出力する
	// RemoteHeaderWithPipe bool

	// trueの場合、リモートマシンにキーインプットを送信しない
	// hogehoge

	// trueの場合、コマンドの補完処理を無効にする
	// DisableCommandComplete bool

	// trueの場合、PATHの補完処理を無効にする
	// DisableCommandComplete bool
}

// psConnect is pShell connect struct.
type psConnect struct {
	Name   string
	Output *output.Output
	*sshlib.Connect
}

const (
	// Default PROMPT.
	defaultPrompt = "[${COUNT}] <<< "

	// Default OPROMPT.
	defaultOPrompt = "[${SERVER}][${COUNT}] > "

	// Default Parallel shell history file.
	defaultHistoryFile = "~/.lssh_history"
)

func (r *Run) pshell() (err error) {
	// print header
	fmt.Println("Start parallel-shell...")
	r.PrintSelectServer()

	// read shell config
	config := r.Conf.Shell

	// overwrite default value config.Prompt
	if config.Prompt == "" {
		config.Prompt = defaultPrompt
	}

	// overwrite default value config.OPrompt
	if config.OPrompt == "" {
		config.OPrompt = defaultOPrompt
	}

	// overwrite default parallel shell history file
	if config.HistoryFile == "" {
		config.HistoryFile = defaultHistoryFile
	}

	// run pre cmd
	execLocalCommand(config.PreCmd)
	defer execLocalCommand(config.PostCmd)

	cons := r.createPsConnects(config)
	if len(cons) == 0 {
		return
	}

	// create new shell struct
	ps := &pShell{
		Signal:      make(chan os.Signal),
		ServerList:  r.ServerList,
		Connects:    cons,
		PROMPT:      config.Prompt,
		History:     map[int]map[string]*pShellHistory{},
		HistoryFile: config.HistoryFile,
	}

	// set signal
	signal.Notify(ps.Signal, syscall.SIGTERM, syscall.SIGINT)

	// old history list
	var historyCommand []string

	oldHistory, err := ps.GetHistoryFromFile()

	if err == nil {
		for _, h := range oldHistory {
			historyCommand = append(historyCommand, h.Command)
		}
	}

	// create complete data
	// TDXX(blacknon): 定期的に裏で取得するよう処理を加える(v0.6.1)
	ps.GetCommandComplete()

	// create go-prompt
	p := prompt.New(
		ps.Executor,
		ps.Completer,
		prompt.OptionHistory(historyCommand),
		prompt.OptionLivePrefix(ps.CreatePrompt),
		prompt.OptionInputTextColor(prompt.Green),
		prompt.OptionPrefixTextColor(prompt.Blue),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator), // test
	)

	// start go-prompt
	p.Run()

	return nil
}

func (r *Run) createPsConnects(config conf.ShellConfig) []*psConnect {
	// Connect
	cons := make([]*psConnect, len(r.ServerList))

	for i, server := range r.ServerList {
		con, err := r.CreateSSHConnect(nil, server)
		if err != nil {
			log.Println(err)
			continue
		}

		// TTY enable
		con.TTY = true

		// Create Output
		o := &output.Output{
			Templete:   config.OPrompt,
			ServerList: r.ServerList,
			Conf:       r.Conf.Server[server],
			AutoColor:  true,
		}

		// Create output prompt
		o.Create(server)

		cons[i] = &psConnect{Name: server, Output: o, Connect: con}
	}

	return cons
}

// CreatePrompt is create shell prompt.
// default value is `[${COUNT}] <<< `.
func (ps *pShell) CreatePrompt() (p string, result bool) {
	// set prompt template (from conf)
	p = ps.PROMPT
	if p == "" {
		p = defaultPrompt
	}

	// Get env
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	pwd := os.Getenv("PWD")

	// replace variable value
	p = strings.Replace(p, "${COUNT}", strconv.Itoa(ps.Count), -1)
	p = strings.Replace(p, "${HOSTNAME}", hostname, -1)
	p = strings.Replace(p, "${USER}", username, -1)
	p = strings.Replace(p, "${PWD}", pwd, -1)

	return p, true
}
