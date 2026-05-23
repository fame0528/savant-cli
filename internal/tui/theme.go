// Package tui implements the Bubble Tea v2 terminal UI with cyberpunk neon aesthetics.
package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Neon-on-black color palette.
var (
	// Near-black primary background
	Background = color.RGBA{10, 10, 10, 255} // #0A0A0A

	// Surface/panel background
	Surface = color.RGBA{20, 20, 20, 255} // #141414

	// Border color
	BorderColor = color.RGBA{34, 34, 34, 255} // #222222

	// Primary text
	TextPrimary = color.RGBA{224, 224, 224, 255} // #E0E0E0

	// Dim/secondary text
	TextDim = color.RGBA{102, 102, 102, 255} // #666666

	// Neon pink - user messages, input prompt
	NeonPink = color.RGBA{255, 0, 255, 255} // #FF00FF

	// Neon cyan - assistant messages, info, status
	NeonCyan = color.RGBA{0, 255, 255, 255} // #00FFFF

	// Neon green - tool output, success
	NeonGreen = color.RGBA{0, 255, 65, 255} // #00FF41

	// Neon yellow - warnings, active items, thinking
	NeonYellow = color.RGBA{240, 255, 0, 255} // #F0FF00

	// Neon red - errors, critical
	NeonRed = color.RGBA{255, 0, 64, 255} // #FF0040

	// Neon orange - tool names, secondary accent
	NeonOrange = color.RGBA{255, 107, 53, 255} // #FF6B35

	// White - user message text
	White = color.RGBA{255, 255, 255, 255} // #FFFFFF
)

// Theme holds all styled components for the neon cyberpunk aesthetic.
type Theme struct {
	// Base
	Base lipgloss.Style

	// Title bar
	TitleBar       lipgloss.Style
	TitleLogo      lipgloss.Style
	TitleSep       lipgloss.Style
	ProviderBadgeLabel lipgloss.Style

	// Chat
	ChatHeader         lipgloss.Style
	UserMsgHeader      lipgloss.Style
	AssistantMsgHeader lipgloss.Style
	UserMessage        lipgloss.Style
	AssistantMessage   lipgloss.Style
	ToolMessage        lipgloss.Style
	ToolIcon           lipgloss.Style
	ToolName           lipgloss.Style
	ToolPanelHeader    lipgloss.Style
	SystemMessage      lipgloss.Style
	ThinkingMessage    lipgloss.Style

	// Input
	InputBox     lipgloss.Style
	InputPrompt  lipgloss.Style
	InputText    lipgloss.Style
	InputWorking lipgloss.Style
	Cursor       lipgloss.Style

	// Sidebar
	SidebarHeader lipgloss.Style
	TabActive     lipgloss.Style
	TabInactive   lipgloss.Style

	// Status
	StatusBar   lipgloss.Style
	StatusLabel lipgloss.Style
	StatusValue lipgloss.Style
	StatusSep   lipgloss.Style

	// Log
	LogHeader lipgloss.Style

	// General
	TextDim   lipgloss.Style
	TextMuted lipgloss.Style
	Warn      lipgloss.Style
	Error     lipgloss.Style
	Info      lipgloss.Style
	Success   lipgloss.Style
	HelpText  lipgloss.Style
	DividerLine lipgloss.Style

	// Dialog system
	Dialog      lipgloss.Style
	Button      lipgloss.Style
	ButtonFocus lipgloss.Style

	// Text aliases
	TextPrimary lipgloss.Style

	// Permission
	PermApprove lipgloss.Style
	PermDeny    lipgloss.Style

	// Tool box borders
	ToolBorderGreen  lipgloss.Style
	ToolBorderCyan   lipgloss.Style
	ToolBorderYellow lipgloss.Style
	ToolBorderOrange lipgloss.Style

	// Glitch frames for logo
	glitchFrames []string
}

