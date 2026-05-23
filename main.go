package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/spenc/savant-cli/internal/config"
	"github.com/spenc/savant-cli/internal/provider"
	"github.com/spenc/savant-cli/internal/tools"
	"github.com/spenc/savant-cli/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse flags
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "--version", "-v":
			fmt.Printf("savant %s (%s) built %s\n", version, commit, date)
			return
		case "--help", "-h":
			printUsage()
			return
		}
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Build provider chain
	var providers []provider.Provider
	for _, pc := range cfg.Providers {
		if !pc.Enabled {
			continue
		}
		switch pc.Name {
		case "9router":
			providers = append(providers, provider.NewNineRouterProvider(pc.BaseURL))
		default:
			providers = append(providers, provider.NewOpenAIProvider(pc.Name, pc.BaseURL, pc.APIKey, pc.Model))
		}
	}

	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no providers configured")
		os.Exit(1)
	}

	// Create router and select provider
	router := provider.NewRouter(providers, cfg.SmartRouting)
	ctx := context.Background()
	selected := router.Select(ctx)

	// Create tool registry
	registry := tools.NewRegistry()

	// Create and run TUI
	model := tui.New(selected, registry, cfg.MaxTurns)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`savant - Terminal-Native AI Coding Assistant

Usage:
  savant              Start interactive TUI
  savant --version    Show version
  savant --help       Show this help

Configuration:
  ~/.savant/config.json    Configuration file

Environment Variables:
  OPENAI_API_KEY           API key for OpenAI-compatible providers
  ANTHROPIC_API_KEY        API key for Anthropic provider
  OLLAMA_HOST              Ollama server URL (default: http://localhost:11434)
  NINEROUTER_URL           9router gateway URL (default: http://localhost:20128)

Commands (in TUI):
  /help           Show help
  /provider       Configure providers
  /model          Switch model
  /session        Session management
  /config         View/edit config
  /quit           Exit

Keybindings:
  Enter           Send message
  Ctrl+C          Cancel / Quit
  Up/Down         Scroll chat
  Left/Right      Move cursor
  Home/End        Jump to start/end of input
`)
}
