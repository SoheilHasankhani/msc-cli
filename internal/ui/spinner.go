package ui

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type spinModel struct {
	spinner spinner.Model
	label   string
	done    bool
	err     error
}

type spinDoneMsg struct{ err error }

func (m spinModel) Init() tea.Cmd { return m.spinner.Tick }

func (m spinModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinDoneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s…", m.spinner.View(), m.label)
}

// WithSpinner runs fn under a Charm spinner on TTY; otherwise prints a plain line.
func WithSpinner(ctx context.Context, w io.Writer, label string, fn func(context.Context) error) error {
	if w == nil {
		w = os.Stderr
	}
	tty := false
	if f, ok := w.(*os.File); ok {
		tty = term.IsTerminal(int(f.Fd())) && os.Getenv("NO_COLOR") == ""
	}
	if !tty {
		fmt.Fprintf(w, "%s…\n", label)
		return fn(ctx)
	}

	m := spinModel{
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		label:   label,
	}
	p := tea.NewProgram(m, tea.WithOutput(w), tea.WithInput(nil), tea.WithoutSignalHandler())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = p.Run()
	}()
	err := fn(ctx)
	p.Send(spinDoneMsg{err: err})
	<-done
	return err
}
