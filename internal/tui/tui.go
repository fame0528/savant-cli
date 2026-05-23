package tui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/spenc/savant-cli/internal/agent"
	"github.com/spenc/savant-cli/internal/commands"
	"github.com/spenc/savant-cli/internal/pet"
	"github.com/spenc/savant-cli/internal/provider"
	"github.com/spenc/savant-cli/internal/session"
	"github.com/spenc/savant-cli/internal/tools"
)

// ─────────────────────────────────────────────────────────────
// Messages
// ─────────────────────────────────────────────────────────────

type agentEventMsg agent.Event
type agentDoneMsg struct{}
type tickMsg time.Time
type spinnerTickMsg struct{}

// eventSub is a tea.Cmd that reads from the agent event channel.
func eventSub(ch <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return agentDoneMsg{}
		}
		return agentEventMsg(e)
	}
}

// ─────────────────────────────────────────────────────────────
// Model
// ─────────────────────────────────────────────────────────────

// Model is the root Bubble Tea model for Savant CLI.
type Model struct {
	// Config
	provider   provider.Provider
	registry   *tools.Registry
	commands   *commands.Registry
	sessionSvc *session.Service
	pet        *pet.Pet
	theme      *Theme
	maxTurns   int
	width      int
	height     int

	// Layout
	sidebarWidth int
	showSidebar  bool
	showLogs     bool
	logLines     []string

	// Chat state
	messages  []chatMessage
	input     string
	cursorPos int // rune index, not byte index
	streaming string
	working   bool
	scrollPos int
	err       error

	// Agent
	ctx     context.Context
	cancel  context.CancelFunc
	evtChan chan agent.Event // channel for agent events

	// Conversation history (provider format) - preserved between turns
	agentMessages []provider.ChatMessage

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

	// Sidebar state
	sidebarTab  int // 0=files, 1=sessions, 2=tasks, 3=pet
	recentFiles []string
}

type chatMessage struct {
	role      string
	content   string
	tool      string
	timestamp time.Time
}

// New creates a new TUI model.
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
		// Flush any remaining streaming text
		if m.streaming != "" {
			m.messages = append(m.messages, chatMessage{
				role:      "assistant",
				content:   m.streaming,
				timestamp: time.Now(),
			})
			m.streaming = ""
		}
		m.turnCount++
		return m, nil

	case tickMsg:
		m.tickCount++
		// Advance animation frame every 2 seconds (20 ticks at 100ms)
		if m.tickCount%20 == 0 {
			m.glitchFrame = (m.glitchFrame + 1) % logoFrameCount
		}
		// Glitch flicker every 10 ticks
		if m.tickCount%10 == 0 {
			m.glitchActive = !m.glitchActive
		}
		// Pet tick every 60 seconds (600 ticks)
		if m.pet != nil && m.tickCount%600 == 0 {
			m.pet.Tick()
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

	// If agent is working, only accept cancel
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

	case "enter":
		if strings.TrimSpace(m.input) == "" {
			return m, nil
		}
		return m.handleSubmit()

	case "backspace":
		runes := []rune(m.input)
		if len(runes) > 0 && m.cursorPos > 0 {
			m.input = string(append(runes[:m.cursorPos-1], runes[m.cursorPos:]...))
			m.cursorPos--
		}

	case "left":
		if m.cursorPos > 0 {
			m.cursorPos--
		}
	case "right":
		runes := []rune(m.input)
		if m.cursorPos < len(runes) {
			m.cursorPos++
		}
	case "home", "ctrl+a":
		m.cursorPos = 0
	case "end", "ctrl+e":
		m.cursorPos = utf8.RuneCountInString(m.input)
	case "up":
		if m.scrollPos > 0 {
			m.scrollPos--
		}
	case "down":
		m.scrollPos++

	case "tab":
		m.sidebarTab = (m.sidebarTab + 1) % 4

	default:
		// Insert character (single rune only)
		r, size := utf8.DecodeRuneInString(key)
		if r != utf8.RuneError && size > 0 {
			runes := []rune(m.input)
			newRunes := make([]rune, 0, len(runes)+1)
			newRunes = append(newRunes, runes[:m.cursorPos]...)
			newRunes = append(newRunes, r)
			newRunes = append(newRunes, runes[m.cursorPos:]...)
			m.input = string(newRunes)
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
		if result == "__QUIT__" {
			return m, tea.Quit
		}
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

	// Pet event: user sent a message
	if m.pet != nil {
		m.pet.OnMessage()
	}

	// Regular message - send to agent
	m.messages = append(m.messages, chatMessage{
		role:      "user",
		content:   prompt,
		timestamp: time.Now(),
	})
	m.working = true
	m.streaming = ""

	// Create event channel and agent with prior conversation history
	m.evtChan = make(chan agent.Event, 64)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	a := agent.NewAgent(m.provider, m.registry, m.maxTurns, m.evtChan, m.agentMessages)

	// Start agent in a goroutine
	go func() {
		err := a.Run(m.ctx, prompt)
		// Save conversation history for next turn
		if err == nil {
			// We'll capture messages when done
		}
		close(m.evtChan)
	}()

	// Subscribe to events
	return m, eventSub(m.evtChan)
}

func (m Model) handleAgentEvent(e agent.Event) (tea.Model, tea.Cmd) {
	switch e.Type {
	case agent.EventText:
		m.streaming += e.Content
		return m, eventSub(m.evtChan)

	case agent.EventToolCall:
		m.messages = append(m.messages, chatMessage{
			role:      "tool",
			tool:      e.Tool,
			content:   fmt.Sprintf("Calling %s...", e.Tool),
			timestamp: time.Now(),
		})
		if m.pet != nil {
			m.pet.OnToolCall()
		}
		return m, eventSub(m.evtChan)

	case agent.EventToolResult:
		// Update the last tool message with the result
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].role == "tool" {
				m.messages[i].content = e.Content
				break
			}
		}
		return m, eventSub(m.evtChan)

	case agent.EventDone:
		// Handled by agentDoneMsg when channel closes
		return m, eventSub(m.evtChan)

	case agent.EventError:
		m.err = e.Error
		m.working = false
		m.streaming = ""
		return m, nil
	}

	return m, eventSub(m.evtChan)
}

