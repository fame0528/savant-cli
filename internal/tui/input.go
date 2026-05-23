package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// InputEditor wraps the Bubbles textarea for multi-line input.
type InputEditor struct {
	textarea textarea.Model
	focused  bool
}

// NewInputEditor creates a new input editor.
func NewInputEditor() InputEditor {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.CharLimit = 0 // unlimited
	ta.SetWidth(80)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Enter submits, Shift+Enter for newline

	return InputEditor{
		textarea: ta,
		focused:  true,
	}
}

// Update handles messages for the input editor.
func (e *InputEditor) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return cmd
}

// Value returns the current input text.
func (e *InputEditor) Value() string {
	return e.textarea.Value()
}

// SetValue sets the input text.
func (e *InputEditor) SetValue(s string) {
	e.textarea.SetValue(s)
}

// Reset clears the input.
func (e *InputEditor) Reset() {
	e.textarea.SetValue("")
}

// Focus gives focus to the input.
func (e *InputEditor) Focus() {
	e.textarea.Focus()
	e.focused = true
}

// Blur removes focus from the input.
func (e *InputEditor) Blur() {
	e.textarea.Blur()
	e.focused = false
}

// SetWidth sets the input width.
func (e *InputEditor) SetWidth(w int) {
	e.textarea.SetWidth(w)
}

// SetHeight sets the input height.
func (e *InputEditor) SetHeight(h int) {
	e.textarea.SetHeight(h)
}

// View renders the input editor with cyberpunk styling.
func (e InputEditor) View(theme *Theme, working bool) string {
	if working {
		return theme.InputWorking.Render(" ⟳ Processing... (Ctrl+C to cancel)")
	}

	prompt := theme.InputPrompt.Render(" ▸ ")
	content := e.textarea.View()

	return theme.InputBox.Render(prompt + content)
}

// ViewWithPrompt renders the input with a custom prompt prefix.
func (e InputEditor) ViewWithPrompt(theme *Theme, prompt string) string {
	p := theme.InputPrompt.Render(prompt)
	content := e.textarea.View()
	return theme.InputBox.Render(p + content)
}

// IsEmpty returns true if the input has no text.
func (e InputEditor) IsEmpty() bool {
	return strings.TrimSpace(e.textarea.Value()) == ""
}

// RenderInputArea renders the complete input area with prompt.
func RenderInputEditor(theme *Theme, editor InputEditor, working bool, width int) string {
	editor.SetWidth(width - 4) // Account for padding
	return editor.View(theme, working)
}

// StyledInput creates a cyberpunk-styled input.
func StyledInput(width int, theme *Theme) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.SetWidth(width)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false

	// Apply cyberpunk styling (Bubbles v2 uses Styles()/SetStyles())
	styles := ta.Styles()
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(TextDim)
	styles.Focused.Text = lipgloss.NewStyle().Foreground(TextPrimary)
	styles.Focused.CursorLine = lipgloss.NewStyle().Background(BorderColor)
	styles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(TextDim)
	styles.Blurred.Text = lipgloss.NewStyle().Foreground(TextPrimary)
	ta.SetStyles(styles)

	return ta
}
