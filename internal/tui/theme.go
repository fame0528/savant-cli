// Package tui implements the Bubble Tea v2 terminal UI with cyberpunk aesthetics.
package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Cyberpunk color palette from the architecture doc.
var (
	// Void Indigo - primary background
	VoidIndigo = color.RGBA{13, 2, 33, 255} // #0D0221

	// HyperCyan - primary accent
	HyperCyan = color.RGBA{0, 240, 255, 255} // #00F0FF

	// SolarOrange - secondary accent
	SolarOrange = color.RGBA{255, 107, 53, 255} // #FF6B35

	// Text color
	TextPrimary = color.RGBA{224, 224, 224, 255} // #E0E0E0

	// Border color
	BorderColor = color.RGBA{26, 26, 46, 255} // #1A1A2E

	// Success green
	SuccessGreen = color.RGBA{0, 255, 65, 255} // #00FF41

	// Error red
	ErrorRed = color.RGBA{255, 0, 64, 255} // #FF0040

	// Dim text
	TextDim = color.RGBA{128, 128, 160, 255} // #8080A0

	// Panel background
	PanelBg = color.RGBA{15, 5, 40, 255} // #0F0528

	// Deep panel
	DeepPanelBg = color.RGBA{10, 2, 28, 255} // #0A021C

	// Highlight background
	HighlightBg = color.RGBA{0, 240, 255, 30} // HyperCyan at 12%

	// Neon purple (for accents)
	NeonPurple = color.RGBA{170, 0, 255, 255} // #AA00FF

	// Neon pink
	NeonPink = color.RGBA{255, 0, 170, 255} // #FF00AA
)

// Theme holds all styled components for the cyberpunk aesthetic.
type Theme struct {
	// Base
	Base lipgloss.Style

	// Title bar
	TitleBar       lipgloss.Style
	TitleLogo      lipgloss.Style
	TitleSep       lipgloss.Style
	ProviderBadgeLabel  lipgloss.Style

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
	StatusBar lipgloss.Style

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
	DividerLine   lipgloss.Style

	// Dialog system
	Dialog      lipgloss.Style
	Button      lipgloss.Style
	ButtonFocus lipgloss.Style

	// Text aliases
	TextPrimary lipgloss.Style

	// Glitch frames for logo
	glitchFrames []string
}

