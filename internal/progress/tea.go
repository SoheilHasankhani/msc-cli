package progress

import (
	"context"
	"fmt"
	"strings"

	cprogress "github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type updateMsg Update

type quitMsg struct{ err error }

type teaModel struct {
	inner       Model
	bars        map[string]cprogress.Model
	width       int
	quitting    bool
	err         error
	cancel      context.CancelFunc
	interrupted bool
}

func newTeaModel(width int, cancel context.CancelFunc) teaModel {
	if width < 40 {
		width = 80
	}
	return teaModel{
		bars:   map[string]cprogress.Model{},
		width:  width,
		cancel: cancel,
	}
}

func (m teaModel) Init() tea.Cmd { return nil }

func (m teaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.resizeBars()
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			if m.cancel != nil {
				m.cancel()
			}
			m.interrupted = true
			m.quitting = true
			return m, tea.Quit
		}
	case updateMsg:
		u := Update(msg)
		m.inner.Apply(u)
		bar := m.ensureBar(u.ID)
		if u.Total > 0 {
			cmd := bar.SetPercent(float64(u.Current) / float64(u.Total))
			m.bars[u.ID] = bar
			return m, cmd
		}
		m.bars[u.ID] = bar
		return m, nil
	case quitMsg:
		m.err = msg.err
		m.quitting = true
		return m, tea.Quit
	case cprogress.FrameMsg:
		var cmds []tea.Cmd
		for id, bar := range m.bars {
			next, cmd := bar.Update(msg)
			if pm, ok := next.(cprogress.Model); ok {
				m.bars[id] = pm
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m teaModel) View() string {
	if len(m.inner.Order) == 0 {
		return ""
	}
	var b strings.Builder
	for _, id := range m.inner.Order {
		bar := m.inner.Bars[id]
		progressLine := ""
		if bar.Err == nil && bar.Warn == nil && bar.Status == "" && !bar.Done {
			if pbar, ok := m.bars[id]; ok {
				progressLine = pbar.View()
			}
		}
		fmt.Fprintln(&b, renderBarLine(bar, progressLine))
	}
	return b.String()
}

func (m *teaModel) ensureBar(id string) cprogress.Model {
	if bar, ok := m.bars[id]; ok {
		return bar
	}
	bar := cprogress.New(
		cprogress.WithDefaultGradient(),
		cprogress.WithWidth(m.barWidth()),
	)
	m.bars[id] = bar
	return bar
}

func (m *teaModel) resizeBars() {
	w := m.barWidth()
	for id, bar := range m.bars {
		bar.Width = w
		m.bars[id] = bar
	}
}

func (m teaModel) barWidth() int {
	w := m.width - 40
	if w < 10 {
		return 10
	}
	return w
}
