package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Logo frames for animated Savant ASCII art.
// Frame 0: dark/off state
// Frames 1-6: progressive power-on (each letter lights up)
// Frame 7: fully lit steady state
// Frames 8-10: glitch frames
// Frames 11-13: breathing/pulse frames

var savantFrames = []string{
	// Frame 0 - All dark (dim dots)
	`
   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
   ░  ░░░░░░          ░     ░░░░░░░░  ░
   ░  ░    ░  ░░  ░░  ░  ░  ░     ░   ░
   ░  ░░░░░   ░  ░   ░  ░   ░░░░░    ░
   ░       ░  ░░  ░░  ░  ░       ░   ░
   ░  ░░░░░  ░       ░  ░   ░░░░░    ░
   ░                                ░
   ░  Terminal-Native AI Assistant  ░
   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░`,

	// Frame 1 - S lights up
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓          ░     ░░░░░░░░  ║
   ║  ▓    ▓  ░░  ░░  ░  ░  ░     ░   ║
   ║  ▓▓▓▓▓   ░  ░   ░  ░   ░░░░░    ║
   ║       ▓  ░░  ░░  ░  ░       ░   ║
   ║  ▓▓▓▓▓  ░       ░  ░   ░░░░░    ║
   ║                                ║
   ║  Terminal-Native AI Assistant  ║
   ╚═══════════════════════════════════╝`,

	// Frame 2 - SA lights up
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓  ▓▓▓▓▓      ░     ░░░░░  ║
   ║  ▓    ▓  ▓    ▓  ░░  ░  ░     ░  ║
   ║  ▓▓▓▓▓   ▓▓▓▓▓    ░  ░   ░░░░   ║
   ║       ▓        ▓  ░░  ░       ░  ║
   ║  ▓▓▓▓▓  ▓▓▓▓▓▓   ░       ░░░░   ║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,

	// Frame 3 - SAV lights up
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓  ▓▓▓▓▓  ▓   ▓     ░░░░░  ║
   ║  ▓    ▓  ▓    ▓ ▓   ▓  ░  ░     ░║
   ║  ▓▓▓▓▓   ▓▓▓▓▓   ▓▓▓    ░  ░░░░ ║
   ║       ▓        ▓    ▓   ░  ░     ║
   ║  ▓▓▓▓▓  ▓▓▓▓▓▓  ▓▓▓    ░  ░░░░  ║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,

	// Frame 4 - SAVA lights up
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓  ▓▓▓▓▓  ▓   ▓  ▓▓▓     ░║
   ║  ▓    ▓  ▓    ▓ ▓   ▓  ▓  ▓  ░  ║
   ║  ▓▓▓▓▓   ▓▓▓▓▓   ▓▓▓   ▓▓▓    ░ ║
   ║       ▓        ▓    ▓  ▓  ▓  ░  ║
   ║  ▓▓▓▓▓  ▓▓▓▓▓▓  ▓▓▓   ▓▓▓   ░  ║
   ║                                  ║
   ║  Terminal-Native AI Assistant    ║
   ╚═══════════════════════════════════╝`,

	// Frame 5 - SAVAN lights up
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓  ▓▓▓▓▓  ▓   ▓  ▓▓▓  ▓  ░║
   ║  ▓    ▓  ▓    ▓ ▓   ▓  ▓  ▓ ▓▓ ░║
   ║  ▓▓▓▓▓   ▓▓▓▓▓   ▓▓▓   ▓▓▓  ▓▓ ║
   ║       ▓        ▓    ▓  ▓  ▓ ▓ ▓ ║
   ║  ▓▓▓▓▓  ▓▓▓▓▓▓  ▓▓▓   ▓▓▓  ▓  ▓║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,

	// Frame 6 - SAVANT fully lit!
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓  ▓▓▓▓▓  ▓   ▓  ▓▓▓  ▓ ▓ ║
   ║  ▓    ▓  ▓    ▓ ▓   ▓  ▓  ▓ ▓▓▓ ║
   ║  ▓▓▓▓▓   ▓▓▓▓▓   ▓▓▓   ▓▓▓  ▓▓▓ ║
   ║       ▓        ▓    ▓  ▓  ▓ ▓ ▓ ║
   ║  ▓▓▓▓▓  ▓▓▓▓▓▓  ▓▓▓   ▓▓▓  ▓ ▓ ║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,

	// Frame 7 - Steady state (bright)
	`
   ╔═══════════════════════════════════╗
   ║  █▀▀▀▀  █▀▀▀█  █   █  █▀▀█  █ █ ║
   ║  █    █  █    █ █   █  █  █  █▀█ ║
   ║  █▀▀▀▀   █▀▀▀█   █▀▀   █▀▀█  █▀█ ║
   ║       █        █    █  █  █ █ █ ║
   ║  █▀▀▀▀  █▀▀▀▀▀  █▀▀   █▀▀█  █ █ ║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,
}

// Glitch frames - distortions of the fully-lit logo
var glitchFrames = []string{
	// Glitch 1 - horizontal shift
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓  ▓▓▓▓▓  ▓   ▓  ▓▓▓  ▓ ▓ ║
   ║  ▓    ▓  ▓    ▓ ▓   ▓  ▓  ▓ ▓▓▓ ║
   ║  ▓▓▓▓▓   ▓▓▓▓▓   ▓▓▓   ▓▓▓  ▓▓▓ ║
   ║       ▓        ▓    ▓  ▓  ▓ ▓ ▓ ║
   ║  ▓▓▓▓▓  ▓▓▓▓▓▓  ▓▓▓   ▓▓▓  ▓ ▓ ║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,

	// Glitch 2 - color inversion
	`
   ╔═══════════════════════════════════╗
   ║  ░░░░░  ░░░░░  ░   ░  ░░░  ░ ░ ║
   ║  ░    ░  ░    ░ ░   ░  ░  ░  ░░░ ║
   ║  ░░░░░   ░░░░░   ░░░   ░░░  ░░░ ║
   ║       ░        ░    ░  ░  ░ ░ ░ ║
   ║  ░░░░░  ░░░░░░  ░░░   ░░░  ░ ░ ║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,

	// Glitch 3 - scanline effect
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓  ▓▓▓▓▓  ▓   ▓  ▓▓▓  ▓ ▓ ║
   ║───────────────────────────────────║
   ║  ▓▓▓▓▓   ▓▓▓▓▓   ▓▓▓   ▓▓▓  ▓▓▓ ║
   ║       ▓        ▓    ▓  ▓  ▓ ▓ ▓ ║
   ║───────────────────────────────────║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,
}

