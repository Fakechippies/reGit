package cmd

import (
	"fmt"
	"os"
	"strings"

	"reGit/dumper"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

const visibleFiles = 4

type progressMsg struct {
	total      int
	downloaded int
	current    string
}

type completeMsg struct {
	err error
}

type model struct {
	total      int
	downloaded int
	current    string
	recent     []string
	progress   progress.Model
	width      int
	done       bool
	err        error
}

func newModel() model {
	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithSpringOptions(18, 0.82),
	)

	return model{
		progress: bar,
		width:    40,
	}
}

func (m model) Init() tea.Cmd {
	return m.progress.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = clamp(msg.Width-8, 20, 72)
		m.progress.Width = m.width
		return m, nil

	case progressMsg:
		if msg.downloaded > m.downloaded && msg.current != "" {
			m.remember(msg.current)
		}

		m.total = msg.total
		m.downloaded = msg.downloaded
		m.current = msg.current

		return m, m.progress.SetPercent(m.percent())

	case completeMsg:
		m.done = true
		m.err = msg.err
		if m.err != nil || !m.progress.IsAnimating() {
			return m, tea.Quit
		}
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		if m.done && !m.progress.IsAnimating() {
			return m, tea.Quit
		}
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.err = fmt.Errorf("interrupted")
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	lines := []string{
		"reGit dump",
		m.progress.View(),
		fmt.Sprintf("%d/%d files", m.downloaded, m.total),
	}

	files := m.visibleFiles()
	for i, file := range files {
		if i == 0 && m.current != "" {
			lines = append(lines, "> "+file)
			continue
		}
		lines = append(lines, "  "+file)
	}

	if m.err != nil {
		lines = append(lines, "", m.err.Error())
	}

	return strings.Join(lines, "\n")
}

func (m *model) remember(path string) {
	if len(m.recent) == 0 || m.recent[len(m.recent)-1] != path {
		m.recent = append(m.recent, path)
	}
	if len(m.recent) > visibleFiles {
		m.recent = m.recent[len(m.recent)-visibleFiles:]
	}
}

func (m model) percent() float64 {
	if m.total == 0 {
		return 0
	}
	return float64(m.downloaded) / float64(m.total)
}

func (m model) visibleFiles() []string {
	files := make([]string, 0, visibleFiles)
	seen := map[string]struct{}{}

	if m.current != "" {
		files = append(files, m.fitPath(m.current))
		seen[m.current] = struct{}{}
	}

	for i := len(m.recent) - 1; i >= 0 && len(files) < visibleFiles; i-- {
		path := m.recent[i]
		if _, ok := seen[path]; ok {
			continue
		}
		files = append(files, m.fitPath(path))
		seen[path] = struct{}{}
	}

	for len(files) < visibleFiles {
		files = append(files, "")
	}

	return files
}

func (m model) fitPath(path string) string {
	maxWidth := max(m.width, 20)
	if len(path) <= maxWidth {
		return path
	}
	if maxWidth <= 3 {
		return path[:maxWidth]
	}
	return "..." + path[len(path)-maxWidth+3:]
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func runWithProgress(handler *dumper.Handler) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return handler.Run()
	}

	program := tea.NewProgram(newModel())

	go func() {
		err := handler.RunWithProgress(func(event dumper.ProgressEvent) {
			program.Send(progressMsg{
				total:      event.Total,
				downloaded: event.Downloaded,
				current:    event.Current,
			})
		})
		program.Send(completeMsg{err: err})
	}()

	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	if final, ok := finalModel.(model); ok && final.err != nil {
		return final.err
	}

	return nil
}
