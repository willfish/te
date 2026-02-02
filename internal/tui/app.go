package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/willfish/te/internal/store"
)

type screen int

const (
	screenTypes screen = iota
	screenElements
	screenDetail
)

type App struct {
	store   *store.Store
	current screen
	types   TypesModel
	elems   ElementsModel
	detail  DetailModel
	width   int
	height  int
}

func NewApp(s *store.Store) App {
	return App{
		store:   s,
		current: screenTypes,
		types:   NewTypesModel(s),
		elems:   NewElementsModel(s),
		detail:  NewDetailModel(),
	}
}

func (a App) Init() tea.Cmd {
	return a.types.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "q":
			switch a.current {
			case screenTypes:
				return a, tea.Quit
			case screenElements:
				a.current = screenTypes
				return a, a.types.Init()
			case screenDetail:
				a.current = screenElements
				return a, nil
			}
		case "esc":
			switch a.current {
			case screenElements:
				a.current = screenTypes
				return a, a.types.Init()
			case screenDetail:
				a.current = screenElements
				return a, nil
			}
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.types = a.types.WithDimensions(msg.Width, msg.Height)
		a.elems = a.elems.WithDimensions(msg.Width, msg.Height)
		a.detail = a.detail.WithDimensions(msg.Width, msg.Height)

	case NavigateToElementsMsg:
		a.current = screenElements
		a.elems = a.elems.ForType(msg.Type, msg.Count)
		return a, a.elems.Init()

	case NavigateToDetailMsg:
		a.current = screenDetail
		a.detail = a.detail.ForElement(msg.Hjid, msg.Data)
		return a, nil
	}

	var cmd tea.Cmd
	switch a.current {
	case screenTypes:
		a.types, cmd = a.types.Update(msg)
	case screenElements:
		a.elems, cmd = a.elems.Update(msg)
	case screenDetail:
		a.detail, cmd = a.detail.Update(msg)
	}

	return a, cmd
}

func (a App) View() string {
	switch a.current {
	case screenTypes:
		return a.types.View()
	case screenElements:
		return a.elems.View()
	case screenDetail:
		return a.detail.View()
	default:
		return ""
	}
}

type NavigateToElementsMsg struct {
	Type  string
	Count int
}

type NavigateToDetailMsg struct {
	Hjid string
	Data string
}
