// Package commands implements the slash command system for Savant CLI.
package commands

import (
	"fmt"
	"strings"
)

// Command represents a slash command.
type Command struct {
	Name        string
	Description string
	Usage       string
	Execute     func(args string) string
}

// Registry holds all registered commands.
type Registry struct {
	commands map[string]*Command
}

// NewRegistry creates a command registry with all built-in commands.
func NewRegistry() *Registry {
	r := &Registry{commands: make(map[string]*Command)}
	r.RegisterHelp()
	r.RegisterProvider()
	r.RegisterModel()
	r.RegisterSession()
	r.RegisterConfig()
	r.RegisterQuit()
	return r
}

// Register adds a command to the registry.
func (r *Registry) Register(cmd *Command) {
	r.commands[cmd.Name] = cmd
}

// Get returns a command by name (without the / prefix).
func (r *Registry) Get(name string) (*Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// All returns all registered commands.
func (r *Registry) All() []*Command {
	cmds := make([]*Command, 0, len(r.commands))
	for _, c := range r.commands {
		cmds = append(cmds, c)
	}
	return cmds
}

// TryExecute attempts to execute a message as a command.
// Returns the result and true if it was a command, or "" and false otherwise.
func (r *Registry) TryExecute(input string) (string, bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return "", false
	}

	parts := strings.SplitN(strings.TrimPrefix(trimmed, "/"), " ", 2)
	name := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	cmd, ok := r.commands[name]
	if !ok {
		return fmt.Sprintf("Unknown command: /%s\nType /help for available commands.", name), true
	}

	return cmd.Execute(args), true
}

// RegisterHelp registers the /help command.
func (r *Registry) RegisterHelp() {
	r.Register(&Command{
		Name:        "help",
		Description: "Show all available commands",
		Usage:       "/help [command]",
		Execute: func(args string) string {
			if args != "" {
				cmd, ok := r.Get(args)
				if !ok {
					return fmt.Sprintf("Unknown command: /%s", args)
				}
				return fmt.Sprintf("/%s - %s\nUsage: %s", cmd.Name, cmd.Description, cmd.Usage)
			}

			var sb strings.Builder
			sb.WriteString("Available Commands:\n")
			sb.WriteString("───────────────────\n")
			for _, cmd := range r.All() {
				sb.WriteString(fmt.Sprintf("  /%-12s %s\n", cmd.Name, cmd.Description))
			}
			sb.WriteString("\nKeybindings:\n")
			sb.WriteString("───────────────────\n")
			sb.WriteString("  Ctrl+S        Toggle sidebar\n")
			sb.WriteString("  Ctrl+L        Toggle log panel\n")
			sb.WriteString("  Ctrl+P        Command palette\n")
			sb.WriteString("  Tab           Cycle sidebar tabs\n")
			sb.WriteString("  Enter         Send message\n")
			sb.WriteString("  Ctrl+C        Cancel / Quit\n")
			return sb.String()
		},
	})
}

// RegisterProvider registers the /provider command.
func (r *Registry) RegisterProvider() {
	r.Register(&Command{
		Name:        "provider",
		Description: "Configure AI providers",
		Usage:       "/provider [list|add|remove|test] [args]",
		Execute: func(args string) string {
			parts := strings.Fields(args)
			if len(parts) == 0 {
				return "Usage: /provider [list|add|remove|test] [args]\n" +
					"  /provider list          Show configured providers\n" +
					"  /provider test [name]   Test provider connectivity\n" +
					"  /provider add <name>    Add a provider interactively"
			}

			switch parts[0] {
			case "list":
				return "Configured Providers:\n" +
					"  1. 9router    - Local gateway (15+ providers)\n" +
					"  2. mimo       - Xiaomi MiMo V2 Pro (free)\n" +
					"  3. opengateway - Gitlawb Opengateway\n" +
					"  4. ollama     - Local Ollama server\n"
			case "test":
				name := "all"
				if len(parts) > 1 {
					name = parts[1]
				}
				return fmt.Sprintf("Testing provider: %s\n(Provider test not yet implemented)", name)
			default:
				return fmt.Sprintf("Unknown provider subcommand: %s", parts[0])
			}
		},
	})
}

