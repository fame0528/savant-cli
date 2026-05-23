package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// DialogAction represents the result of a dialog interaction.
type DialogAction int

const (
	DialogNone DialogAction = iota
	DialogConfirm
	DialogCancel
	DialogSelect
)

// Dialog is an interface for overlay dialogs.
type Dialog interface {
	ID() string
	HandleKey(msg tea.KeyPressMsg) DialogAction
	Render(theme *Theme, width, height int) string
}

// DialogOverlay manages a stack of dialogs rendered as overlays.
type DialogOverlay struct {
	dialogs []Dialog
}

// NewDialogOverlay creates a new dialog overlay manager.
func NewDialogOverlay() *DialogOverlay {
	return &DialogOverlay{}
}

// Push adds a dialog to the top of the stack.
func (d *DialogOverlay) Push(dialog Dialog) {
	// Remove if already present (bring to front)
	for i, dl := range d.dialogs {
		if dl.ID() == dialog.ID() {
			d.dialogs = append(d.dialogs[:i], d.dialogs[i+1:]...)
			break
		}
	}
	d.dialogs = append(d.dialogs, dialog)
}

// Pop removes the top dialog.
func (d *DialogOverlay) Pop() {
	if len(d.dialogs) > 0 {
		d.dialogs = d.dialogs[:len(d.dialogs)-1]
	}
}

// Remove removes a dialog by ID.
func (d *DialogOverlay) Remove(id string) {
	for i, dl := range d.dialogs {
		if dl.ID() == id {
			d.dialogs = append(d.dialogs[:i], d.dialogs[i+1:]...)
			return
		}
	}
}

// HasDialog returns true if a dialog with the given ID is active.
func (d *DialogOverlay) HasDialog(id string) bool {
	for _, dl := range d.dialogs {
		if dl.ID() == id {
			return true
		}
	}
	return false
}

// IsEmpty returns true if no dialogs are active.
func (d *DialogOverlay) IsEmpty() bool {
	return len(d.dialogs) == 0
}

// Top returns the topmost dialog, or nil if empty.
func (d *DialogOverlay) Top() Dialog {
	if len(d.dialogs) == 0 {
		return nil
	}
	return d.dialogs[len(d.dialogs)-1]
}

// HandleKey sends a key event to the topmost dialog.
func (d *DialogOverlay) HandleKey(msg tea.KeyPressMsg) DialogAction {
	top := d.Top()
	if top == nil {
		return DialogNone
	}
	return top.HandleKey(msg)
}

// Render draws all active dialogs as overlays.
func (d *DialogOverlay) Render(theme *Theme, width, height int) string {
	if len(d.dialogs) == 0 {
		return ""
	}

	// Only render the topmost dialog
	top := d.Top()
	return top.Render(theme, width, height)
}

// ─────────────────────────────────────────────────────────────
// Confirm Dialog
// ─────────────────────────────────────────────────────────────

// ConfirmDialog is a simple yes/no confirmation dialog.
type ConfirmDialog struct {
	id       string
	title    string
	message  string
	selected int // 0 = yes, 1 = no
}

// NewConfirmDialog creates a confirmation dialog.
func NewConfirmDialog(id, title, message string) *ConfirmDialog {
	return &ConfirmDialog{
		id:      id,
		title:   title,
		message: message,
	}
}

func (d *ConfirmDialog) ID() string { return d.id }

func (d *ConfirmDialog) HandleKey(msg tea.KeyPressMsg) DialogAction {
	switch msg.String() {
	case "left", "h":
		d.selected = 0
	case "right", "l":
		d.selected = 1
	case "enter":
		if d.selected == 0 {
			return DialogConfirm
		}
		return DialogCancel
	case "esc", "ctrl+c":
		return DialogCancel
	case "y", "Y":
		return DialogConfirm
	case "n", "N":
		return DialogCancel
	}
	return DialogNone
}

