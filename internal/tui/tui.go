package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/spenc/savant-cli/internal/agent"
	"github.com/spenc/savant-cli/internal/commands"
	"github.com/spenc/savant-cli/internal/pet"
	"github.com/spenc/savant-cli/internal/provider"
	"github.com/spenc/savant-cli/internal/session"
	"github.com/spenc/savant-cli/internal/tools"
)

// Messages
type agentEventMsg agent.Event
type agentDoneMsg struct{}
type tickMsg time.Time
type spinnerTickMsg struct{}

// Layout panels
const (
	panelSidebar = iota
	panelChat
	panelStatus
	panelInput
	panelLogs
)

// Model is the root Bubble Tea model for Savant CLI.
type Model struct {
	// Config
	provider     provider.Provider
	registry     *tools.Registry
	commands     *commands.Registry
	sessionSvc   *session.Service
	pet          *pet.Pet
	theme        *Theme
	maxTurns     int
	width        int
	height       int

	// Layout
	sidebarWidth int
	showSidebar  bool
	showLogs     bool
	logLines     []string

	// Chat state
	messages   []chatMessage
	input      string
	cursorPos  int
	streaming  string
	working    bool
	scrollPos  int
	err        error

	// Agent
	ctx    context.Context
	cancel context.CancelFunc

	// Animation
	spinnerFrame int
	tickCount    int
	glitchActive bool
	glitchFrame  int

	// Stats
	totalTokensIn  int
	totalTokensOut int
	totalCost      float64
	turnCount      int
	providerLatency time.Duration

	// Sidebar state
	sidebarTab   int // 0=files, 1=sessions, 2=tasks
	recentFiles  []string

	// Command palette
	showCmdPalette bool
	cmdFilter      string
}

type chatMessage struct {
	role      string
	content   string
	tool      string
	timestamp time.Time
}

func New(p provider.Provider, registry *tools.Registry, cmdReg *commands.Registry, sessionSvc *session.Service, petObj *pet.Pet, maxTurns int) Model {
	return Model{
		provider:     p,
		registry:     registry,
		commands:     cmdReg,
		sessionSvc:   sessionSvc,
		pet:          petObj,
		theme:        NewCyberpunkTheme(),
		maxTurns:     maxTurns,
		sidebarWidth: 30,
		showSidebar:  true,
		showLogs:     false,
		sidebarTab:   0,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), spinnerTickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func spinnerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case agentEventMsg:
		return m.handleAgentEvent(agent.Event(msg))

	case agentDoneMsg:
		m.working = false
		m.streaming = ""
		m.turnCount++
		return m, nil

	case tickMsg:
		m.tickCount++
		if m.tickCount%10 == 0 {
			m.glitchActive = !m.glitchActive
		}
		return m, tickCmd()

	case spinnerTickMsg:
		m.spinnerFrame = (m.spinnerFrame + 1) % 8
		if m.working {
			return m, spinnerTickCmd()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.working {
		if key == "ctrl+c" {
			if m.cancel != nil {
				m.cancel()
			}
			m.working = false
			m.streaming = ""
			return m, nil
		}
		return m, nil
	}

	switch key {
	case "ctrl+c":
		return m, tea.Quit

	case "ctrl+s":
		m.showSidebar = !m.showSidebar
	case "ctrl+l":
		m.showLogs = !m.showLogs
	case "ctrl+p":
		m.showCmdPalette = !m.showCmdPalette

	case "enter":
		if strings.TrimSpace(m.input) == "" {
			return m, nil
		}
		return m.handleSubmit()

	case "backspace":
		if len(m.input) > 0 && m.cursorPos > 0 {
			m.input = m.input[:m.cursorPos-1] + m.input[m.cursorPos:]
			m.cursorPos--
		}

	case "left":
		if m.cursorPos > 0 {
			m.cursorPos--
		}
	case "right":
		if m.cursorPos < len(m.input) {
			m.cursorPos++
		}
	case "home", "ctrl+a":
		m.cursorPos = 0
	case "end", "ctrl+e":
		m.cursorPos = len(m.input)
	case "up":
		if m.scrollPos > 0 {
			m.scrollPos--
		}
	case "down":
		m.scrollPos++

	case "tab":
		m.sidebarTab = (m.sidebarTab + 1) % 4

	default:
		if len(key) == 1 {
			m.input = m.input[:m.cursorPos] + key + m.input[m.cursorPos:]
			m.cursorPos++
		}
	}

	return m, nil
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	prompt := strings.TrimSpace(m.input)
	m.input = ""
	m.cursorPos = 0

	// Check for slash commands first
	if result, ok := m.commands.TryExecute(prompt); ok {
		m.messages = append(m.messages, chatMessage{
			role:      "user",
			content:   prompt,
			timestamp: time.Now(),
		})
		m.messages = append(m.messages, chatMessage{
			role:      "system",
			content:   result,
			timestamp: time.Now(),
		})
		return m, nil
	}

	// Regular message - send to agent
	m.messages = append(m.messages, chatMessage{
		role:      "user",
		content:   prompt,
		timestamp: time.Now(),
	})
	m.working = true
	m.streaming = ""

	m.ctx, m.cancel = context.WithCancel(context.Background())

	return m, func() tea.Msg {
		onEvent := func(e agent.Event) {}
		a := agent.NewAgent(m.provider, m.registry, m.maxTurns, onEvent)
		err := a.Run(m.ctx, prompt)
		if err != nil {
			return agentEventMsg(agent.Event{Type: agent.EventError, Error: err})
		}
		return agentDoneMsg{}
	}
}

func (m Model) handleAgentEvent(e agent.Event) (tea.Model, tea.Cmd) {
	switch e.Type {
	case agent.EventText:
		m.streaming += e.Content
	case agent.EventToolCall:
		m.messages = append(m.messages, chatMessage{
			role:      "tool",
			tool:      e.Tool,
			content:   fmt.Sprintf("Calling %s...", e.Tool),
			timestamp: time.Now(),
		})
	case agent.EventToolResult:
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].role == "tool" {
				m.messages[i].content = e.Content
				break
			}
		}
	case agent.EventDone:
		if m.streaming != "" {
			m.messages = append(m.messages, chatMessage{
				role:      "assistant",
				content:   m.streaming,
				timestamp: time.Now(),
			})
			m.streaming = ""
		}
		m.working = false
	case agent.EventError:
		m.err = e.Error
		m.working = false
		m.streaming = ""
	}
	return m, nil
}

