package provider

import (
	"context"
	"strings"

	"github.com/spenc/savant-cli/internal/types"
)

// Router selects the best provider based on availability and configuration.
type Router struct {
	providers    []Provider
	smartRouting types.SmartRoutingConfig
}

// NewRouter creates a router from configured providers.
func NewRouter(providers []Provider, smartRouting types.SmartRoutingConfig) *Router {
	return &Router{
		providers:    providers,
		smartRouting: smartRouting,
	}
}

// Select picks the best available provider.
// If 9router is available, use it. Otherwise fall back to direct providers.
func (r *Router) Select(ctx context.Context) Provider {
	for _, p := range r.providers {
		if p.IsAvailable(ctx) {
			return p
		}
	}
	// Return the first provider even if unavailable (will error on use)
	if len(r.providers) > 0 {
		return r.providers[0]
	}
	return nil
}

// SmartRouting determines whether to use a cheap or strong model.
func (r *Router) SmartRouting(userText string, turnNumber int) types.RoutingDecision {
	if !r.smartRouting.Enabled {
		return types.RoutingDecision{
			Model:      r.smartRouting.StrongModel,
			Complexity: "strong",
			Reason:     "smart routing disabled",
		}
	}

	// First turn always uses strong model
	if turnNumber <= 1 {
		return types.RoutingDecision{
			Model:      r.smartRouting.StrongModel,
			Complexity: "strong",
			Reason:     "first turn",
		}
	}

	trimmed := strings.TrimSpace(userText)
	charCount := len(trimmed)
	wordCount := len(strings.Fields(trimmed))

	maxChars := r.smartRouting.SimpleMaxChars
	if maxChars == 0 {
		maxChars = 160
	}
	maxWords := r.smartRouting.SimpleMaxWords
	if maxWords == 0 {
		maxWords = 28
	}

	// Check for strong keywords
	strongKeywords := []string{
		"plan", "design", "architect", "refactor", "debug",
		"investigate", "analyze", "implement", "optimize",
		"review", "audit", "diagnose", "root cause", "propose",
		"trace", "reproduce",
	}
	lower := strings.ToLower(trimmed)
	for _, kw := range strongKeywords {
		if strings.Contains(lower, kw) {
			return types.RoutingDecision{
				Model:      r.smartRouting.StrongModel,
				Complexity: "strong",
				Reason:     "keyword: " + kw,
			}
		}
	}

	// Check for code fences
	if strings.Contains(trimmed, "```") {
		return types.RoutingDecision{
			Model:      r.smartRouting.StrongModel,
			Complexity: "strong",
			Reason:     "code fence detected",
		}
	}

	// Simple message
	if charCount <= maxChars && wordCount <= maxWords {
		return types.RoutingDecision{
			Model:      r.smartRouting.SimpleModel,
			Complexity: "simple",
			Reason:     "short message",
		}
	}

	return types.RoutingDecision{
		Model:      r.smartRouting.StrongModel,
		Complexity: "strong",
		Reason:     "long or complex message",
	}
}
