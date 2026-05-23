package tui

import (
	"context"
	"fmt"
	"os"
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

// ─────────────────────────────────────────────────────────────
// Messages
// ─────────────────────────────────────────────────────────────

type agentEventMsg agent.Event
type agentDoneMsg struct{}
type tickMsg time.Time
type spinnerTickMsg struct{}

// eventSub reads from the agent event channel and returns a tea.Cmd.
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
	provider   provider.Provider
	registry   *tools.Registry
	commands   *commands.Registry
	sessionSvc *session.Service
	pet        *pet.Pet
	theme      *Theme
	maxTurns   int
	width      int
	height     int

	sidebarWidth int
	showSidebar  bool
	showLogs     bool
	logLines     []string

	messages  []chatMessage
	streaming string
	working   bool
	scrollPos int
	err       error

	input     string
	cursorPos int

	fileTree    *FileTree // kept for completions ScanFiles
	completions *Completions
	dialogs     *DialogOverlay

	ctx     context.Context
	cancel  context.CancelFunc
	evtChan chan agent.Event

	agentMessages []provider.ChatMessage

	spinnerFrame int
	tickCount    int
	bootAnimFrame int

	totalTokensIn  int
	totalTokensOut int
	totalCost      float64
	turnCount      int

	sidebarTab int
}

type chatMessage struct {
	role      string
	content   string
	tool      string
	timestamp time.Time
}