// ─────────────────────────────────────────────────────────────
// Safe string utilities
// ─────────────────────────────────────────────────────────────

// safeRepeat returns empty string if n <= 0.
func safeRepeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
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
	if chatWidth < 10 {
		chatWidth = 10
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

		maxLines := max(len(chatLines)+len(toolLines), len(sideLines))
		for i := 0; i < maxLines; i++ {
			// Sidebar column
			if i < len(sideLines) {
				sb.WriteString(sideLines[i])
			} else {
				sb.WriteString(safeRepeat(" ", m.sidebarWidth))
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
// TITLE BAR
// ─────────────────────────────────────────────────────────────

func (m Model) renderTitleBar() string {
	logo := GetAnimatedLogo(m.glitchFrame, m.theme)
	provInfo := m.theme.ProviderBadge(m.provider.Name())

	logoW := len(stripAnsi(logo))
	provW := len(stripAnsi(provInfo))
	sepWidth := m.width - logoW - provW - 4
	sep := m.theme.AnimatedSeparator(sepWidth, m.tickCount)

	return lipgloss.JoinHorizontal(lipgloss.Top, logo, sep, provInfo)
}

// ─────────────────────────────────────────────────────────────
// SIDEBAR
// ─────────────────────────────────────────────────────────────

func (m Model) renderSidebar() string {
	var sb strings.Builder

	tabs := []string{"📁 Files", "💬 Sessions", "📋 Tasks", "🐾 Pet"}
	tabBar := ""
	for i, tab := range tabs {
		if i == m.sidebarTab {
			tabBar += m.theme.TabActive.Render(tab)
		} else {
			tabBar += m.theme.TabInactive.Render(tab)
		}
	}
	sb.WriteString(m.theme.SidebarHeader.Render(" ╔" + safeRepeat("═", m.sidebarWidth-4) + "╗ "))
	sb.WriteString("\n")
	sb.WriteString(tabBar)
	sb.WriteString("\n")
	sb.WriteString(" ╟" + safeRepeat("─", m.sidebarWidth-3) + "╢")
	sb.WriteString("\n")

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

	sb.WriteString(" ╚" + safeRepeat("═", m.sidebarWidth-3) + "╝ ")

	lines := strings.Split(sb.String(), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		stripped := stripAnsi(line)
		if len(stripped) < m.sidebarWidth {
			line += safeRepeat(" ", m.sidebarWidth-len(stripped))
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

	frame := p.Frame(m.tickCount)
	for _, line := range strings.Split(frame, "\n") {
		sb.WriteString(m.theme.Info.Render("  " + line + "\n"))
	}

	mood := p.Mood().Emoji()
	sb.WriteString(m.theme.Info.Render(fmt.Sprintf("  %s %s\n", p.Name, mood)))

	barWidth := m.sidebarWidth - 8
	if barWidth < 10 {
		barWidth = 10
	}
	sb.WriteString(m.theme.Success.Render("  "+p.HPBar(barWidth)+"\n"))
	sb.WriteString(m.theme.Info.Render("  "+p.XPBar(barWidth)+"\n"))

	sb.WriteString(m.theme.TextDim.Render("  "+p.StatusLine()+"\n"))
	sb.WriteString("\n")
	sb.WriteString(m.theme.TextDim.Render("  "+p.Stats()+"\n"))
	sb.WriteString("\n")

	sb.WriteString(m.theme.Warn.Render("  Commands:\n"))
	sb.WriteString(m.theme.TextDim.Render("  /pet feed   /pet play\n"))
	sb.WriteString(m.theme.TextDim.Render("  /pet rest   /pet heal\n"))
	sb.WriteString(m.theme.TextDim.Render("  /pet stats\n"))

	return sb.String()
}

// ─────────────────────────────────────────────────────────────
// CHAT AREA
// ─────────────────────────────────────────────────────────────

func (m Model) renderChatArea(width int) string {
	if len(m.messages) == 0 && m.streaming == "" {
		return m.renderWelcome(width)
	}

	var lines []string

	header := safeRepeat("═", max(1, width-12))
	lines = append(lines, m.theme.ChatHeader.Render(" ╔═ CHAT "+header+"╗"))

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

	footer := safeRepeat("═", max(1, width-3))
	lines = append(lines, m.theme.ChatHeader.Render(" ╚"+footer+"╝"))

	chatHeight := m.height - 10
	if m.showLogs {
		chatHeight -= 6
	}
	if chatHeight > 0 && len(lines) > chatHeight {
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
	if width > 18 && len(content) > width-18 {
		content = content[:width-21] + "..."
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
	logo := GetAnimatedLogo(m.glitchFrame, m.theme)
	divider := m.theme.Divider(max(1, width-4))
	help := m.theme.HelpText.Render(
		"  Welcome to Savant CLI — Terminal-Native AI Coding Assistant\n\n" +
			"  Commands:\n" +
			"    /help        Show all commands\n" +
			"    /provider    Configure AI providers\n" +
			"    /model       Switch model\n" +
			"    /session     Session management\n" +
			"    /config      View/edit configuration\n" +
			"    /pet         Interact with your pet\n\n" +
			"  Keybindings:\n" +
			"    Ctrl+S       Toggle sidebar\n" +
			"    Ctrl+L       Toggle log panel\n" +
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
// TOOL PANEL
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
	header := safeRepeat("═", max(1, width-12))
	sb.WriteString(m.theme.ToolPanelHeader.Render(" ╔═ TOOLS "+header+"╗"))
	sb.WriteString("\n")

	start := 0
	if len(toolMsgs) > 5 {
		start = len(toolMsgs) - 5
	}
	for _, msg := range toolMsgs[start:] {
		icon := m.theme.ToolIcon.Render("⚡")
		name := m.theme.ToolName.Render(msg.tool)
		result := msg.content
		if width > 18 && len(result) > width-18 {
			result = result[:width-21] + "..."
		}
		sb.WriteString(m.theme.ToolMessage.Render(fmt.Sprintf(" %s %s → %s\n", icon, name, result)))
	}

	footer := safeRepeat("═", max(1, width-3))
	sb.WriteString(m.theme.ToolPanelHeader.Render(" ╚" + footer + "╝"))
	return sb.String()
}

// ─────────────────────────────────────────────────────────────
// INPUT AREA
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
// STATUS BAR
// ─────────────────────────────────────────────────────────────

func (m Model) renderStatusBar() string {
	left := fmt.Sprintf(" %s ", m.provider.Name())
	center := fmt.Sprintf(" Turns: %d | Tokens: %d/%d | Cost: $%.4f ",
		m.turnCount, m.totalTokensIn, m.totalTokensOut, m.totalCost)
	right := " Ctrl+S:Sidebar | Ctrl+L:Logs | Ctrl+C:Quit "

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
		left + safeRepeat(" ", gap1) + center + safeRepeat(" ", gap2) + right,
	)
}

// ─────────────────────────────────────────────────────────────
// LOG PANEL
// ─────────────────────────────────────────────────────────────

func (m Model) renderLogPanel() string {
	var sb strings.Builder
	header := safeRepeat("═", max(1, m.width-11))
	sb.WriteString(m.theme.LogHeader.Render(" ╔═ LOGS "+header+"╗"))
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

	footer := safeRepeat("═", max(1, m.width-3))
	sb.WriteString(m.theme.LogHeader.Render(" ╚" + footer + "╝"))
	return sb.String()
}
