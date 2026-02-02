package tui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/willfish/te/internal/store"
)

const (
	colType  = "type"
	colCount = "count"
)

type typesLoadedMsg struct {
	counts []store.TypeCount
}

type TypesModel struct {
	store  *store.Store
	table  table.Model
	loaded bool
	width  int
	height int
}

func NewTypesModel(s *store.Store) TypesModel {
	columns := []table.Column{
		table.NewColumn(colType, "Type", 40).WithFiltered(true),
		table.NewColumn(colCount, "Count", 15),
	}

	t := table.New(columns).
		WithBaseStyle(lipgloss.NewStyle().Padding(0, 1)).
		Focused(true).
		WithPageSize(30).
		HeaderStyle(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")))

	return TypesModel{store: s, table: t}
}

func (m TypesModel) Init() tea.Cmd {
	return func() tea.Msg {
		counts, err := m.store.TypeCounts()
		if err != nil {
			return nil
		}
		return typesLoadedMsg{counts: counts}
	}
}

func (m TypesModel) WithDimensions(w, h int) TypesModel {
	m.width = w
	m.height = h
	m.table = m.table.WithTargetWidth(w).WithPageSize(h - 6)
	return m
}

func (m TypesModel) Update(msg tea.Msg) (TypesModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case typesLoadedMsg:
		rows := make([]table.Row, len(msg.counts))
		for i, tc := range msg.counts {
			rows[i] = table.NewRow(table.RowData{
				colType:  tc.Type,
				colCount: strconv.Itoa(tc.Count),
			})
		}
		m.table = m.table.WithRows(rows)
		m.loaded = true
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "enter" && m.loaded {
			selected := m.table.HighlightedRow()
			typeName, _ := selected.Data[colType].(string)
			if typeName == "" {
				return m, nil
			}
			countStr, _ := selected.Data[colCount].(string)
			count, _ := strconv.Atoi(countStr)
			return m, func() tea.Msg {
				return NavigateToElementsMsg{Type: typeName, Count: count}
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m TypesModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("Element Types")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑/↓ navigate • enter select • / filter • q quit")
	return fmt.Sprintf("\n  %s\n\n%s\n\n  %s", title, m.table.View(), help)
}
