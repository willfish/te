package tui

import (
	"encoding/json"
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/willfish/te/internal/store"
)

const (
	colHjid    = "hjid"
	colSummary = "summary"
	pageSize   = 100
)

type elementsLoadedMsg struct {
	elements []store.Element
}

type ElementsModel struct {
	store       *store.Store
	table       table.Model
	elementType string
	totalCount  int
	offset      int
	width       int
	height      int
}

func NewElementsModel(s *store.Store) ElementsModel {
	columns := []table.Column{
		table.NewColumn(colHjid, "HJID", 20).WithFiltered(true),
		table.NewColumn(colSummary, "Summary", 80),
	}

	t := table.New(columns).
		WithBaseStyle(lipgloss.NewStyle().Padding(0, 1)).
		Focused(true).
		WithPageSize(30).
		HeaderStyle(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")))

	return ElementsModel{store: s, table: t}
}

func (m ElementsModel) ForType(elementType string, count int) ElementsModel {
	m.elementType = elementType
	m.totalCount = count
	m.offset = 0
	return m
}

func (m ElementsModel) WithDimensions(w, h int) ElementsModel {
	m.width = w
	m.height = h
	m.table = m.table.WithTargetWidth(w).WithPageSize(h - 6)
	return m
}

func (m ElementsModel) Init() tea.Cmd {
	return m.loadPage()
}

func (m ElementsModel) loadPage() tea.Cmd {
	elementType := m.elementType
	offset := m.offset
	s := m.store
	return func() tea.Msg {
		elements, err := s.Elements(elementType, pageSize, offset)
		if err != nil {
			return nil
		}
		return elementsLoadedMsg{elements: elements}
	}
}

func summarise(jsonData string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &obj); err != nil {
		return jsonData[:min(80, len(jsonData))]
	}

	// Try common summary fields
	for _, key := range []string{"sid", "description", "code", "descriptionPeriod.sid"} {
		if v, ok := obj[key]; ok {
			return fmt.Sprintf("%s=%v", key, v)
		}
	}

	// Fall back to first few keys
	summary := ""
	count := 0
	for k, v := range obj {
		if k == "__content__" || k == "hjid" {
			continue
		}
		if count > 0 {
			summary += ", "
		}
		summary += fmt.Sprintf("%s=%v", k, v)
		count++
		if count >= 3 {
			break
		}
	}
	return summary
}

func (m ElementsModel) Update(msg tea.Msg) (ElementsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case elementsLoadedMsg:
		rows := make([]table.Row, len(msg.elements))
		for i, el := range msg.elements {
			rows[i] = table.NewRow(table.RowData{
				colHjid:    el.Hjid,
				colSummary: summarise(el.Data),
				"data":     el.Data,
			})
		}
		m.table = m.table.WithRows(rows)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.table.HighlightedRow()
			hjid, _ := selected.Data[colHjid].(string)
			data, _ := selected.Data["data"].(string)
			return m, func() tea.Msg {
				return NavigateToDetailMsg{Hjid: hjid, Data: data}
			}
		case "n":
			if m.offset+pageSize < m.totalCount {
				m.offset += pageSize
				return m, m.loadPage()
			}
		case "p":
			if m.offset >= pageSize {
				m.offset -= pageSize
				return m, m.loadPage()
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m ElementsModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).
		Render(fmt.Sprintf("%s (%s total)", m.elementType, strconv.Itoa(m.totalCount)))
	page := fmt.Sprintf("Page %d/%d", m.offset/pageSize+1, (m.totalCount+pageSize-1)/pageSize)
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("↑/↓ navigate • enter detail • n/p next/prev page • / filter • esc back")
	return fmt.Sprintf("\n  %s  %s\n\n%s\n\n  %s", title, page, m.table.View(), help)
}
