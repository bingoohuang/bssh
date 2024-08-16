package gum

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/gum/cursor"
	"github.com/charmbracelet/gum/style"
	"github.com/charmbracelet/gum/timeout"
	"github.com/charmbracelet/lipgloss"
)

// InputOptions are the customization options for the input.
type InputOptions struct {
	Placeholder string        `help:"Placeholder value" default:"Type something..." env:"GUM_INPUT_PLACEHOLDER"`
	Prompt      string        `help:"Prompt to display" default:"> " env:"GUM_INPUT_PROMPT"`
	PromptStyle style.Styles  `embed:"" prefix:"prompt." envprefix:"GUM_INPUT_PROMPT_"`
	CursorStyle style.Styles  `embed:"" prefix:"cursor." set:"defaultForeground=212" envprefix:"GUM_INPUT_CURSOR_"`
	CursorMode  string        `prefix:"cursor." name:"mode" help:"Cursor mode" default:"blink" enum:"blink,hide,static" env:"GUM_INPUT_CURSOR_MODE"`
	Value       string        `help:"Initial value (can also be passed via stdin)" default:""`
	CharLimit   int           `help:"Maximum value length (0 for no limit)" default:"400"`
	Width       int           `help:"Input width (0 for terminal width)" default:"40" env:"GUM_INPUT_WIDTH"`
	Password    bool          `help:"Mask input characters" default:"false"`
	Header      string        `help:"Header value" default:"" env:"GUM_INPUT_HEADER"`
	HeaderStyle style.Styles  `embed:"" prefix:"header." set:"defaultForeground=240" envprefix:"GUM_INPUT_HEADER_"`
	Timeout     time.Duration `help:"Timeout until input aborts" default:"0" env:"GUM_INPUT_TIMEOUT"`
}

// Run provides a shell script interface for the text input bubble.
// https://github.com/charmbracelet/bubbles/textinput
func (o InputOptions) Run() (string, error) {
	i := textinput.New()
	if o.Value != "" {
		i.SetValue(o.Value)
	} else if in, _ := Read(); in != "" {
		i.SetValue(in)
	}

	i.Focus()
	i.Prompt = o.Prompt
	i.Placeholder = o.Placeholder
	i.Width = o.Width
	i.PromptStyle = o.PromptStyle.ToLipgloss()
	i.Cursor.Style = o.CursorStyle.ToLipgloss()
	i.Cursor.SetMode(cursor.Modes[o.CursorMode])
	i.CharLimit = o.CharLimit

	if o.Password {
		i.EchoMode = textinput.EchoPassword
		i.EchoCharacter = '•'
	}

	p := tea.NewProgram(model{
		textinput:   i,
		aborted:     false,
		header:      o.Header,
		headerStyle: o.HeaderStyle.ToLipgloss(),
		timeout:     o.Timeout,
		hasTimeout:  o.Timeout > 0,
		autoWidth:   o.Width < 1,
	}, tea.WithOutput(os.Stderr))
	tm, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run input: %w", err)
	}
	m := tm.(model)

	if m.aborted {
		return "", ErrAborted
	}

	return m.textinput.Value(), nil
}

type model struct {
	autoWidth   bool
	header      string
	headerStyle lipgloss.Style
	textinput   textinput.Model
	quitting    bool
	aborted     bool
	timeout     time.Duration
	hasTimeout  bool
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		timeout.Init(m.timeout, nil),
	)
}
func (m model) View() string {
	if m.quitting {
		return ""
	}
	if m.header != "" {
		header := m.headerStyle.Render(m.header)
		return lipgloss.JoinVertical(lipgloss.Left, header, m.textinput.View())
	}

	return m.textinput.View()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timeout.TickTimeoutMsg:
		if msg.TimeoutValue <= 0 {
			m.quitting = true
			m.aborted = true
			return m, tea.Quit
		}
		m.timeout = msg.TimeoutValue
		return m, timeout.Tick(msg.TimeoutValue, msg.Data)
	case tea.WindowSizeMsg:
		if m.autoWidth {
			m.textinput.Width = msg.Width - lipgloss.Width(m.textinput.Prompt) - 1
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			m.aborted = true
			return m, tea.Quit
		case "enter":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	return m, cmd
}
