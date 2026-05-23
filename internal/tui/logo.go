package tui

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

// GetLogo returns the appropriate logo size for the terminal width.
func GetLogo(width int) string {
	if width >= 100 {
		return SavantLogo
	}
	if width >= 55 {
		return SavantLogoMedium
	}
	return SavantLogoCompact
}
