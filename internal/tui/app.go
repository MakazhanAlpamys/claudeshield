package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/sandbox"
	"github.com/MakazhanAlpamys/claudeshield/pkg/types"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D4FF")).
			Padding(0, 1)

	statusRunning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF88")).
			Bold(true)

	statusStopped = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6666"))

	statusBlocked = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Bold(true)

	statusAllowed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#44FF44"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(1, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
)

// Tab represents a TUI tab.
type Tab int

const (
	TabSessions Tab = iota
	TabAudit
	TabRules
)

// Model is the main TUI model.
type Model struct {
	sessions     []*types.Session
	auditLog     []types.AuditEntry
	activeTab    Tab
	table        table.Model
	spinner      spinner.Model
	width        int
	height       int
	err          error
	quitting     bool
	notification string
	engine       *sandbox.Engine
	auditLogDir  string
}

type keyMap struct {
	Quit    key.Binding
	Tab     key.Binding
	Refresh key.Binding
	Stop    key.Binding
	Help    key.Binding
}

var keys = keyMap{
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch tab")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Stop:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stop session")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}

// tickMsg triggers periodic data refresh.
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// NewModel creates the initial TUI model.
func NewModel(engine *sandbox.Engine, auditLogDir string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF"))

	columns := []table.Column{
		{Title: "Session", Width: 25},
		{Title: "Agent", Width: 15},
		{Title: "State", Width: 12},
		{Title: "Container", Width: 14},
		{Title: "Project", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		BorderBottom(true).
		Bold(true)
	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00D4FF"))
	t.SetStyles(tableStyle)

	return Model{
		activeTab:   TabSessions,
		spinner:     s,
		table:       t,
		engine:      engine,
		auditLogDir: auditLogDir,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.refreshData(), tickCmd())
}

// refreshData fetches sessions and audit entries.
func (m Model) refreshData() tea.Cmd {
	return func() tea.Msg {
		if m.engine != nil {
			ctx := context.Background()
			sessions, err := m.engine.ListSessions(ctx)
			if err == nil {
				return SessionsMsg{Sessions: sessions}
			}
		}
		return nil
	}
}

func (m Model) refreshAudit() tea.Cmd {
	return func() tea.Msg {
		if m.auditLogDir != "" {
			entries, err := audit.ReadSession(m.auditLogDir, "")
			if err == nil {
				return AuditMsg{Entries: entries}
			}
		}
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.Tab):
			m.activeTab = (m.activeTab + 1) % 3
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		cmds = append(cmds, m.refreshData(), m.refreshAudit(), tickCmd())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case SessionsMsg:
		m.sessions = msg.Sessions
		m.updateTable()

	case AuditMsg:
		m.auditLog = msg.Entries

	case NotificationMsg:
		m.notification = msg.Text

	case error:
		m.err = msg
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return "👋 ClaudeShield stopped.\n"
	}

	header := m.renderHeader()
	tabs := m.renderTabs()
	content := m.renderContent()
	footer := m.renderFooter()

	return fmt.Sprintf("%s\n%s\n%s\n%s", header, tabs, content, footer)
}

func (m *Model) renderHeader() string {
	logo := `
 ╭───────────────────────────────────╮
 │  🛡️  ClaudeShield v0.1.0         │
 │  Secure sandbox for Claude Code  │
 ╰───────────────────────────────────╯`
	return titleStyle.Render(logo)
}

func (m *Model) renderTabs() string {
	tabs := []string{"Sessions", "Audit Log", "Rules"}
	rendered := ""
	for i, t := range tabs {
		style := lipgloss.NewStyle().Padding(0, 2)
		if Tab(i) == m.activeTab {
			style = style.Bold(true).
				Foreground(lipgloss.Color("#00D4FF")).
				Underline(true)
		} else {
			style = style.Foreground(lipgloss.Color("#888888"))
		}
		rendered += style.Render(t)
	}
	return rendered
}

func (m *Model) renderContent() string {
	switch m.activeTab {
	case TabSessions:
		return m.renderSessions()
	case TabAudit:
		return m.renderAudit()
	case TabRules:
		return m.renderRules()
	}
	return ""
}

func (m *Model) renderSessions() string {
	if len(m.sessions) == 0 {
		return borderStyle.Render(
			fmt.Sprintf("%s No active sessions. Run 'claudeshield start' to begin.", m.spinner.View()),
		)
	}
	return borderStyle.Render(m.table.View())
}

func (m *Model) renderAudit() string {
	if len(m.auditLog) == 0 {
		return borderStyle.Render("No audit entries yet.")
	}

	content := ""
	start := 0
	if len(m.auditLog) > 20 {
		start = len(m.auditLog) - 20
	}

	for _, entry := range m.auditLog[start:] {
		actionStyle := statusAllowed
		if entry.Action == types.ActionBlock {
			actionStyle = statusBlocked
		}

		line := fmt.Sprintf("[%s] %s %s %s",
			entry.Timestamp.Format("15:04:05"),
			actionStyle.Render(string(entry.Action)),
			entry.Command,
			entry.Reason,
		)
		content += line + "\n"
	}

	return borderStyle.Render(content)
}

func (m *Model) renderRules() string {
	return borderStyle.Render("Policy rules loaded from .claudeshield.yaml\nUse 'claudeshield init' to create a config.")
}

func (m *Model) renderFooter() string {
	notification := ""
	if m.notification != "" {
		notification = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Render("⚠  " + m.notification)
	}

	help := helpStyle.Render("tab: switch • r: refresh • s: stop • q: quit • ?: help")
	return fmt.Sprintf("%s\n%s", notification, help)
}

func (m *Model) updateTable() {
	var rows []table.Row
	for _, s := range m.sessions {
		containerID := s.ContainerID
		if len(containerID) > 12 {
			containerID = containerID[:12]
		}
		rows = append(rows, table.Row{
			s.ID,
			s.AgentName,
			string(s.State),
			containerID,
			s.ProjectDir,
		})
	}
	m.table.SetRows(rows)
}

// UpdateSessions sets new session data.
func (m *Model) UpdateSessions(sessions []*types.Session) {
	m.sessions = sessions
	m.updateTable()
}

// Message types for tea.Cmd

type SessionsMsg struct {
	Sessions []*types.Session
}

type AuditMsg struct {
	Entries []types.AuditEntry
}

type NotificationMsg struct {
	Text string
}
