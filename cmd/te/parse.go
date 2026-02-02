package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/willfish/te/internal/parsing"
	"github.com/willfish/te/internal/store"
)

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type ProgressReader struct {
	r          io.Reader
	total      int64
	read       int64
	onProgress func(float64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.read += int64(n)
	if pr.total > 0 && pr.onProgress != nil {
		pr.onProgress(float64(pr.read) / float64(pr.total))
	}
	return n, err
}

type progressUpdateMsg float64

func watchProgress(ch <-chan float64) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-ch
		if !ok {
			return progressDoneMsg{}
		}
		return progressUpdateMsg(p)
	}
}

type progressDoneMsg struct{}

type parseModel struct {
	stopwatch stopwatch.Model
	progress  progress.Model
	done      bool
}

func (m parseModel) Init() tea.Cmd {
	return tea.Batch(
		watchProgress(progressCh),
		m.stopwatch.Init(),
	)
}

func (m parseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var cmd tea.Cmd
	m.stopwatch, cmd = m.stopwatch.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, tea.Batch(cmds...)

	case progressDoneMsg:
		m.done = true
		cmd := m.progress.IncrPercent(1.0 - m.progress.Percent())
		cmds = append(cmds, cmd, m.stopwatch.Stop(), tea.Quit)
		return m, tea.Batch(cmds...)

	case progressUpdateMsg:
		current := m.progress.Percent()
		newPercent := float64(msg)
		if newPercent >= 1.0 {
			newPercent = 1.0
			cmds = append(cmds, m.stopwatch.Stop())
		}

		delta := newPercent - current
		cmd := m.progress.IncrPercent(delta)
		cmds = append(cmds, cmd, watchProgress(progressCh))

		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	default:
		return m, tea.Batch(cmds...)
	}
}

func (m parseModel) View() string {
	pad := strings.Repeat(" ", padding)
	centerStyle := lipgloss.NewStyle().Width(m.progress.Width).Align(lipgloss.Center)
	stopwatchCentered := centerStyle.Render("Elapsed: " + m.stopwatch.View())
	progressCentered := centerStyle.Render(m.progress.View())

	status := "Press any key to quit"
	if m.done {
		status = "Done! Press any key to quit"
	}
	help := helpStyle(status)

	return "\n" + pad + stopwatchCentered + "\n\n" + pad + progressCentered + "\n\n" + pad + help
}

var progressCh chan float64

func runParse(filename, dbPath string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stating file: %w", err)
	}

	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}

	// Non-interactive: parse synchronously with text progress
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		defer s.Close() //nolint:errcheck

		lastPct := -1
		pr := &ProgressReader{
			r:     f,
			total: fi.Size(),
			onProgress: func(p float64) {
				pct := int(p * 100)
				if pct > lastPct {
					lastPct = pct
					fmt.Fprintf(os.Stderr, "\rParsing... %d%%", pct)
				}
			},
		}

		if err := parsing.Parse(pr, s); err != nil {
			return fmt.Errorf("parsing: %w", err)
		}
		fmt.Fprintln(os.Stderr, "\rParsing... done.")
		return nil
	}

	// Interactive: parse in background goroutine with BubbleTea progress UI
	progressCh = make(chan float64)
	parseErr := make(chan error, 1)

	pr := &ProgressReader{
		r:     f,
		total: fi.Size(),
		onProgress: func(p float64) {
			select {
			case progressCh <- p:
			default:
			}
		},
	}

	go func() {
		parseErr <- parsing.Parse(pr, s)
		close(progressCh)
	}()

	m := parseModel{
		stopwatch: stopwatch.New(),
		progress:  progress.New(progress.WithDefaultGradient()),
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		// Wait for goroutine before closing store
		<-parseErr
		_ = s.Close()
		return fmt.Errorf("running progress UI: %w", err)
	}

	// Wait for parse goroutine to finish
	if err := <-parseErr; err != nil {
		_ = s.Close()
		return fmt.Errorf("parsing: %w", err)
	}

	return s.Close()
}