func (d *ConfirmDialog) Render(theme *Theme, width, height int) string {
	// Dialog box
	dialogWidth := 50
	if dialogWidth > width-4 {
		dialogWidth = width - 4
	}

	// Title
	titleLine := theme.Info.Bold(true).Render(d.title)
	titlePadded := fmt.Sprintf("  %-*s", dialogWidth-6, titleLine)

	// Message
	msgLines := wrapText(d.message, dialogWidth-6)
	msgRendered := ""
	for _, line := range msgLines {
		msgRendered += theme.TextPrimary.Render("  " + line) + "\n"
	}

	// Buttons
	yesStyle := theme.Button
	noStyle := theme.Button
	if d.selected == 0 {
		yesStyle = theme.ButtonFocus
	} else {
		noStyle = theme.ButtonFocus
	}
	buttons := yesStyle.Render(" Yes ") + "  " + noStyle.Render(" No ")

	// Build dialog
	border := "╭" + strings.Repeat("─", dialogWidth-2) + "╮"
	footer := "╰" + strings.Repeat("─", dialogWidth-2) + "╯"

	var sb strings.Builder
	sb.WriteString(theme.Dialog.Render(border) + "\n")
	sb.WriteString(theme.Dialog.Render("│") + titlePadded + theme.Dialog.Render("│") + "\n")
	sb.WriteString(theme.Dialog.Render("│"+strings.Repeat(" ", dialogWidth-2)+"│") + "\n")
	for _, line := range msgLines {
		padded := fmt.Sprintf("  %-*s", dialogWidth-6, line)
		sb.WriteString(theme.Dialog.Render("│") + theme.TextPrimary.Render(padded) + theme.Dialog.Render("│") + "\n")
	}
	sb.WriteString(theme.Dialog.Render("│"+strings.Repeat(" ", dialogWidth-2)+"│") + "\n")
	sb.WriteString(theme.Dialog.Render("│") + centerText(buttons, dialogWidth-2) + theme.Dialog.Render("│") + "\n")
	sb.WriteString(theme.Dialog.Render(footer) + "\n")

	return sb.String()
}

// ─────────────────────────────────────────────────────────────
// List Dialog
// ─────────────────────────────────────────────────────────────

// ListDialog shows a list of options to select from.
type ListDialog struct {
	id       string
	title    string
	items    []string
	selected int
	scroll   int
	maxShow  int
}

// NewListDialog creates a list selection dialog.
func NewListDialog(id, title string, items []string) *ListDialog {
	return &ListDialog{
		id:      id,
		title:   title,
		items:   items,
		maxShow: 10,
	}
}

func (d *ListDialog) ID() string { return d.id }

func (d *ListDialog) HandleKey(msg tea.KeyPressMsg) DialogAction {
	switch msg.String() {
	case "up", "k":
		if d.selected > 0 {
			d.selected--
			if d.selected < d.scroll {
				d.scroll = d.selected
			}
		}
	case "down", "j":
		if d.selected < len(d.items)-1 {
			d.selected++
			if d.selected >= d.scroll+d.maxShow {
				d.scroll = d.selected - d.maxShow + 1
			}
		}
	case "enter":
		return DialogSelect
	case "esc", "ctrl+c":
		return DialogCancel
	}
	return DialogNone
}

func (d *ListDialog) SelectedIndex() int { return d.selected }

