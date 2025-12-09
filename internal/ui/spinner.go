package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

type errMsg error

type taskResultMsg struct {
	data any
	err  error
}

type spinnerModel struct {
	spinner  spinner.Model
	text     string
	task     func() (any, error)
	result   any
	err      error
	quitting bool
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			res, err := m.task()
			return taskResultMsg{data: res, err: err}
		},
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.err = fmt.Errorf("cancelled by user")
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case taskResultMsg:
		m.result = msg.data
		m.err = msg.err
		m.quitting = true
		return m, tea.Quit

	case errMsg:
		m.err = msg
		m.quitting = true
		return m, tea.Quit

	default:
		return m, nil
	}
}

func (m spinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), textStyle.Render(m.text))
}

// Spin runs a blocking task with a spinner overlay.
// The task function returns a result (any) and an error.
// Spin returns (any, error).
func Spin(text string, task func() (any, error)) (any, error) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	m := spinnerModel{
		spinner: s,
		text:    text,
		task:    task,
	}

	// Use stderr to avoid polluting stdout
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	fm, ok := finalModel.(spinnerModel)
	if !ok {
		return nil, fmt.Errorf("internal error: invalid model type")
	}

	return fm.result, fm.err
}
