package tui

import (
	"charm.land/lipgloss/v2"
)

// The Savant header art - user-created block characters that clearly spell S-A-V-A-N-T.
const savantHeader = `▄█████ ▄████▄ ██  ██ ▄████▄ ███  ██ ██████
▀▀▀▄▄▄ ██▄▄██ ██▄▄██ ██▄▄██ ██ ▀▄██   ██
█████▀ ██  ██  ▀██▀  ██  ██ ██   ██   ██`

// bootFrame returns the header art for a boot animation frame.
// Plays once on startup then disappears.
// Frame 0: dim (just appeared)
// Frame 1: cyan (powering on)
// Frame 2: bright cyan (fully lit)
// Frame 3: green (ready)
// Frame 4: fade out (boot complete)
// Frame 5+: empty (logo gone, show welcome text)
func bootFrame(frame int, theme *Theme) string {
	if frame > 4 {
		return ""
	}

	var style lipgloss.Style
	switch frame {
	case 0:
		style = theme.TextDim
	case 1:
		style = lipgloss.NewStyle().Foreground(NeonCyan)
	case 2:
		style = lipgloss.NewStyle().Foreground(NeonCyan).Bold(true)
	case 3:
		style = lipgloss.NewStyle().Foreground(NeonGreen).Bold(true)
	case 4:
		style = theme.TextDim
	}

	return style.Render(savantHeader)
}