// RegisterModel registers the /model command.
func (r *Registry) RegisterModel() {
	r.Register(&Command{
		Name:        "model",
		Description: "Switch or view the current model",
		Usage:       "/model [name]",
		Execute: func(args string) string {
			if args == "" {
				return "Current model: cc/claude-opus-4-5-20251101\n" +
					"Use /model <name> to switch.\n" +
					"Available models depend on your provider."
			}
			return fmt.Sprintf("Switching to model: %s\n(Model switching not yet implemented)", args)
		},
	})
}

// RegisterSession registers the /session command.
func (r *Registry) RegisterSession() {
	r.Register(&Command{
		Name:        "session",
		Description: "Session management",
		Usage:       "/session [new|list|switch|delete]",
		Execute: func(args string) string {
			parts := strings.Fields(args)
			if len(parts) == 0 {
				return "Usage: /session [new|list|switch|delete] [args]\n" +
					"  /session new [title]    Start a new session\n" +
					"  /session list           List recent sessions\n" +
					"  /session switch <id>    Switch to a session\n" +
					"  /session delete <id>    Delete a session"
			}

			switch parts[0] {
			case "new":
				title := "New Session"
				if len(parts) > 1 {
					title = strings.Join(parts[1:], " ")
				}
				return fmt.Sprintf("Created new session: %s\n(Session management not yet implemented)", title)
			case "list":
				return "No sessions yet.\n(Session management not yet implemented)"
			default:
				return fmt.Sprintf("Unknown session subcommand: %s", parts[0])
			}
		},
	})
}

// RegisterPet registers the /pet command for the virtual pet system.
func (r *Registry) RegisterPet(feedFn, playFn, restFn, healFn, statsFn func() string) {
	r.Register(&Command{
		Name:        "pet",
		Description: "Interact with your virtual coding companion",
		Usage:       "/pet [feed|play|rest|heal|stats]",
		Execute: func(args string) string {
			parts := strings.Fields(args)
			if len(parts) == 0 {
				return "Usage: /pet [feed|play|rest|heal|stats]\n" +
					"  /pet feed     Feed your pet\n" +
					"  /pet play     Play with your pet\n" +
					"  /pet rest     Let your pet rest\n" +
					"  /pet heal     Heal your pet\n" +
					"  /pet stats    View pet statistics\n\n" +
					"  Or press Tab to view the Pet sidebar!"
			}
			switch parts[0] {
			case "feed":
				return feedFn()
			case "play":
				return playFn()
			case "rest":
				return restFn()
			case "heal":
				return healFn()
			case "stats":
				return statsFn()
			default:
				return fmt.Sprintf("Unknown pet command: %s\nUse: /pet [feed|play|rest|heal|stats]", parts[0])
			}
		},
	})
}

// RegisterConfig registers the /config command.
func (r *Registry) RegisterConfig() {
	r.Register(&Command{
		Name:        "config",
		Description: "View or edit configuration",
		Usage:       "/config [show|edit|reset]",
		Execute: func(args string) string {
			parts := strings.Fields(args)
			if len(parts) == 0 || parts[0] == "show" {
				return "Configuration:\n" +
					"  Config dir:    ~/.savant/\n" +
					"  Config file:   ~/.savant/config.json\n" +
					"  Database:      ~/.savant/savant.db\n" +
					"  Theme:         cyberpunk\n" +
					"  Max turns:     100\n" +
					"  Auto-compact:  enabled (80% threshold)\n" +
					"  Smart routing: enabled\n"
			}

			switch parts[0] {
			case "edit":
				return "Open ~/.savant/config.json in your editor to modify settings."
			case "reset":
				return "Configuration reset not yet implemented."
			default:
				return fmt.Sprintf("Unknown config subcommand: %s", parts[0])
			}
		},
	})
}
