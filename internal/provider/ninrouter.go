package provider

import (
	"context"
	"net/http"
	"time"
)

// NineRouterProvider wraps an OpenAI-compatible provider with 9router health checking.
type NineRouterProvider struct {
	*OpenAIProvider
}

// NewNineRouterProvider creates a provider backed by 9router gateway.
func NewNineRouterProvider(baseURL string) *NineRouterProvider {
	return &NineRouterProvider{
		OpenAIProvider: NewOpenAIProvider("9router", baseURL, "", ""),
	}
}

// IsAvailable checks if 9router is running.
func (p *NineRouterProvider) IsAvailable(ctx context.Context) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return false
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}
