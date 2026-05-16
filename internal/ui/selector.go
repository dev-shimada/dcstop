package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dev-shimada/dcstop/internal/devcontainer"
	"github.com/dev-shimada/dcstop/internal/docker"
)

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginLeft(2)
	selectedNameSty = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	normalNameSty   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	composeTypeSty  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	imageTypeSty    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	runningCountSty = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	dimSty          = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	spinnerSty      = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	checkSty        = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	errorSty        = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
)

// ProjectItem holds display data for a project selector entry.
type ProjectItem struct {
	Name       string
	ConfigType string // "image" or "compose"
	Running    int
	Total      int
}

// ── list item & delegate ──────────────────────────────────────────────────────

type projectListItem struct {
	idx     int
	project ProjectItem
}

func (p projectListItem) FilterValue() string { return p.project.Name }

type projectDelegate struct{}

func (d projectDelegate) Height() int                               { return 1 }
func (d projectDelegate) Spacing() int                              { return 0 }
func (d projectDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d projectDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	p, ok := item.(projectListItem)
	if !ok {
		return
	}
	proj := p.project
	isSelected := index == m.Index()

	var dot string
	if proj.Running > 0 {
		dot = runningCountSty.Render("●")
	} else {
		dot = dimSty.Render("○")
	}

	name := truncate(proj.Name, 35)
	paddedName := fmt.Sprintf("%-35s", name)
	var nameStr string
	if isSelected {
		nameStr = selectedNameSty.Render(paddedName)
	} else {
		nameStr = normalNameSty.Render(paddedName)
	}

	paddedType := fmt.Sprintf("%-8s", proj.ConfigType)
	var typeStr string
	if proj.ConfigType == "compose" {
		typeStr = composeTypeSty.Render(paddedType)
	} else {
		typeStr = imageTypeSty.Render(paddedType)
	}

	var countStr string
	if proj.Running > 0 {
		countStr = runningCountSty.Render(fmt.Sprintf("%d running", proj.Running)) +
			dimSty.Render(fmt.Sprintf(" / %d total", proj.Total))
	} else {
		countStr = dimSty.Render(fmt.Sprintf("%d running / %d total", proj.Running, proj.Total))
	}

	cursor := "  "
	if isSelected {
		cursor = "▸ "
	}

	_, _ = fmt.Fprintf(w, "%s%s %s  %s  %s\n", cursor, dot, nameStr, typeStr, countStr)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// ── select model ──────────────────────────────────────────────────────────────

type selectModel struct {
	list      list.Model
	selected  int
	cancelled bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.cancelled = true
			return m, tea.Quit
		}
		if msg.Type == tea.KeyEnter && m.list.FilterState() != list.Filtering {
			if item, ok := m.list.SelectedItem().(projectListItem); ok {
				m.selected = item.idx
				return m, tea.Quit
			}
		}
		if msg.String() == "q" && m.list.FilterState() != list.Filtering {
			m.cancelled = true
			return m, tea.Quit
		}
		// Ctrl+N / Ctrl+P for next/previous (Emacs-style)
		// Note: Cmd+N / Cmd+P are intercepted by the terminal app and never reach here.
		if msg.Type == tea.KeyCtrlN && m.list.FilterState() != list.Filtering {
			m.list.CursorDown()
			return m, nil
		}
		if msg.Type == tea.KeyCtrlP && m.list.FilterState() != list.Filtering {
			m.list.CursorUp()
			return m, nil
		}
	case tea.WindowSizeMsg:
		logoHeight := len(logoLines) + 1 // logo rows + blank line
		h := msg.Height - logoHeight - 4
		if h < 5 {
			h = 5
		}
		if h > 20 {
			h = 20
		}
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(h)
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectModel) View() string {
	return LogoString() + m.list.View()
}

// SelectProject shows an interactive filterable list of projects and returns
// the index of the selected item.
func SelectProject(label string, projects []ProjectItem) (int, error) {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectListItem{idx: i, project: p}
	}

	d := projectDelegate{}
	logoHeight := len(logoLines) + 1
	height := len(projects) + 7
	if height > 20 {
		height = 20
	}
	l := list.New(items, d, 80, height-logoHeight)
	l.Title = label
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	m := selectModel{list: l, selected: -1}
	prog := tea.NewProgram(m, tea.WithAltScreen())
	result, err := prog.Run()
	if err != nil {
		return 0, fmt.Errorf("selector error: %w", err)
	}

	final := result.(selectModel)
	if final.cancelled || final.selected < 0 {
		return 0, fmt.Errorf("selection cancelled")
	}
	return final.selected, nil
}

// ── spinner ───────────────────────────────────────────────────────────────────

type doneMsg struct{ err error }

type spinnerModel struct {
	spinner spinner.Model
	label   string
	fn      func() error
	done    bool
	err     error
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg { return doneMsg{err: m.fn()} },
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("interrupted")
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		if m.err != nil {
			return errorSty.Render("✗") + " " + m.label + "\n"
		}
		return checkSty.Render("✓") + " " + m.label + "\n"
	}
	return spinnerSty.Render(m.spinner.View()) + " " + m.label + "\n"
}

// RunWithSpinner runs fn while displaying a spinner labeled with label.
// The spinner is replaced by ✓ on success or ✗ on failure.
func RunWithSpinner(label string, fn func() error) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerSty

	m := spinnerModel{spinner: s, label: label, fn: fn}
	prog := tea.NewProgram(m)
	result, err := prog.Run()
	if err != nil {
		return fmt.Errorf("spinner error: %w", err)
	}
	return result.(spinnerModel).err
}

// ── confirm ───────────────────────────────────────────────────────────────────

type confirmModel struct {
	message string
	result  bool
	done    bool
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "y", "Y":
		m.result = true
		m.done = true
		return m, tea.Quit
	case "n", "N", "ctrl+c", "esc", "enter":
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m confirmModel) View() string {
	return dimSty.Render(m.message) + " " + selectedNameSty.Render("[y/N]") + " "
}

// Confirm shows a yes/no prompt. Defaults to no.
func Confirm(message string) (bool, error) {
	m := confirmModel{message: message}
	prog := tea.NewProgram(m)
	result, err := prog.Run()
	if err != nil {
		return false, err
	}
	return result.(confirmModel).result, nil
}

// ── DeduplicateConfigs ────────────────────────────────────────────────────────

// DeduplicateConfigs removes configs with duplicate project names,
// keeping only the first occurrence of each unique project name.
func DeduplicateConfigs(configs []*devcontainer.Config) []*devcontainer.Config {
	if len(configs) <= 1 {
		return configs
	}
	seen := make(map[string]bool)
	result := make([]*devcontainer.Config, 0, len(configs))
	for _, cfg := range configs {
		projectName := docker.DeriveProjectNameFromConfig(cfg)
		if !seen[projectName] {
			seen[projectName] = true
			result = append(result, cfg)
		}
	}
	return result
}
