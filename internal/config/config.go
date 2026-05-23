// Package config handles Savant CLI configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spenc/savant-cli/internal/types"
)

const (
	DefaultConfigDir = ".savant"
	DefaultPort      = 20128
)

// Config is the top-level Savant configuration.
type Config struct {
	// Providers lists configured AI providers, in priority order.
	Providers []types.ProviderConfig `json:"providers"`

	// DefaultProvider is the name of the provider to use when none is specified.
	DefaultProvider string `json:"default_provider"`

	// DefaultModel is the model to use when none is specified.
	DefaultModel string `json:"default_model"`

	// SmartRouting controls cheap-for-simple routing.
	SmartRouting types.SmartRoutingConfig `json:"smart_routing"`

	// AgentRouting maps agent names to provider overrides.
	AgentRouting map[string]string `json:"agent_routing,omitempty"`

	// AgentModels maps model aliases to provider overrides.
	AgentModels map[string]types.ProviderOverride `json:"agent_models,omitempty"`

	// Theme selects the UI theme.
	Theme string `json:"theme"`

	// Permissions configures tool approval behavior.
	Permissions PermissionsConfig `json:"permissions"`

	// MaxTurns caps the number of agent loop iterations.
	MaxTurns int `json:"max_turns"`

	// AutoCompact enables automatic context compaction.
	AutoCompact bool `json:"auto_compact"`

	// AutoCompactThreshold is the context usage percentage to trigger compaction.
	AutoCompactThreshold float64 `json:"auto_compact_threshold"`
}

// PermissionsConfig controls tool approval behavior.
type PermissionsConfig struct {
	// AutoApproveReads skips approval for read-only tools.
	AutoApproveReads bool `json:"auto_approve_reads"`

	// AutoApproveWrites skips approval for write tools (dangerous).
	AutoApproveWrites bool `json:"auto_approve_writes"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Providers: []types.ProviderConfig{
			{
				Name:    "9router",
				BaseURL: fmt.Sprintf("http://localhost:%d/v1", DefaultPort),
				Model:   "cc/claude-opus-4-5-20251101",
				Enabled: true,
			},
			{
				Name:    "mimo",
				BaseURL: "https://api.xiaomimimo.com/v1",
				Model:   "MiMo-V2-Pro",
				Enabled: true,
			},
			{
				Name:    "opengateway",
				BaseURL: "https://opengateway.gitlawb.com/v1",
				Model:   "xiaomi-mimo-v2-pro",
				Enabled: true,
			},
			{
				Name:    "ollama",
				BaseURL: "http://localhost:11434/v1",
				Model:   "llama3.1:8b",
				Enabled: true,
			},
		},
		DefaultProvider: "9router",
		DefaultModel:    "cc/claude-opus-4-5-20251101",
		SmartRouting: types.SmartRoutingConfig{
			Enabled:        true,
			SimpleModel:    "MiMo-V2-Pro",
			StrongModel:    "cc/claude-opus-4-5-20251101",
			SimpleMaxChars: 160,
			SimpleMaxWords: 28,
		},
		Theme:     "cyberpunk",
		Permissions: PermissionsConfig{
			AutoApproveReads:  true,
			AutoApproveWrites: false,
		},
		MaxTurns:             100,
		AutoCompact:          true,
		AutoCompactThreshold: 0.80,
	}
}

// ConfigDir returns the path to ~/.savant.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: use temp dir
		return filepath.Join(os.TempDir(), DefaultConfigDir)
	}
	return filepath.Join(home, DefaultConfigDir)
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

// Load reads the config from disk, returning defaults if the file doesn't exist.
func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := ConfigPath()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// DataDir returns the path to the Savant data directory (for DB, sessions, etc.)
func DataDir() string {
	return ConfigDir()
}

// DBPath returns the path to the SQLite database.
func DBPath() string {
	return filepath.Join(DataDir(), "savant.db")
}

// OSName returns a human-readable OS name for display.
func OSName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}
