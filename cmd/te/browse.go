package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/willfish/te/internal/store"
	"github.com/willfish/te/internal/tui"
)

func runBrowse(dbPath string) error {
	s, err := store.OpenReadOnly(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close() //nolint:errcheck

	app := tui.NewApp(s)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