func New(p provider.Provider, registry *tools.Registry, cmdReg *commands.Registry, sessionSvc *session.Service, petObj *pet.Pet, maxTurns int) Model {
	cwd, _ := os.Getwd()
	return Model{
		provider:     p,
		registry:     registry,
		commands:     cmdReg,
		sessionSvc:   sessionSvc,
		pet:          petObj,
		theme:        NewCyberpunkTheme(),
		maxTurns:     maxTurns,
		sidebarWidth: 30,
		showSidebar:  false,
		showLogs:     false,
		sidebarTab:   0,
		fileTree:     NewFileTree(cwd, 28),
		completions:  NewCompletions(40),
		dialogs:      NewDialogOverlay(),
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.showSidebar {
			m.fileTree = NewFileTree(m.getCwd(), m.sidebarWidth-4)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case agentEventMsg:
		return m.handleAgentEvent(agent.Event(msg))

	case agentDoneMsg:
		m.working = false
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
		// Boot animation: advance every 10 ticks (1 second per frame)
		if m.bootAnimFrame <= 4 && m.tickCount%10 == 0 {
			m.bootAnimFrame++
		}
		if m.pet != nil && m.tickCount%600 == 0 {
			m.pet.Tick()
		}
		return m, tickCmd()
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Dialog overlay
	if !m.dialogs.IsEmpty() {
		action := m.dialogs.HandleKey(msg)
		switch action {
		case DialogConfirm, DialogCancel, DialogSelect:
			m.dialogs.Pop()
			return m, nil
		}
		return m, nil
	}

	// Completions popup
	if m.completions.IsVisible() {
		switch key {
		case "up", "ctrl+p":
			m.completions.MoveUp()
			return m, nil
		case "down", "ctrl+n":
			m.completions.MoveDown()
			return m, nil
		case "enter", "tab":
			selected := m.completions.Selected()
			if selected != nil {
				atIdx := strings.LastIndex(m.input, "@")
				if atIdx >= 0 {
					m.input = m.input[:atIdx] + "@" + selected.Path + " "
					m.cursorPos = len([]rune(m.input))
				}
			}
			m.completions.Hide()
			return m, nil
		case "esc":
			m.completions.Hide()
			return m, nil
		}
		return m, nil
	}

	// Working: only cancel
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
	case "ctrl+b":
		m.showSidebar = !m.showSidebar
		return m, nil
	case "ctrl+l":
		m.showLogs = !m.showLogs
		return m, nil
	case "ctrl+p":
		cmds := m.commands.All()
		items := make([]string, len(cmds))
		for i, cmd := range cmds {
			items[i] = fmt.Sprintf("/%s - %s", cmd.Name, cmd.Description)
		}
		m.dialogs.Push(NewListDialog("commands", "Commands", items))
		return m, nil
	case "tab":
		m.sidebarTab = (m.sidebarTab + 1) % 4
		return m, nil
	case "up":
		if m.scrollPos > 0 {
			m.scrollPos--
		}
		return m, nil
	case "down":
		m.scrollPos++
		return m, nil
	case "enter":
		if strings.TrimSpace(m.input) != "" {
			return m.handleSubmit()
		}
		return m, nil
	case "backspace":
		runes := []rune(m.input)
		if len(runes) > 0 && m.cursorPos > 0 {
			m.input = string(append(runes[:m.cursorPos-1], runes[m.cursorPos:]...))
			m.cursorPos--
		}
		return m, nil
	case "left":
		if m.cursorPos > 0 {
			m.cursorPos--
		}
		return m, nil
	case "right":
		runes := []rune(m.input)
		if m.cursorPos < len(runes) {
			m.cursorPos++
		}
		return m, nil
	case "home", "ctrl+a":
		m.cursorPos = 0
		return m, nil
	case "end", "ctrl+e":
		m.cursorPos = len([]rune(m.input))
		return m, nil
	case "esc":
		m.input = ""
		m.cursorPos = 0
		return m, nil
	default:
		// Insert character from KeyPressMsg
		k := tea.Key(msg)
		text := k.Text
		if text != "" {
			runes := []rune(m.input)
			insert := []rune(text)
			newRunes := make([]rune, 0, len(runes)+len(insert))
			newRunes = append(newRunes, runes[:m.cursorPos]...)
			newRunes = append(newRunes, insert...)
			newRunes = append(newRunes, runes[m.cursorPos:]...)
			m.input = string(newRunes)
			m.cursorPos += len(insert)

			// Check for @ mentions
			if strings.Contains(m.input, "@") && !m.completions.IsVisible() {
				atIdx := strings.LastIndex(m.input, "@")
				if atIdx >= 0 {
					query := m.input[atIdx+1:]
					if len(query) > 0 && !strings.Contains(query, " ") {
						items := ScanFiles(m.getCwd())
						m.completions.Show(items)
						m.completions.Filter(query)
					}
				}
			}
		}
		return m, nil
	}
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	prompt := strings.TrimSpace(m.input)
	m.input = ""
	m.cursorPos = 0
	m.completions.Hide()

	// Slash commands
	if result, ok := m.commands.TryExecute(prompt); ok {
		if result == "__QUIT__" {
			return m, tea.Quit
		}
		m.messages = append(m.messages, chatMessage{role: "user", content: prompt, timestamp: time.Now()})
		m.messages = append(m.messages, chatMessage{role: "system", content: result, timestamp: time.Now()})
		return m, nil
	}

	if m.pet != nil {
		m.pet.OnMessage()
	}

	m.messages = append(m.messages, chatMessage{role: "user", content: prompt, timestamp: time.Now()})
	m.working = true
	m.streaming = ""
	m.err = nil

	m.evtChan = make(chan agent.Event, 64)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	a := agent.NewAgent(m.provider, m.registry, m.maxTurns, m.evtChan, m.agentMessages)

	go func() {
		a.Run(m.ctx, prompt)
		close(m.evtChan)
	}()

	return m, eventSub(m.evtChan)
}

func (m Model) handleAgentEvent(e agent.Event) (tea.Model, tea.Cmd) {
	switch e.Type {
	case agent.EventText:
		m.streaming += e.Content
		return m, eventSub(m.evtChan)
	case agent.EventThinking:
		// Show thinking content as a system message with yellow styling
		m.messages = append(m.messages, chatMessage{role: "thinking", content: e.Content, timestamp: time.Now()})
		return m, eventSub(m.evtChan)
	case agent.EventToolCall:
		m.messages = append(m.messages, chatMessage{role: "tool", tool: e.Tool, content: "Calling " + e.Tool + "...", timestamp: time.Now()})
		if m.pet != nil {
			m.pet.OnToolCall()
		}
		return m, eventSub(m.evtChan)
	case agent.EventToolResult:
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].role == "tool" {
				m.messages[i].content = e.Content
				break
			}
		}
		return m, eventSub(m.evtChan)
	case agent.EventHistoryUpdate:
		// Save conversation history for next turn
		m.agentMessages = e.Messages
		return m, eventSub(m.evtChan)
	case agent.EventDone:
		return m, eventSub(m.evtChan)
	case agent.EventError:
		m.err = e.Error
		m.working = false
		m.streaming = ""
		return m, nil
	}
	return m, eventSub(m.evtChan)
}

func (m Model) getCwd() string {
	cwd, _ := os.Getwd()
	return cwd
}

func safeRepeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

