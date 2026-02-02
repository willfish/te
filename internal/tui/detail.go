package tui

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DetailModel struct {
	viewport viewport.Model
	hjid     string
	width    int
	height   int
}

func NewDetailModel() DetailModel {
	return DetailModel{
		viewport: viewport.New(80, 24),
	}
}

func (m DetailModel) WithDimensions(w, h int) DetailModel {
	m.width = w
	m.height = h
	m.viewport.Width = w - 4
	m.viewport.Height = h - 6
	return m
}

func (m DetailModel) ForElement(hjid, data string) DetailModel {
	m.hjid = hjid

	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(data), "", "  "); err != nil {
		buf.WriteString(data)
	}

	m.viewport.SetContent(buf.String())
	m.viewport.GotoTop()
	return m
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DetailModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).
		Render(fmt.Sprintf("Element %s", m.hjid))
	scrollPct := fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100)
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("↑/↓/pgup/pgdn scroll • esc back")
	return fmt.Sprintf("\n  %s  %s\n\n%s\n\n  %s", title, scrollPct, m.viewport.View(), help)
}
