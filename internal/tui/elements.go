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
	maxSummary = 120
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
	loaded      bool
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
	m.loaded = false
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
		if len(jsonData) > maxSummary {
			return jsonData[:maxSummary]
		}
		return jsonData
	}

	for _, key := range []string{"sid", "description", "code", "descriptionPeriod.sid"} {
		if v, ok := obj[key]; ok {
			s := fmt.Sprintf("%s=%v", key, v)
			if len(s) > maxSummary {
				return s[:maxSummary]
			}
			return s
		}
	}

	summary := ""
	count := 0
	for k, v := range obj {
		if k == "__content__" || k == "hjid" {
			continue
		}
		if count > 0 {
			summary += ", "
		}
		val := fmt.Sprintf("%v", v)
		if len(val) > 40 {
			val = val[:40] + "..."
		}
		summary += fmt.Sprintf("%s=%s", k, val)
		count++
		if count >= 3 || len(summary) > maxSummary {
			break
		}
	}
	if len(summary) > maxSummary {
		summary = summary[:maxSummary]
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
		m.loaded = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if !m.loaded {
				return m, nil
			}
			selected := m.table.HighlightedRow()
			hjid, _ := selected.Data[colHjid].(string)
			if hjid == "" {
				return m, nil
			}
			data, _ := selected.Data["data"].(string)
			return m, func() tea.Msg {
				return NavigateToDetailMsg{Hjid: hjid, Data: data}
			}
		case "n":
			if m.offset+pageSize < m.totalCount {
				m.offset += pageSize
				m.loaded = false
				return m, m.loadPage()
			}
		case "p":
			if m.offset >= pageSize {
				m.offset -= pageSize
				m.loaded = false
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
		Render("↑/↓ navigate • enter detail • n/p next/prev page • / filter • q/esc back")
	return fmt.Sprintf("\n  %s  %s\n\n%s\n\n  %s", title, page, m.table.View(), help)
}