// ─────────────────────────────────────────────────────────────
// VIEW
// ─────────────────────────────────────────────────────────────

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Initializing Savant...")
	}

	chatWidth := m.width
	if m.showSidebar {
		chatWidth -= m.sidebarWidth + 1
	}
	if chatWidth < 20 {
		chatWidth = 20
	}

	// Calculate how many lines each section needs
	footerLines := 1 // input
	if m.showLogs {
		footerLines += 5
	}
	footerLines++ // status bar

	mainHeight := m.height - 1 - footerLines // -1 for title bar
	if mainHeight < 1 {
		mainHeight = 1
	}

	// Build the output line by line
	var lines []string

	// Title bar
	lines = append(lines, m.renderTitleBar())

	// Main content
	if m.showSidebar {
		mainContent := m.renderMainArea(chatWidth, mainHeight)
		lines = append(lines, strings.Split(mainContent, "\n")...)
	} else {
		mainContent := m.renderChatArea(chatWidth, mainHeight)
		lines = append(lines, strings.Split(mainContent, "\n")...)
	}

	// Input area
	lines = append(lines, m.renderInputArea())

	// Log panel
	if m.showLogs {
		logContent := m.renderLogPanel()
		lines = append(lines, strings.Split(logContent, "\n")...)
	}

	// Status bar
	lines = append(lines, m.renderStatusBar())

	// Pad to exactly m.height
	for len(lines) < m.height {
		lines = append(lines, "")
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}

	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.BackgroundColor = Background
	v.ForegroundColor = TextPrimary
	v.WindowTitle = "SAVANT CLI"
	return v
}

func (m Model) renderTitleBar() string {
	logo := m.theme.TitleLogo.Render(" SAVANT ")
	sep := m.theme.AnimatedSeparator(max(1, m.width-30), m.tickCount)
	prov := m.theme.ProviderBadge(m.provider.Name())
	return lipgloss.JoinHorizontal(lipgloss.Top, logo, sep, prov)
}