// NewCyberpunkTheme creates the neon-on-black theme.
func NewCyberpunkTheme() *Theme {
	t := &Theme{
		Base: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(Background),

		TitleBar: lipgloss.NewStyle().
			Background(Surface).
			Padding(0, 1),

		TitleLogo: lipgloss.NewStyle().
			Foreground(NeonCyan).
			Bold(true).
			Background(Surface),

		TitleSep: lipgloss.NewStyle().
			Foreground(BorderColor).
			Background(Surface),

		ProviderBadgeLabel: lipgloss.NewStyle().
			Foreground(Background).
			Background(NeonCyan).
			Bold(true).
			Padding(0, 1),

		ChatHeader: lipgloss.NewStyle().
			Foreground(NeonCyan).
			Bold(true),

		// User messages: pink left border, white text
		UserMsgHeader: lipgloss.NewStyle().
			Foreground(White).
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(NeonPink).
			PaddingLeft(1),

		// Assistant messages: cyan left border, gray text
		AssistantMsgHeader: lipgloss.NewStyle().
			Foreground(NeonCyan).
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(NeonCyan).
			PaddingLeft(1),

		UserMessage: lipgloss.NewStyle().
			Foreground(White).
			PaddingLeft(4),

		AssistantMessage: lipgloss.NewStyle().
			Foreground(TextPrimary).
			PaddingLeft(4),

		ToolMessage: lipgloss.NewStyle().
			Foreground(TextPrimary).
			PaddingLeft(2),

		ToolIcon: lipgloss.NewStyle().
			Foreground(NeonOrange).
			Bold(true),

		ToolName: lipgloss.NewStyle().
			Foreground(NeonOrange).
			Bold(true),

		ToolPanelHeader: lipgloss.NewStyle().
			Foreground(NeonGreen).
			Bold(true),

		SystemMessage: lipgloss.NewStyle().
			Foreground(TextDim).
			Italic(true),

		ThinkingMessage: lipgloss.NewStyle().
			Foreground(NeonYellow).
			Italic(true),

		InputBox: lipgloss.NewStyle().
			Background(Surface).
			Padding(0, 1),

		InputPrompt: lipgloss.NewStyle().
			Foreground(NeonPink).
			Bold(true).
			Background(Surface),

		InputText: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(Surface),

		InputWorking: lipgloss.NewStyle().
			Foreground(NeonYellow).
			Background(Surface).
			Bold(true).
			Padding(0, 1),

		Cursor: lipgloss.NewStyle().
			Foreground(NeonCyan).
			Bold(true).
			Background(Surface),

		SidebarHeader: lipgloss.NewStyle().
			Foreground(NeonCyan).
			Bold(true),

		TabActive: lipgloss.NewStyle().
			Foreground(Background).
			Background(NeonCyan).
			Bold(true).
			Padding(0, 1),

		TabInactive: lipgloss.NewStyle().
			Foreground(TextDim).
			Background(Surface).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(Surface).
			Padding(0, 1),

		StatusLabel: lipgloss.NewStyle().
			Foreground(TextDim).
			Background(Surface),

		StatusValue: lipgloss.NewStyle().
			Foreground(NeonCyan).
			Bold(true).
			Background(Surface),

		StatusSep: lipgloss.NewStyle().
			Foreground(BorderColor).
			Background(Surface),

		LogHeader: lipgloss.NewStyle().
			Foreground(TextDim).
			Bold(true),

		TextDim: lipgloss.NewStyle().
			Foreground(TextDim),

		TextMuted: lipgloss.NewStyle().
			Foreground(color.RGBA{80, 80, 80, 255}),

		Warn: lipgloss.NewStyle().
			Foreground(NeonYellow),

		Error: lipgloss.NewStyle().
			Foreground(NeonRed).
			Bold(true),

		Info: lipgloss.NewStyle().
			Foreground(NeonCyan),

		Success: lipgloss.NewStyle().
			Foreground(NeonGreen).
			Bold(true),

		HelpText: lipgloss.NewStyle().
			Foreground(TextPrimary).
			PaddingLeft(2),

		DividerLine: lipgloss.NewStyle().
			Foreground(BorderColor),

		Dialog: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(Surface).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(NeonCyan).
			Padding(1, 2),

		Button: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(BorderColor).
			Padding(0, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(TextDim),

		ButtonFocus: lipgloss.NewStyle().
			Foreground(Background).
			Background(NeonCyan).
			Bold(true).
			Padding(0, 2),

		TextPrimary: lipgloss.NewStyle().
			Foreground(TextPrimary),

		PermApprove: lipgloss.NewStyle().
			Foreground(Background).
			Background(NeonGreen).
			Bold(true).
			Padding(0, 2),

		PermDeny: lipgloss.NewStyle().
			Foreground(Background).
			Background(NeonRed).
			Bold(true).
			Padding(0, 2),

		ToolBorderGreen: lipgloss.NewStyle().
			Foreground(NeonGreen),

		ToolBorderCyan: lipgloss.NewStyle().
			Foreground(NeonCyan),

		ToolBorderYellow: lipgloss.NewStyle().
			Foreground(NeonYellow),

		ToolBorderOrange: lipgloss.NewStyle().
			Foreground(NeonOrange),
	}

	// Glitch logo frames
	t.glitchFrames = []string{
		"███████╗ █████╗ ██╗   ██╗ █████╗ ███╗   ██╗████████╗",
		"██╔════╝██╔══██╗██║   ██║██╔══██╗████╗  ██║╚══██╔══╝",
		"███████╗███████║██║   ██║███████║██╔██╗ ██║   ██║   ",
		"╚════██║██╔══██║╚██╗ ██╔╝██╔══██║██║╚██╗██║   ██║   ",
		"███████║██║  ██║ ╚████╔╝ ██║  ██║██║ ╚████║   ██║   ",
		"╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚═╝  ╚═╝╚═╝  ╚═══╝   ╚═╝   ",
	}

	return t
}

// ProviderBadge returns a styled provider name badge.
func (t *Theme) ProviderBadge(name string) string {
	return t.ProviderBadgeStyle().Render(fmt.Sprintf(" ▲ %s ", strings.ToUpper(name)))
}

// ProviderBadgeStyle returns the badge style.
func (t *Theme) ProviderBadgeStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Background).
		Background(NeonCyan).
		Bold(true).
		Padding(0, 1)
}

