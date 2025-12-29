package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a single tab in the TUI
type Tab struct {
	Title   string
	Content string
}

// Model represents the TUI state
type Model struct {
	tabs         []Tab
	activeTab    int
	viewport     viewport.Model
	ready        bool
	width        int
	height       int
	keyMap       KeyMap
}

// KeyMap defines the keyboard shortcuts
type KeyMap struct {
	NextTab      key.Binding
	PrevTab      key.Binding
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Quit         key.Binding
}

// DefaultKeyMap returns the default keyboard shortcuts
func DefaultKeyMap() KeyMap {
	return KeyMap{
		NextTab: key.NewBinding(
			key.WithKeys("tab", "right", "l"),
			key.WithHelp("tab/→/l", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab", "left", "h"),
			key.WithHelp("shift+tab/←/h", "previous tab"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup/b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "f"),
			key.WithHelp("pgdn/f", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
			key.WithHelp("u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("d", "½ page down"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewModel creates a new TUI model with the given tabs
func NewModel(tabs []Tab) Model {
	return Model{
		tabs:      tabs,
		activeTab: 0,
		keyMap:    DefaultKeyMap(),
	}
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.NextTab):
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			m.viewport.SetContent(m.tabs[m.activeTab].Content)
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, m.keyMap.PrevTab):
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			m.viewport.SetContent(m.tabs[m.activeTab].Content)
			m.viewport.GotoTop()
			return m, nil
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMargins := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargins)
			m.viewport.YPosition = headerHeight
			if len(m.tabs) > 0 {
				m.viewport.SetContent(m.tabs[m.activeTab].Content)
			}
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargins
		}

		m.width = msg.Width
		m.height = msg.Height
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

// headerView renders the tab bar
func (m Model) headerView() string {
	var tabs []string

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(lipgloss.Color("235")).
		Padding(0, 2)

	for i, tab := range m.tabs {
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(tab.Title))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(tab.Title))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan")).
		Padding(0, 1)

	title := titleStyle.Render("Drift Analysis Report")

	header := lipgloss.JoinVertical(lipgloss.Left,
		title,
		tabBar,
		strings.Repeat("─", max(m.width, 1)),
	)

	return header
}

// footerView renders the footer with help text
func (m Model) footerView() string {
	// Get content from current tab instead of viewport
	content := ""
	if m.activeTab < len(m.tabs) {
		content = m.tabs[m.activeTab].Content
	}

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render(fmt.Sprintf(" %3.f%%  %d/%d ",
			m.viewport.ScrollPercent()*100,
			m.viewport.YOffset,
			len(strings.Split(content, "\n")),
		))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	help := helpStyle.Render(" tab: next • ←/→: switch • ↑/↓/pgup/pgdn: scroll • q: quit ")

	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(info)-lipgloss.Width(help)))

	footer := lipgloss.JoinHorizontal(lipgloss.Top, help, line, info)
	return footer
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