func (m Model) renderMainArea(chatWidth, height int) string {
	sidebar := m.renderSidebar(height)
	chat := m.renderChatArea(chatWidth, height)

	sideLines := strings.Split(sidebar, "\n")
	chatLines := strings.Split(chat, "\n")

	var sb strings.Builder
	maxLines := max(len(sideLines), len(chatLines))
	if maxLines > height {
		maxLines = height
	}

	for i := 0; i < maxLines; i++ {
		if i < len(sideLines) {
			line := sideLines[i]
			stripped := stripAnsi(line)
			if len(stripped) < m.sidebarWidth {
				line += safeRepeat(" ", m.sidebarWidth-len(stripped))
			}
			if len(stripped) > m.sidebarWidth {
				line = line[:m.sidebarWidth]
			}
			sb.WriteString(line)
		} else {
			sb.WriteString(safeRepeat(" ", m.sidebarWidth))
		}
		sb.WriteString("│")
		if i < len(chatLines) {
			sb.WriteString(chatLines[i])
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m Model) renderSidebar(height int) string {
	var sb strings.Builder

	tabs := []string{"Info", "Pet", "Tasks", "Logs"}
	icons := []string{"ℹ", "🐾", "📋", "📄"}
	tabBar := ""
	for i, tab := range tabs {
		if i == m.sidebarTab {
			tabBar += m.theme.TabActive.Render(icons[i]+" "+tab) + " "
		} else {
			tabBar += m.theme.TabInactive.Render(icons[i]+" "+tab) + " "
		}
	}
	sb.WriteString(tabBar)
	sb.WriteString("\n")
	sb.WriteString(m.theme.DividerLine.Render(safeRepeat("─", m.sidebarWidth)))
	sb.WriteString("\n")

	contentHeight := height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	switch m.sidebarTab {
	case 0:
		sb.WriteString(m.renderInfoPanel(contentHeight))
	case 1:
		sb.WriteString(m.renderPetPanel(contentHeight))
	case 2:
		sb.WriteString(m.renderTaskPanel(contentHeight))
	case 3:
		sb.WriteString(m.renderLogsPanel(contentHeight))
	}

	return sb.String()
}

func (m Model) renderInfoPanel(maxLines int) string {
	var sb strings.Builder

	// Provider info
	sb.WriteString(m.theme.Info.Bold(true).Render("  Provider") + "\n")
	sb.WriteString(m.theme.TextPrimary.Render("  "+m.provider.Name()) + "\n")
	sb.WriteString(m.theme.DividerLine.Render("  "+safeRepeat("─", m.sidebarWidth-4)) + "\n")

	// Session info
	sb.WriteString(m.theme.Info.Bold(true).Render("  Session") + "\n")
	sb.WriteString(m.theme.TextDim.Render(fmt.Sprintf("  Turns: %d", m.turnCount)) + "\n")
	sb.WriteString(m.theme.TextDim.Render(fmt.Sprintf("  Messages: %d", len(m.messages))) + "\n")
	sb.WriteString(m.theme.TextDim.Render(fmt.Sprintf("  History: %d msgs", len(m.agentMessages))) + "\n")
	sb.WriteString(m.theme.DividerLine.Render("  "+safeRepeat("─", m.sidebarWidth-4)) + "\n")

	// Working directory
	sb.WriteString(m.theme.Info.Bold(true).Render("  Directory") + "\n")
	cwd := m.getCwd()
	if len(cwd) > m.sidebarWidth-4 {
		cwd = "..." + cwd[len(cwd)-m.sidebarWidth+7:]
	}
	sb.WriteString(m.theme.TextDim.Render("  "+cwd) + "\n")

	return sb.String()
}

func (m Model) renderTaskPanel(maxLines int) string {
	var sb strings.Builder
	if m.working {
		sb.WriteString(m.theme.Warn.Render("  ⟳ Processing...\n"))
	}
	sb.WriteString(m.theme.TextDim.Render("  No tasks queued.\n"))
	return sb.String()
}

func (m Model) renderLogsPanel(maxLines int) string {
	var sb strings.Builder
	sb.WriteString(m.theme.Info.Bold(true).Render("  Logs") + "\n")
	sb.WriteString(m.theme.DividerLine.Render("  "+safeRepeat("─", m.sidebarWidth-4)) + "\n")
	if len(m.logLines) == 0 {
		sb.WriteString(m.theme.TextDim.Render("  No log entries.\n"))
	} else {
		start := 0
		if len(m.logLines) > maxLines-2 {
			start = len(m.logLines) - maxLines + 2
		}
		for _, line := range m.logLines[start:] {
			sb.WriteString(m.theme.TextDim.Render("  "+line) + "\n")
		}
	}
	return sb.String()
}

func (m Model) renderPetPanel(maxLines int) string {
	if m.pet == nil {
		return m.theme.TextDim.Render("  No pet yet.\n")
	}
	p := m.pet
	var sb strings.Builder
	frame := p.Frame(m.tickCount)
	for _, line := range strings.Split(frame, "\n") {
		sb.WriteString(m.theme.Info.Render(" " + line + "\n"))
	}
	mood := p.Mood().Emoji()
	sb.WriteString(m.theme.Info.Render(fmt.Sprintf(" %s %s\n", p.Name, mood)))
	barWidth := m.sidebarWidth - 6
	if barWidth < 8 {
		barWidth = 8
	}
	sb.WriteString(m.theme.Success.Render(" "+p.HPBar(barWidth)+"\n"))
	sb.WriteString(m.theme.Info.Render(" "+p.XPBar(barWidth)+"\n"))
	sb.WriteString(m.theme.TextDim.Render(" "+p.StatusLine()+"\n"))
	return sb.String()
}

func (m Model) renderChatArea(width, height int) string {
	if len(m.messages) == 0 && m.streaming == "" && !m.working {
		return m.renderWelcome(width, height)
	}

	var lines []string
	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			lines = append(lines, m.renderUserMsg(msg, width)...)
		case "assistant":
			lines = append(lines, m.renderAssistantMsg(msg, width)...)
		case "tool":
			lines = append(lines, m.renderToolMsg(msg, width))
		case "thinking":
			lines = append(lines, m.theme.ThinkingMessage.Render("  💭 "+msg.content))
		case "system":
			lines = append(lines, m.theme.SystemMessage.Render("  ✦ "+msg.content))
		}
	}
	if m.streaming != "" {
		lines = append(lines, m.renderStreamingMsg(width)...)
	}
	if m.working && m.streaming == "" {
		spinner := m.theme.Spinner(m.spinnerFrame)
		lines = append(lines, m.theme.Info.Render("  "+spinner+" Thinking..."))
	}
	if m.err != nil {
		lines = append(lines, m.theme.Error.Render("  ✗ "+m.err.Error()))
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}
	// Clip to height (scroll from bottom)
	if len(lines) > height {
		lines = lines[len(lines)-height:]
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderUserMsg(msg chatMessage, width int) []string {
	timeStr := msg.timestamp.Format("3:04PM")
	header := m.theme.UserMsgHeader.Render(fmt.Sprintf(" YOU [%s]", timeStr))
	wrapped := wordWrap(msg.content, width-6)
	result := []string{header}
	for _, line := range wrapped {
		result = append(result, m.theme.UserMessage.Render("  "+line))
	}
	return result
}

func (m Model) renderAssistantMsg(msg chatMessage, width int) []string {
	timeStr := msg.timestamp.Format("3:04PM")
	header := m.theme.AssistantMsgHeader.Render(fmt.Sprintf(" SAVANT [%s]", timeStr))
	wrapped := wordWrap(msg.content, width-6)
	result := []string{header}
	for _, line := range wrapped {
		result = append(result, m.theme.AssistantMessage.Render("  "+line))
	}
	return result
}

func (m Model) renderToolMsg(msg chatMessage, width int) string {
	icon := m.theme.ToolIcon.Render("⚡")
	name := m.theme.ToolName.Render(msg.tool)
	content := msg.content
	maxLen := width - 12
	if maxLen < 10 {
		maxLen = 10
	}
	if len(content) > maxLen {
		content = content[:maxLen-3] + "..."
	}
	return m.theme.ToolMessage.Render(fmt.Sprintf("  %s %s: %s", icon, name, content))
}

func (m Model) renderStreamingMsg(width int) []string {
	spinner := m.theme.Spinner(m.spinnerFrame)
	header := m.theme.AssistantMsgHeader.Render(fmt.Sprintf(" SAVANT %s", spinner))
	wrapped := wordWrap(m.streaming, width-6)
	result := []string{header}
	for _, line := range wrapped {
		result = append(result, m.theme.AssistantMessage.Render("  "+line+"▌"))
	}
	return result
}

func (m Model) renderWelcome(width, height int) string {
	var sb strings.Builder

	// Boot animation: show logo for 5 seconds, then disappear
	logo := bootFrame(m.bootAnimFrame, m.theme)
	if logo != "" {
		sb.WriteString(logo)
		sb.WriteString("\n")
		sb.WriteString(m.theme.Divider(max(1, width-4)))
		sb.WriteString("\n")
	}
	sb.WriteString(m.theme.Divider(max(1, width-4)))
	sb.WriteString("\n")
	help := []string{
		"",
		"  Type a message to start chatting. Commands:",
		"",
		"  /help       Show all commands     /config     View config",
		"  /provider   Configure providers   /pet        Your pet",
		"  /model      Switch model          /session    Sessions",
		"",
		"  Ctrl+B  Sidebar  Ctrl+L  Logs  Ctrl+P  Commands  Tab  Tabs",
		"",
		fmt.Sprintf("  Provider: %s", m.provider.Name()),
	}
	for _, line := range help {
		sb.WriteString(m.theme.HelpText.Render(line))
		sb.WriteString("\n")
	}
	welcomeStr := sb.String()
	welcomeLines := strings.Split(welcomeStr, "\n")
	for len(welcomeLines) < height {
		welcomeLines = append(welcomeLines, "")
	}
	if len(welcomeLines) > height {
		welcomeLines = welcomeLines[:height]
	}
	return strings.Join(welcomeLines, "\n")
}

func (m Model) renderInputArea() string {
	if m.working {
		spinner := m.theme.Spinner(m.spinnerFrame)
		return m.theme.InputWorking.Render(fmt.Sprintf(" %s Processing... (Ctrl+C to cancel)", spinner))
	}
	prompt := m.theme.InputPrompt.Render(" ▸ ")
	runes := []rune(m.input)
	cursor := min(m.cursorPos, len(runes))
	before := string(runes[:cursor])
	after := string(runes[cursor:])
	cursorChar := m.theme.Cursor.Render("█")
	inputContent := m.theme.InputText.Render(before) + cursorChar + m.theme.InputText.Render(after)
	return m.theme.InputBox.Render(prompt + inputContent)
}

func (m Model) renderStatusBar() string {
	left := fmt.Sprintf(" %s ", m.provider.Name())
	center := fmt.Sprintf(" Turns:%d ", m.turnCount)
	right := " Ctrl+B:Sidebar │ Ctrl+L:Logs │ Ctrl+C:Quit "
	leftLen := len(left)
	centerLen := len(center)
	rightLen := len(right)
	total := leftLen + centerLen + rightLen
	if total >= m.width {
		return m.theme.StatusBar.Render(left + right)
	}
	gap1 := (m.width - total) / 2
	gap2 := m.width - total - gap1
	return m.theme.StatusBar.Render(left + safeRepeat(" ", gap1) + center + safeRepeat(" ", gap2) + right)
}

func (m Model) renderLogPanel() string {
	var sb strings.Builder
	sb.WriteString(m.theme.LogHeader.Render(" LOGS"))
	sb.WriteString("\n")
	if len(m.logLines) == 0 {
		sb.WriteString(m.theme.TextDim.Render("  No log entries.\n"))
	} else {
		start := 0
		if len(m.logLines) > 3 {
			start = len(m.logLines) - 3
		}
		for _, line := range m.logLines[start:] {
			sb.WriteString(m.theme.TextDim.Render("  " + line + "\n"))
		}
	}
	return sb.String()
}