// AnimatedSeparator returns a neon-styled separator line.
func (t *Theme) AnimatedSeparator(width int, tick int) string {
	if width <= 0 {
		return ""
	}

	pattern := "─│─"
	repeat := width/len(pattern) + 1
	full := strings.Repeat(pattern, repeat)

	offset := tick % len(pattern)
	if offset > 0 && offset < len(full) {
		full = full[offset:]
	}
	if len(full) > width {
		full = full[:width]
	}

	return t.TitleSep.Render(full)
}

// Divider returns a simple divider line.
func (t *Theme) Divider(width int) string {
	if width <= 0 {
		return ""
	}
	return t.DividerColor().Render(strings.Repeat("─", width))
}

func (t *Theme) DividerColor() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(BorderColor)
}

// Spinner returns an animated spinner character.
func (t *Theme) Spinner(frame int) string {
	frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	return t.Warn.Render(frames[frame%len(frames)])
}

// ToolBorder returns the appropriate border style for a tool type.
func (t *Theme) ToolBorder(toolName string) lipgloss.Style {
	switch toolName {
	case "bash":
		return t.ToolBorderGreen
	case "read":
		return t.ToolBorderCyan
	case "edit", "write":
		return t.ToolBorderOrange
	case "grep":
		return t.ToolBorderYellow
	case "glob":
		return t.ToolBorderGreen
	default:
		return t.ToolBorderGreen
	}
}
