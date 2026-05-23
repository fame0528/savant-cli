package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/spenc/savant-cli/internal/agent"
	"github.com/spenc/savant-cli/internal/provider"
	"github.com/spenc/savant-cli/internal/tools"
)

// Messages from the agent loop
type agentEventMsg agent.Event
type agentDoneMsg struct{}

// Model is the root Bubble Tea model for Savant CLI.
type Model struct {
	// Config
	provider provider.Provider
	registry *tools.Registry
	theme    *Theme
	maxTurns int
	width    int
	height   int

	// State
	messages  []chatMessage
	input     string
	cursorPos int
	streaming string // current streaming text
	working   bool   // agent is thinking
	scrollPos int
	err       error

	// Agent
	ctx    context.Context
	cancel context.CancelFunc
}

// chatMessage is a rendered message in the chat.
type chatMessage struct {
	role    string
	content string
	tool    string
}

// New creates a new TUI model.
func New(p provider.Provider, registry *tools.Registry, maxTurns int) Model {
	return Model{
		provider: p,
		registry: registry,
		theme:    NewCyberpunkTheme(),
		maxTurns: maxTurns,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
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
	case "ctrl+c", "q":
		return m, tea.Quit

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

	default:
		// Insert character
		if len(key) == 1 {
			m.input = m.input[:m.cursorPos] + key + m.input[m.cursorPos:]
			m.cursorPos++
		}
	}

	return m, nil
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	prompt := strings.TrimSpace(m.input)
	m.messages = append(m.messages, chatMessage{
		role:    "user",
		content: prompt,
	})
	m.input = ""
	m.cursorPos = 0
	m.working = true
	m.streaming = ""

	// Start agent in background
	m.ctx, m.cancel = context.WithCancel(context.Background())

	return m, func() tea.Msg {
		onEvent := func(e agent.Event) {
			// Events will be handled via the agent's return
		}
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
			role:    "tool",
			tool:    e.Tool,
			content: fmt.Sprintf("Calling %s...", e.Tool),
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
				role:    "assistant",
				content: m.streaming,
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

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	var sb strings.Builder

	// Title bar
	sb.WriteString(m.renderTitle())
	sb.WriteString("\n")

	// Chat area
	chatHeight := m.height - 6
	sb.WriteString(m.renderChat(chatHeight))
	sb.WriteString("\n")

	// Input area
	sb.WriteString(m.renderInput())
	sb.WriteString("\n")

	// Status bar
	sb.WriteString(m.renderStatusBar())

	v := tea.NewView(sb.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.BackgroundColor = VoidIndigo
	v.ForegroundColor = TextPrimary
	return v
}

func (m Model) renderTitle() string {
	title := " SAVANT "
	prov := fmt.Sprintf(" [%s] ", m.provider.Name())
	remaining := m.width - len(title) - len(prov) - 2
	if remaining < 0 {
		remaining = 0
	}
	bar := strings.Repeat("═", remaining)
	return m.theme.Title.Render(title + bar + prov)
}

func (m Model) renderChat(height int) string {
	if len(m.messages) == 0 && m.streaming == "" {
		welcome := "Welcome to Savant CLI. Type a message to get started.\n"
		welcome += "Commands: /help /provider /model /session /config\n"
		return m.theme.AssistantMessage.Render(welcome)
	}

	var lines []string

	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			lines = append(lines, m.theme.UserMessage.Render("▸ You: "+msg.content))
		case "assistant":
			lines = append(lines, m.theme.AssistantMessage.Render("▸ Savant: "+msg.content))
		case "tool":
			toolLine := fmt.Sprintf("  ⚡ %s: %s", msg.tool, msg.content)
			if len(toolLine) > m.width-4 {
				toolLine = toolLine[:m.width-7] + "..."
			}
			lines = append(lines, m.theme.ToolMessage.Render(toolLine))
		}
	}

	if m.streaming != "" {
		lines = append(lines, m.theme.AssistantMessage.Render("▸ Savant: "+m.streaming))
	}

	if m.working && m.streaming == "" {
		lines = append(lines, m.theme.Info.Render("  ⟳ Thinking..."))
	}

	if m.err != nil {
		lines = append(lines, m.theme.Error.Render("  ✗ Error: "+m.err.Error()))
	}

	if len(lines) > height {
		lines = lines[len(lines)-height:]
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderInput() string {
	prompt := " ▸ "
	if m.working {
		return m.theme.Input.Render(prompt + "Processing... (Ctrl+C to cancel)")
	}
	return m.theme.Input.Render(prompt + m.input + "█")
}

func (m Model) renderStatusBar() string {
	left := fmt.Sprintf(" %s ", m.provider.Name())
	right := " Ctrl+C: Quit | Enter: Send "
	gap := m.width - len(left) - len(right)
	if gap < 0 {
		gap = 0
	}
	return m.theme.StatusBar.Render(left + strings.Repeat(" ", gap) + right)
}