// ─────────────────────────────────────────────────────────────
// VIEW - Complex multi-panel layout
// ─────────────────────────────────────────────────────────────

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Initializing Savant...")
	}

	chatWidth := m.width
	if m.showSidebar {
		chatWidth -= m.sidebarWidth + 1
	}

	// Build each panel
	titleBar := m.renderTitleBar()
	sidebar := ""
	if m.showSidebar {
		sidebar = m.renderSidebar()
	}
	chat := m.renderChatArea(chatWidth)
	toolPanel := m.renderToolPanel(chatWidth)
	inputArea := m.renderInputArea()
	statusBar := m.renderStatusBar()
	logPanel := ""
	if m.showLogs {
		logPanel = m.renderLogPanel()
	}

	// Assemble layout
	var sb strings.Builder
	sb.WriteString(titleBar)
	sb.WriteString("\n")

	// Main area: sidebar + chat/tool panels side by side
	if m.showSidebar {
		chatLines := strings.Split(chat, "\n")
		sideLines := strings.Split(sidebar, "\n")
		toolLines := strings.Split(toolPanel, "\n")

		// Interleave sidebar with chat + tool
		maxLines := max(len(chatLines)+len(toolLines), len(sideLines))
		for i := 0; i < maxLines; i++ {
			// Sidebar column
			if i < len(sideLines) {
				sb.WriteString(sideLines[i])
			} else {
				sb.WriteString(strings.Repeat(" ", m.sidebarWidth))
			}
			sb.WriteString("│")

			// Chat + tool column
			if i < len(chatLines) {
				sb.WriteString(chatLines[i])
			} else if i < len(chatLines)+len(toolLines) {
				sb.WriteString(toolLines[i-len(chatLines)])
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(chat)
		sb.WriteString(toolPanel)
	}

	sb.WriteString(inputArea)
	sb.WriteString("\n")

	if m.showLogs {
		sb.WriteString(logPanel)
		sb.WriteString("\n")
	}

	sb.WriteString(statusBar)

	v := tea.NewView(sb.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.BackgroundColor = VoidIndigo
	v.ForegroundColor = TextPrimary
	v.WindowTitle = "SAVANT CLI - Terminal-Native AI Coding Assistant"
	return v
}

// ─────────────────────────────────────────────────────────────
// TITLE BAR - Logo + provider + animated elements
// ─────────────────────────────────────────────────────────────

func (m Model) renderTitleBar() string {
	logo := m.theme.GlitchLogo(m.glitchFrame, m.glitchActive)
	provInfo := m.theme.ProviderBadge(m.provider.Name())

	// Animated separator
	sep := m.theme.AnimatedSeparator(m.width-len(logo)-len(provInfo)-4, m.tickCount)

	return lipgloss.JoinHorizontal(lipgloss.Top, logo, sep, provInfo)
}

// ─────────────────────────────────────────────────────────────
// SIDEBAR - Multi-tab panel (Files / Sessions / Tasks)
// ─────────────────────────────────────────────────────────────

func (m Model) renderSidebar() string {
	var sb strings.Builder

	// Tab bar
	tabs := []string{"📁 Files", "💬 Sessions", "📋 Tasks", "🐾 Pet"}
	tabBar := ""
	for i, tab := range tabs {
		if i == m.sidebarTab {
			tabBar += m.theme.TabActive.Render(tab)
		} else {
			tabBar += m.theme.TabInactive.Render(tab)
		}
	}
	sb.WriteString(m.theme.SidebarHeader.Render(" ╔"+"═"+strings.Repeat("═", m.sidebarWidth-4)+"╗ "))
	sb.WriteString("\n")
	sb.WriteString(tabBar)
	sb.WriteString("\n")
	sb.WriteString(" ╟"+strings.Repeat("─", m.sidebarWidth-3)+"╢")
	sb.WriteString("\n")

	// Tab content
	switch m.sidebarTab {
	case 0:
		sb.WriteString(m.renderFileTree())
	case 1:
		sb.WriteString(m.renderSessionList())
	case 2:
		sb.WriteString(m.renderTaskList())
	case 3:
		sb.WriteString(m.renderPetPanel())
	}

	// Bottom border
	sb.WriteString(" ╚"+strings.Repeat("═", m.sidebarWidth-3)+"╝ ")

	// Pad to fill width
	lines := strings.Split(sb.String(), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		stripped := stripAnsi(line)
		if len(stripped) < m.sidebarWidth {
			line += strings.Repeat(" ", m.sidebarWidth-len(stripped))
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func (m Model) renderFileTree() string {
	if len(m.recentFiles) == 0 {
		return m.theme.TextDim.Render("  No files opened yet.\n")
	}
	var sb strings.Builder
	for _, f := range m.recentFiles {
		sb.WriteString(m.theme.TextDim.Render("  ├─ " + f + "\n"))
	}
	return sb.String()
}

func (m Model) renderSessionList() string {
	var sb strings.Builder
	if m.turnCount == 0 {
		sb.WriteString(m.theme.TextDim.Render("  No active sessions.\n"))
	} else {
		sb.WriteString(m.theme.Info.Render(fmt.Sprintf("  Current session: %d turns\n", m.turnCount)))
	}
	return sb.String()
}

func (m Model) renderTaskList() string {
	var sb strings.Builder
	if m.working {
		sb.WriteString(m.theme.Warn.Render("  ⟳ Processing...\n"))
	}
	sb.WriteString(m.theme.TextDim.Render("  No tasks queued.\n"))
	return sb.String()
}

func (m Model) renderPetPanel() string {
	if m.pet == nil {
		return m.theme.TextDim.Render("  No pet yet.\n")
	}

	p := m.pet
	var sb strings.Builder

	// Pet animation frame
	frame := p.Frame(m.tickCount)
	for _, line := range strings.Split(frame, "\n") {
		sb.WriteString(m.theme.Info.Render("  " + line + "\n"))
	}

	// Name + mood
	mood := p.Mood().Emoji()
	sb.WriteString(m.theme.Info.Render(fmt.Sprintf("  %s %s\n", p.Name, mood)))

	// Stats bars
	barWidth := m.sidebarWidth - 8
	if barWidth < 10 {
		barWidth = 10
	}
	sb.WriteString(m.theme.Success.Render("  "+p.HPBar(barWidth)+"\n"))
	sb.WriteString(m.theme.Info.Render("  "+p.XPBar(barWidth)+"\n"))

	// Status
	sb.WriteString(m.theme.TextDim.Render("  "+p.StatusLine()+"\n"))
	sb.WriteString("\n")
	sb.WriteString(m.theme.TextDim.Render("  "+p.Stats()+"\n"))
	sb.WriteString("\n")

	// Actions
	sb.WriteString(m.theme.Warn.Render("  Commands:\n"))
	sb.WriteString(m.theme.TextDim.Render("  /pet feed   /pet play\n"))
	sb.WriteString(m.theme.TextDim.Render("  /pet rest   /pet heal\n"))
	sb.WriteString(m.theme.TextDim.Render("  /pet stats  /pet name\n"))

	return sb.String()
}

// ─────────────────────────────────────────────────────────────
// CHAT AREA - Messages with rich formatting
// ─────────────────────────────────────────────────────────────

func (m Model) renderChatArea(width int) string {
	if len(m.messages) == 0 && m.streaming == "" {
		return m.renderWelcome(width)
	}

	var lines []string

	// Header
	lines = append(lines, m.theme.ChatHeader.Render(" ╔═ CHAT "+strings.Repeat("═", width-12)+"╗"))

	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			lines = append(lines, m.renderUserMsg(msg, width))
		case "assistant":
			lines = append(lines, m.renderAssistantMsg(msg, width))
		case "tool":
			lines = append(lines, m.renderToolMsg(msg, width))
		case "system":
			lines = append(lines, m.theme.SystemMessage.Render("  ✦ "+msg.content))
		}
	}

	if m.streaming != "" {
		lines = append(lines, m.renderStreamingMsg(width))
	}

	if m.working && m.streaming == "" {
		spinner := m.theme.Spinner(m.spinnerFrame)
		lines = append(lines, m.theme.Info.Render("  "+spinner+" Processing..."))
	}

	if m.err != nil {
		lines = append(lines, m.theme.Error.Render("  ✗ ERROR: "+m.err.Error()))
	}

	// Footer
	lines = append(lines, m.theme.ChatHeader.Render(" ╚"+strings.Repeat("═", width-3)+"╝"))

	// Scroll
	chatHeight := m.height - 10
	if m.showLogs {
		chatHeight -= 6
	}
	if len(lines) > chatHeight {
		lines = lines[len(lines)-chatHeight:]
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderUserMsg(msg chatMessage, width int) string {
	timeStr := msg.timestamp.Format("15:04:05")
	header := m.theme.UserMsgHeader.Render(fmt.Sprintf(" ▸ YOU [%s]", timeStr))
	content := m.theme.UserMessage.Render("   " + msg.content)
	return header + "\n" + content
}

func (m Model) renderAssistantMsg(msg chatMessage, width int) string {
	timeStr := msg.timestamp.Format("15:04:05")
	header := m.theme.AssistantMsgHeader.Render(fmt.Sprintf(" ▸ SAVANT [%s]", timeStr))
	content := m.theme.AssistantMessage.Render("   " + msg.content)
	return header + "\n" + content
}

func (m Model) renderToolMsg(msg chatMessage, width int) string {
	icon := m.theme.ToolIcon.Render("⚡")
	name := m.theme.ToolName.Render(msg.tool)
	content := msg.content
	if len(content) > width-10 {
		content = content[:width-13] + "..."
	}
	return m.theme.ToolMessage.Render(fmt.Sprintf("   %s %s: %s", icon, name, content))
}

func (m Model) renderStreamingMsg(width int) string {
	spinner := m.theme.Spinner(m.spinnerFrame)
	header := m.theme.AssistantMsgHeader.Render(fmt.Sprintf(" ▸ SAVANT %s", spinner))
	content := m.theme.AssistantMessage.Render("   " + m.streaming + "▌")
	return header + "\n" + content
}

func (m Model) renderWelcome(width int) string {
	logo := m.theme.Logo()
	divider := m.theme.Divider(width - 4)
	help := m.theme.HelpText.Render(
		"  Welcome to Savant CLI — Terminal-Native AI Coding Assistant\n\n" +
		"  Commands:\n" +
		"    /help        Show all commands\n" +
		"    /provider    Configure AI providers\n" +
		"    /model       Switch model\n" +
		"    /session     Session management\n" +
		"    /config      View/edit configuration\n\n" +
		"  Keybindings:\n" +
		"    Ctrl+S       Toggle sidebar\n" +
		"    Ctrl+L       Toggle log panel\n" +
		"    Ctrl+P       Command palette\n" +
		"    Tab          Cycle sidebar tabs\n" +
		"    Enter        Send message\n" +
		"    Ctrl+C       Cancel / Quit\n\n" +
		"  Providers:\n" +
		"    9router      Local gateway (15+ providers)\n" +
		"    MiMo V2 Pro  Free via Xiaomi API\n" +
		"    Ollama       Local models\n",
	)

	return logo + "\n" + divider + "\n" + help
}

// ─────────────────────────────────────────────────────────────
// TOOL PANEL - Shows recent tool executions
// ─────────────────────────────────────────────────────────────

func (m Model) renderToolPanel(width int) string {
	var toolMsgs []chatMessage
	for _, msg := range m.messages {
		if msg.role == "tool" {
			toolMsgs = append(toolMsgs, msg)
		}
	}

	if len(toolMsgs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(m.theme.ToolPanelHeader.Render(" ╔═ TOOLS "+strings.Repeat("═", width-12)+"╗"))
	sb.WriteString("\n")

	// Show last 5 tool calls
	start := 0
	if len(toolMsgs) > 5 {
		start = len(toolMsgs) - 5
	}
	for _, msg := range toolMsgs[start:] {
		icon := m.theme.ToolIcon.Render("⚡")
		name := m.theme.ToolName.Render(msg.tool)
		result := msg.content
		if len(result) > width-15 {
			result = result[:width-18] + "..."
		}
		sb.WriteString(m.theme.ToolMessage.Render(fmt.Sprintf(" %s %s → %s\n", icon, name, result)))
	}

	sb.WriteString(m.theme.ToolPanelHeader.Render(" ╚"+strings.Repeat("═", width-3)+"╝"))
	return sb.String()
}

// ─────────────────────────────────────────────────────────────
// INPUT AREA - Rich input with prompt styling
// ─────────────────────────────────────────────────────────────

func (m Model) renderInputArea() string {
	if m.working {
		spinner := m.theme.Spinner(m.spinnerFrame)
		return m.theme.InputWorking.Render(fmt.Sprintf(" %s Processing... (Ctrl+C to cancel)", spinner))
	}

	prompt := m.theme.InputPrompt.Render(" ▸ ")
	input := m.theme.InputText.Render(m.input)
	cursor := m.theme.Cursor.Render("█")

	return m.theme.InputBox.Render(prompt + input + cursor)
}

// ─────────────────────────────────────────────────────────────
// STATUS BAR - Multi-column status with stats
// ─────────────────────────────────────────────────────────────

func (m Model) renderStatusBar() string {
	// Left: provider + model
	left := fmt.Sprintf(" %s ", m.provider.Name())

	// Center: stats
	center := fmt.Sprintf(" Turns: %d | Tokens: %d/%d | Cost: $%.4f ",
		m.turnCount, m.totalTokensIn, m.totalTokensOut, m.totalCost)

	// Right: keybindings
	right := " Ctrl+S:Sidebar | Ctrl+L:Logs | Ctrl+C:Quit "

	// Fill gaps
	leftLen := len(left)
	centerLen := len(center)
	rightLen := len(right)
	total := leftLen + centerLen + rightLen

	if total > m.width {
		return m.theme.StatusBar.Render(left + center + right)
	}

	gap1 := (m.width - total) / 2
	gap2 := m.width - total - gap1

	return m.theme.StatusBar.Render(
		left + strings.Repeat(" ", gap1) + center + strings.Repeat(" ", gap2) + right,
	)
}

// ─────────────────────────────────────────────────────────────
// LOG PANEL - Scrolling log output
// ─────────────────────────────────────────────────────────────

func (m Model) renderLogPanel() string {
	var sb strings.Builder
	sb.WriteString(m.theme.LogHeader.Render(" ╔═ LOGS "+strings.Repeat("═", m.width-11)+"╗"))
	sb.WriteString("\n")

	if len(m.logLines) == 0 {
		sb.WriteString(m.theme.TextDim.Render("  No log entries.\n"))
	} else {
		start := 0
		if len(m.logLines) > 4 {
			start = len(m.logLines) - 4
		}
		for _, line := range m.logLines[start:] {
			sb.WriteString(m.theme.TextDim.Render("  " + line + "\n"))
		}
	}

	sb.WriteString(m.theme.LogHeader.Render(" ╚"+strings.Repeat("═", m.width-3)+"╝"))
	return sb.String()
}
