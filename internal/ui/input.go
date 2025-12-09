package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func GetInput(prompt string, placeholder string, password bool) (string, error) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	if password {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}

	m := inputModel{
		textInput: ti,
		prompt:    prompt,
	}

	// Use Stderr to avoid polluting stdout
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := finalModel.(inputModel); ok && m.complete {
		return m.textInput.Value(), nil
	}
	return "", fmt.Errorf("cancelled")
}

type inputModel struct {
	textInput textinput.Model
	prompt    string
	complete  bool
	quitting  bool
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.complete = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	if m.complete {
		return ""
	}
	if m.quitting {
		return quitTextStyle.Render("Cancelled.")
	}
	return fmt.Sprintf(
		"\n%s\n\n%s\n\n",
		titleStyle.Render(m.prompt),
		m.textInput.View(),
	)
}