// Breathing frames - subtle brightness pulsing
var breatheFrames = []string{
	// Breathe 1 - slightly dim
	`
   ╔═══════════════════════════════════╗
   ║  ▒▒▒▒▒  ▒▒▒▒▒  ▒   ▒  ▒▒▒  ▒ ▒ ║
   ║  ▒    ▒  ▒    ▒ ▒   ▒  ▒  ▒  ▒▒▒ ║
   ║  ▒▒▒▒▒   ▒▒▒▒▒   ▒▒▒   ▒▒▒  ▒▒▒ ║
   ║       ▒        ▒    ▒  ▒  ▒ ▒ ▒ ║
   ║  ▒▒▒▒▒  ▒▒▒▒▒▒  ▒▒▒   ▒▒▒  ▒ ▒ ║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,

	// Breathe 2 - full bright
	`
   ╔═══════════════════════════════════╗
   ║  ▓▓▓▓▓  ▓▓▓▓▓  ▓   ▓  ▓▓▓  ▓ ▓ ║
   ║  ▓    ▓  ▓    ▓ ▓   ▓  ▓  ▓  ▓▓▓ ║
   ║  ▓▓▓▓▓   ▓▓▓▓▓   ▓▓▓   ▓▓▓  ▓▓▓ ║
   ║       ▓        ▓    ▓  ▓  ▓ ▓ ▓ ║
   ║  ▓▓▓▓▓  ▓▓▓▓▓▓  ▓▓▓   ▓▓▓  ▓ ▓ ║
   ║                                   ║
   ║  Terminal-Native AI Assistant     ║
   ╚═══════════════════════════════════╝`,
}

// Total frames: boot (8) + glitch (3) + breathe (2) = 13
const (
	logoFrameCount = 13
	logoBootFrames = 8
	logoGlitchStart = 8
	logoBreatheStart = 11
)

// GetAnimatedLogo returns the current animation frame with cyberpunk styling.
// frame is incremented externally (e.g., every 200ms).
// phase: 0=boot, 1=steady+glitch, 2=breathe
func GetAnimatedLogo(frame int, theme *Theme) string {
	totalFrames := len(savantFrames) + len(glitchFrames) + len(breatheFrames)
	f := frame % totalFrames

	var raw string
	var style lipgloss.Style

	switch {
	case f < len(savantFrames):
		// Boot sequence - cyan for early, green for fully lit
		raw = savantFrames[f]
		if f < 6 {
			style = theme.Info // HyperCyan
		} else {
			style = theme.Success // Green
		}
	case f < len(savantFrames)+len(glitchFrames):
		// Glitch frames - orange/red
		gi := f - len(savantFrames)
		raw = glitchFrames[gi]
		if gi == 1 {
			style = theme.Error // Inverted
		} else {
			style = theme.Warn // Orange
		}
	default:
		// Breathe frames - cyan pulse
		bi := f - len(savantFrames) - len(glitchFrames)
		raw = breatheFrames[bi]
		style = theme.Info
	}

	return style.Render(raw)
}

// GetLogo returns the appropriate static logo size for the terminal width.
func GetLogo(width int) string {
	if width >= 100 {
		return SavantLogo
	}
	if width >= 55 {
		return SavantLogoMedium
	}
	return SavantLogoCompact
}

// SavantLogo is the full ASCII art logo from docs/savant_ascii.md.
const SavantLogo = `                           =
                           --×÷=                                   =÷×--
                            --××÷÷=                                =÷÷××--
                            ---××××÷==                          ==÷××××---
                            ----××÷÷÷÷÷=                     =÷÷×××××----
                            -----××÷÷÷÷÷÷==               ==÷×××××××-----
                            ------×××××××÷÷÷=           =÷÷××××××××------
                            --------××××÷÷÷÷÷÷=≠     =÷÷××××××××--------`

// SavantLogoCompact is a shorter version for smaller terminals.
const SavantLogoCompact = ` ╔═══════════════════════════════════════╗
 ║   ____                _              ║
 ║  / ___|  __ _ _ __  | |_ ___ _ __   ║
 ║  \___ \ / _' | '_ \ | __/ _ \ '__|  ║
 ║   ___) | (_| | | | || ||  __/ |     ║
 ║  |____/ \__,_|_| |_| \__\___|_|     ║
 ║                                      ║
 ║  Terminal-Native AI Coding Assistant  ║
 ╚═══════════════════════════════════════╝`

// SavantLogoMedium is a medium-size version.
const SavantLogoMedium = `   ╔════════════════════════════════════════════════╗
   ║   ____                  _                      ║
   ║  / ___|  __ _ _   _ ___| |_ __ _ _ __   ___   ║
   ║  \___ \ / _' | | | / __| __/ _' | '_ \ / _ \  ║
   ║   ___) | (_| | |_| \__ \ || (_| | | | |  __/  ║
   ║  |____/ \__,_|\__, |___/\__\__,_|_| |_|\___|  ║
   ║                |___/                           ║
   ║                                                ║
   ║       Terminal-Native AI Coding Assistant       ║
   ╚════════════════════════════════════════════════╝`

// SavantLogoFull is the large cyberpunk ASCII art logo.
const SavantLogoFull = `                           =
                           --×÷=                                   =÷×--
                            --××÷÷=                                =÷÷××--
                            ---××××÷==                          ==÷××××---
                            ----××÷÷÷÷÷=                     =÷÷×××××----
                            -----××÷÷÷÷÷÷==               ==÷×××××××-----
                            ------×××××××÷÷÷=           =÷÷××××××××------
                            --------××××÷÷÷÷÷÷=≠     =÷÷××××××××--------`

// renderLogoWithNeonGlow renders a logo line with neon glow effect.
func renderLogoWithNeonGlow(line string, baseColor, glowColor interface{}) string {
	// Add glow by rendering the same line twice with slight offset
	// This creates a bloom effect in terminals that support it
	return strings.ReplaceAll(line, "▓", "█")
}