// NewCyberpunkTheme creates the cyberpunk theme with all styles.
func NewCyberpunkTheme() *Theme {
	t := &Theme{
		Base: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(VoidIndigo),

		TitleBar: lipgloss.NewStyle().
			Background(BorderColor).
			Padding(0, 1),

		TitleLogo: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Bold(true).
			Background(BorderColor),

		TitleSep: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Background(BorderColor),

		ProviderBadgeLabel: lipgloss.NewStyle().
			Foreground(VoidIndigo).
			Background(HyperCyan).
			Bold(true).
			Padding(0, 1),

		ChatHeader: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Bold(true),

		UserMsgHeader: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(HyperCyan).
			PaddingLeft(1),

		AssistantMsgHeader: lipgloss.NewStyle().
			Foreground(SolarOrange).
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(SolarOrange).
			PaddingLeft(1),

		UserMessage: lipgloss.NewStyle().
			Foreground(TextPrimary).
			PaddingLeft(4),

		AssistantMessage: lipgloss.NewStyle().
			Foreground(TextPrimary).
			PaddingLeft(4),

		ToolMessage: lipgloss.NewStyle().
			Foreground(TextDim).
			Italic(true).
			PaddingLeft(2),

		ToolIcon: lipgloss.NewStyle().
			Foreground(SolarOrange).
			Bold(true),

		ToolName: lipgloss.NewStyle().
			Foreground(NeonPurple).
			Bold(true),

		ToolPanelHeader: lipgloss.NewStyle().
			Foreground(SolarOrange).
			Bold(true),

		SystemMessage: lipgloss.NewStyle().
			Foreground(SolarOrange).
			Bold(true),

		InputBox: lipgloss.NewStyle().
			Background(BorderColor).
			Padding(0, 1),

		InputPrompt: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Bold(true).
			Background(BorderColor),

		InputText: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(BorderColor),

		InputWorking: lipgloss.NewStyle().
			Foreground(SolarOrange).
			Background(BorderColor).
			Bold(true).
			Padding(0, 1),

		Cursor: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Bold(true).
			Background(BorderColor),

		SidebarHeader: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Bold(true),

		TabActive: lipgloss.NewStyle().
			Foreground(VoidIndigo).
			Background(HyperCyan).
			Bold(true).
			Padding(0, 1),

		TabInactive: lipgloss.NewStyle().
			Foreground(TextDim).
			Background(PanelBg).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(BorderColor).
			Padding(0, 1),

		LogHeader: lipgloss.NewStyle().
			Foreground(TextDim).
			Bold(true),

		TextDim: lipgloss.NewStyle().
			Foreground(TextDim),

		TextMuted: lipgloss.NewStyle().
			Foreground(color.RGBA{96, 96, 120, 255}),

		Warn: lipgloss.NewStyle().
			Foreground(SolarOrange),

		Error: lipgloss.NewStyle().
			Foreground(ErrorRed).
			Bold(true),

		Info: lipgloss.NewStyle().
			Foreground(HyperCyan),

		Success: lipgloss.NewStyle().
			Foreground(SuccessGreen).
			Bold(true),

		HelpText: lipgloss.NewStyle().
			Foreground(TextPrimary).
			PaddingLeft(2),

		DividerLine: lipgloss.NewStyle().
			Foreground(BorderColor),

		Dialog: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(PanelBg).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(HyperCyan).
			Padding(1, 2),

		Button: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(BorderColor).
			Padding(0, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(TextDim),

		ButtonFocus: lipgloss.NewStyle().
			Foreground(VoidIndigo).
			Background(HyperCyan).
			Bold(true).
			Padding(0, 2),

		TextPrimary: lipgloss.NewStyle().
			Foreground(TextPrimary),
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

// Logo returns the full Savant ASCII logo.
func (t *Theme) Logo() string {
	return t.TitleLogo.Render(strings.Join(t.glitchFrames, "\n"))
}

// GlitchLogo returns the logo with optional glitch effect.
func (t *Theme) GlitchLogo(frame int, glitch bool) string {
	logo := GetLogo(100) // Use full-size for title bar
	if !glitch {
		return t.TitleLogo.Render(logo)
	}

	// Glitch: shift random characters on one line
	lines := strings.Split(logo, "\n")
	glitched := make([]string, len(lines))
	copy(glitched, lines)

	idx := frame % len(lines)
	if idx < len(glitched) && len(glitched[idx]) > 2 {
		glitched[idx] = " " + glitched[idx][:len(glitched[idx])-1]
	}

	return t.TitleLogo.Render(strings.Join(glitched, "\n"))
}

// ProviderBadge returns a styled provider name badge.
func (t *Theme) ProviderBadge(name string) string {
	return t.ProviderBadgeStyle().Render(fmt.Sprintf(" ▲ %s ", strings.ToUpper(name)))
}

// ProviderBadgeStyle returns the badge style.
func (t *Theme) ProviderBadgeStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(VoidIndigo).
		Background(HyperCyan).
		Bold(true).
		Padding(0, 1)
}

// AnimatedSeparator returns a cyberpunk-styled separator line.
func (t *Theme) AnimatedSeparator(width int, tick int) string {
	if width <= 0 {
		return ""
	}

	// Animated pattern
	pattern := "═╪═"
	repeat := width/len(pattern) + 1
	full := strings.Repeat(pattern, repeat)

	// Animate: shift by tick
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
	return t.Info.Render(frames[frame%len(frames)])
}
