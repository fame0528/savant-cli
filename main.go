package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "charm.land/bubbletea/v2"

	"github.com/spenc/savant-cli/internal/commands"
	"github.com/spenc/savant-cli/internal/config"
	"github.com/spenc/savant-cli/internal/db"
	"github.com/spenc/savant-cli/internal/pet"
	"github.com/spenc/savant-cli/internal/provider"
	"github.com/spenc/savant-cli/internal/session"
	"github.com/spenc/savant-cli/internal/skills"
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

	// Open database
	database, err := db.Open(config.DBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create session service
	sessionSvc := session.NewService(database)

	// Skills directory
	skillsDir := filepath.Join(config.ConfigDir(), "skills")
	os.MkdirAll(skillsDir, 0o755)

	// Usage telemetry store
	usageStore := skills.NewUsageStore(skillsDir)

	// Load or create pet
	petDir := config.ConfigDir()
	petObj := pet.LoadPet(petDir)
	if petObj == nil {
		petObj = pet.NewPet("Byte")
	}

	// Save pet on exit
	defer func() {
		if err := petObj.Save(petDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save pet: %v\n", err)
		}
	}()

	// Signal handler for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		petObj.Save(petDir)
		usageStore.SaveExternal()
		os.Exit(0)
	}()

	// Build provider chain
	providers := buildProviders(cfg)
	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no providers configured")
		os.Exit(1)
	}

	// Create router and select provider
	router := provider.NewRouter(providers, cfg.SmartRouting)
	ctx := context.Background()
	selected := router.Select(ctx)

	// Create tool registry (includes skill_manage tool)
	registry := tools.NewRegistry(skillsDir)

	// Create command registry
	cmdReg := commands.NewRegistry()
	cmdReg.RegisterPet(
		petObj.Feed,
		petObj.Play,
		petObj.Rest,
		petObj.Heal,
		petObj.Stats,
	)
	cmdReg.RegisterConfigReal(cfg)

	// Start background Curator (lifecycle management)
	curatorConfig := skills.DefaultCuratorConfig()
	curator := skills.NewCurator(curatorConfig, skillsDir, usageStore)
	if curator.ShouldRunNow() {
		go func() {
			consolidateFn := func(ctx context.Context, prompt string) (string, error) {
				resp, err := selected.Chat(ctx, provider.ChatRequest{
					Messages: []provider.ChatMessage{
						{Role: "system", Content: "You are a skill curator. Analyze the skills and consolidate them."},
						{Role: "user", Content: prompt},
					},
				})
				if err != nil {
					return "", err
				}
				if len(resp.Choices) > 0 {
					return resp.Choices[0].Message.Content, nil
				}
				return "", nil
			}
			if err := curator.Run(context.Background(), consolidateFn); err != nil {
				fmt.Fprintf(os.Stderr, "Curator warning: %v\n", err)
			}
		}()
	}

	// Start background Extraction (session analysis)
	extractionConfig := skills.DefaultExtractionConfig()
	extractor := skills.NewExtractor(extractionConfig, skillsDir, sessionSvc, usageStore)
	if extractor.ShouldRunNow() {
		go func() {
			extractFn := func(ctx context.Context, prompt string) (string, error) {
				resp, err := selected.Chat(ctx, provider.ChatRequest{
					Messages: []provider.ChatMessage{
						{Role: "system", Content: "You are a skill extraction agent. Analyze session transcripts and extract reusable skills."},
						{Role: "user", Content: prompt},
					},
				})
				if err != nil {
					return "", err
				}
				if len(resp.Choices) > 0 {
					return resp.Choices[0].Message.Content, nil
				}
				return "", nil
			}
			if _, err := extractor.Run(context.Background(), extractFn); err != nil {
				fmt.Fprintf(os.Stderr, "Extraction warning: %v\n", err)
			}
		}()
	}

	// Create and run TUI
	model := tui.New(selected, registry, cmdReg, sessionSvc, petObj, cfg.MaxTurns)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildProviders(cfg *config.Config) []provider.Provider {
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
	return providers
}

func printUsage() {
	fmt.Print(`savant - Terminal-Native AI Coding Assistant

Usage:
  savant              Start interactive TUI
  savant --version    Show version
  savant --help       Show this help

Configuration:
  ~/.savant/config.json    Configuration file
  ~/.savant/savant.db      Session database
  ~/.savant/pet.json       Pet state persistence
  ~/.savant/skills/        Skills directory (Agent Skills standard)

Environment Variables:
  OPENAI_API_KEY           API key for OpenAI-compatible providers
  ANTHROPIC_API_KEY        API key for Anthropic provider
  OLLAMA_HOST              Ollama server URL (default: http://localhost:11434)
  NINEROUTER_URL           9router gateway URL (default: http://localhost:20128)

Commands (in TUI):
  /help           Show all commands
  /provider       Configure AI providers
  /model          Switch model
  /session        Session management
  /config         View/edit configuration
  /pet            Interact with your virtual coding companion

Keybindings:
  Enter           Send message
  Ctrl+C          Cancel / Quit
  Ctrl+B          Toggle sidebar
  Ctrl+L          Toggle log panel
  Ctrl+P          Command palette
  Tab             Cycle sidebar tabs
  Up/Down         Scroll chat
  Left/Right      Move cursor
  Home/End        Jump to start/end of input
`)
}