func (d *ListDialog) Render(theme *Theme, width, height int) string {
	dialogWidth := 50
	if dialogWidth > width-4 {
		dialogWidth = width - 4
	}

	// Title
	titleLine := theme.Info.Bold(true).Render(d.title)

	// Items
	end := d.scroll + d.maxShow
	if end > len(d.items) {
		end = len(d.items)
	}

	border := "╭" + strings.Repeat("─", dialogWidth-2) + "╮"
	footer := "╰" + strings.Repeat("─", dialogWidth-2) + "╯"

	var sb strings.Builder
	sb.WriteString(theme.Dialog.Render(border) + "\n")
	sb.WriteString(theme.Dialog.Render("│") + centerText(titleLine, dialogWidth-2) + theme.Dialog.Render("│") + "\n")
	sb.WriteString(theme.Dialog.Render("│"+strings.Repeat("─", dialogWidth-2)+"│") + "\n")

	for i := d.scroll; i < end; i++ {
		item := d.items[i]
		if len(item) > dialogWidth-8 {
			item = item[:dialogWidth-11] + "..."
		}

		var line string
		if i == d.selected {
			line = theme.Info.Bold(true).Render(" ▸ " + item)
		} else {
			line = theme.TextPrimary.Render("   " + item)
		}

		padded := fmt.Sprintf("%-*s", dialogWidth-2, stripAnsi(line))
		if i == d.selected {
			sb.WriteString(theme.Dialog.Render("│") + theme.Info.Bold(true).Render(padded) + theme.Dialog.Render("│") + "\n")
		} else {
			sb.WriteString(theme.Dialog.Render("│") + theme.TextPrimary.Render(padded) + theme.Dialog.Render("│") + "\n")
		}
	}

	sb.WriteString(theme.Dialog.Render(footer) + "\n")

	return sb.String()
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := ""
	for _, word := range words {
		if current == "" {
			current = word
		} else if len(current)+1+len(word) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func centerText(text string, width int) string {
	stripped := stripAnsi(text)
	padding := width - len(stripped)
	if padding <= 0 {
		return text
	}
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

func renderOverlay(bg, overlay string, theme *Theme) string {
	// Simple overlay: render the dialog on top of the background
	// In a real implementation, this would composite at specific coordinates
	return overlay
}

// DialogStyle returns the dialog style for a given position.
func DialogStyle(theme *Theme, width, height, dialogWidth, dialogHeight int) lipgloss.Style {
	// Center the dialog
	x := (width - dialogWidth) / 2
	y := (height - dialogHeight) / 2

	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(NeonCyan).
		Background(Surface).
		Foreground(TextPrimary).
		Padding(1, 2).
		Width(dialogWidth).
		MaxHeight(dialogHeight).
		MarginTop(y).
		MarginLeft(x)
}

// ─────────────────────────────────────────────────────────────
// Permission Dialog
// ─────────────────────────────────────────────────────────────

// PermissionDialog asks the user to approve or deny a tool execution.
type PermissionDialog struct {
	id       string
	toolName string
	command  string // e.g., the bash command or file path
	selected int    // 0 = approve, 1 = deny
}

// NewPermissionDialog creates a permission approval dialog.
func NewPermissionDialog(toolName, command string) *PermissionDialog {
	return &PermissionDialog{
		id:       "permission",
		toolName: toolName,
		command:  command,
	}
}

func (d *PermissionDialog) ID() string { return d.id }

func (d *PermissionDialog) HandleKey(msg tea.KeyPressMsg) DialogAction {
	switch msg.String() {
	case "left", "h":
		d.selected = 0
	case "right", "l":
		d.selected = 1
	case "enter":
		if d.selected == 0 {
			return DialogConfirm
		}
		return DialogCancel
	case "esc", "ctrl+c", "n", "N":
		return DialogCancel
	case "y", "Y":
		return DialogConfirm
	case "a", "A":
		d.selected = 0
		return DialogConfirm
	case "d", "D":
		d.selected = 1
		return DialogCancel
	}
	return DialogNone
}

// Approved returns true if the user approved the action.
func (d *PermissionDialog) Approved() bool {
	return d.selected == 0
}

func (d *PermissionDialog) Render(theme *Theme, width, height int) string {
	dialogWidth := 55
	if dialogWidth > width-4 {
		dialogWidth = width - 4
	}

	// Build the dialog
	var sb strings.Builder

	// Header with neon yellow warning
	header := theme.Warn.Bold(true).Render("⚠ PERMISSION REQUIRED")
	sb.WriteString(theme.Dialog.Render("╭"+strings.Repeat("─", dialogWidth-2)+"╮") + "\n")
	sb.WriteString(theme.Dialog.Render("│") + centerText(header, dialogWidth-2) + theme.Dialog.Render("│") + "\n")
	sb.WriteString(theme.Dialog.Render("│"+strings.Repeat("─", dialogWidth-2)+"│") + "\n")

	// Tool name in neon orange
	toolLine := theme.ToolName.Render("  Tool: " + d.toolName)
	sb.WriteString(theme.Dialog.Render("│") + fmt.Sprintf("%-*s", dialogWidth-2, stripAnsi(toolLine)) + theme.Dialog.Render("│") + "\n")

	// Command/path in dim text
	cmdLine := d.command
	if utf8.RuneCountInString(cmdLine) > dialogWidth-8 {
		runes := []rune(cmdLine)
		cmdLine = string(runes[:dialogWidth-11]) + "..."
	}
	sb.WriteString(theme.Dialog.Render("│") + theme.TextDim.Render(fmt.Sprintf("  %-*s", dialogWidth-4, cmdLine)) + theme.Dialog.Render("│") + "\n")

	sb.WriteString(theme.Dialog.Render("│"+strings.Repeat(" ", dialogWidth-2)+"│") + "\n")

	// Buttons: Approve (green) / Deny (red)
	approveBtn := theme.PermApprove.Render(" [A] Approve ")
	denyBtn := theme.PermDeny.Render(" [D] Deny ")

	if d.selected == 0 {
		approveBtn = theme.PermApprove.Bold(true).Render(" ▸ [A] Approve ")
	} else {
		denyBtn = theme.PermDeny.Bold(true).Render(" ▸ [D] Deny ")
	}

	buttons := approveBtn + "  " + denyBtn
	sb.WriteString(theme.Dialog.Render("│") + centerText(buttons, dialogWidth-2) + theme.Dialog.Render("│") + "\n")
	sb.WriteString(theme.Dialog.Render("╰"+strings.Repeat("─", dialogWidth-2)+"╯") + "\n")

	return sb.String()
}
