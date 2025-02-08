package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/willfish/te/internal/parsing"
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
			return nil
		}
		return progressUpdateMsg(p)
	}
}

func main() {
	progressCh = make(chan float64)
	filename := os.ExpandEnv("./2024.xml")
	f, err := os.Open(filename)
	defer f.Close()

	if err != nil {
		log.Fatal(err)
	}

	fi, err := f.Stat()
	if err != nil {
		fmt.Println("Error stating file:", err)
		os.Exit(1)
	}

	pr := &ProgressReader{
		r:     f,
		total: fi.Size(),
		onProgress: func(p float64) {
			// Send progress updates to the channel. Using a non-blocking send.
			select {
			case progressCh <- p:
			default:
			}
		},
	}

	go func() {
		parsing.Parse(pr, filename)
		close(progressCh)
	}()
	m := model{
		stopwatch: stopwatch.New(),
		progress:  progress.New(progress.WithDefaultGradient()),
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

var progressCh chan float64

type tickMsg time.Time

type model struct {
	stopwatch stopwatch.Model
	progress  progress.Model
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		watchProgress(progressCh),
		m.stopwatch.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m model) View() string {
	pad := strings.Repeat(" ", padding)
	help := helpStyle("Press any key to quit")
	centerStyle := lipgloss.NewStyle().Width(m.progress.Width).Align(lipgloss.Center)
	stopwatchCentered := centerStyle.Render("Elapsed: " + m.stopwatch.View())
	progressCentered := centerStyle.Render(m.progress.View())

	return "\n" + pad + stopwatchCentered + "\n\n" + pad + progressCentered + "\n\n" + pad + help
}
