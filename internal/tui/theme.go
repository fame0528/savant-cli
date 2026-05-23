// Package tui implements the Bubble Tea v2 terminal UI with cyberpunk aesthetics.
package tui

import (
	"image/color"

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

	// Highlight background
	HighlightBg = color.RGBA{0, 240, 255, 30} // HyperCyan at 12%
)

// Theme holds all styled components for the cyberpunk aesthetic.
type Theme struct {
	// Base styles
	Base lipgloss.Style

	// Chat styles
	UserMessage      lipgloss.Style
	AssistantMessage lipgloss.Style
	ToolMessage      lipgloss.Style
	SystemMessage    lipgloss.Style

	// UI component styles
	Title       lipgloss.Style
	StatusBar   lipgloss.Style
	Border      lipgloss.Style
	Sidebar     lipgloss.Style
	Dialog      lipgloss.Style
	Input       lipgloss.Style
	Button      lipgloss.Style
	ButtonFocus lipgloss.Style

	// Status indicators
	Success lipgloss.Style
	Error   lipgloss.Style
	Warn    lipgloss.Style
	Info    lipgloss.Style

	// Markdown/rendering
	Code      lipgloss.Style
	CodeBlock lipgloss.Style
	Link      lipgloss.Style
}

// NewCyberpunkTheme creates the cyberpunk theme.
func NewCyberpunkTheme() *Theme {
	return &Theme{
		Base: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(VoidIndigo),

		UserMessage: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Bold(true).
			PaddingLeft(1),

		AssistantMessage: lipgloss.NewStyle().
			Foreground(TextPrimary).
			PaddingLeft(1),

		ToolMessage: lipgloss.NewStyle().
			Foreground(TextDim).
			Italic(true).
			PaddingLeft(2),

		SystemMessage: lipgloss.NewStyle().
			Foreground(SolarOrange).
			Bold(true),

		Title: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Bold(true).
			Background(BorderColor).
			Padding(0, 2),

		StatusBar: lipgloss.NewStyle().
			Foreground(TextDim).
			Background(BorderColor).
			Padding(0, 1),

		Border: lipgloss.NewStyle().
			Foreground(BorderColor),

		Sidebar: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(PanelBg).
			Border(lipgloss.NormalBorder()).
			BorderForeground(BorderColor).
			Padding(1),

		Dialog: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(PanelBg).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(HyperCyan).
			Padding(1, 2),

		Input: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Background(BorderColor).
			Padding(0, 1),

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

		Success: lipgloss.NewStyle().
			Foreground(SuccessGreen).
			Bold(true),

		Error: lipgloss.NewStyle().
			Foreground(ErrorRed).
			Bold(true),

		Warn: lipgloss.NewStyle().
			Foreground(SolarOrange),

		Info: lipgloss.NewStyle().
			Foreground(HyperCyan),

		Code: lipgloss.NewStyle().
			Foreground(SolarOrange).
			Background(BorderColor).
			Padding(0, 1),

		CodeBlock: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(PanelBg).
			Border(lipgloss.NormalBorder()).
			BorderForeground(BorderColor).
			Padding(1),

		Link: lipgloss.NewStyle().
			Foreground(HyperCyan).
			Underline(true),
	}
}

// Logo returns the Savant ASCII art logo.
func (t *Theme) Logo() string {
	return t.Title.Render(`
 ╔═══════════════════════════════════════╗
 ║   ____                _              ║
 ║  / ___|  __ _ _ __  | |_ ___ _ __   ║
 ║  \___ \ / _' | '_ \ | __/ _ \ '__|  ║
 ║   ___) | (_| | | | || ||  __/ |     ║
 ║  |____/ \__,_|_| |_| \__\___|_|     ║
 ║                                      ║
 ║  Terminal-Native AI Coding Assistant  ║
 ╚═══════════════════════════════════════╝`)
}

// ScanlineOverlay returns a scanline effect string for idle animations.
func (t *Theme) ScanlineOverlay(width int) string {
	line := lipgloss.NewStyle().
		Foreground(color.RGBA{255, 255, 255, 8}).
		Render("─")
	result := ""
	for i := 0; i < width; i++ {
		result += line
	}
	return result
}
