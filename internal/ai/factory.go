package ai

import (
	"fmt"
	"strings"
)

// NewProvider creates an AI provider by name, using the given API key.
// providerName must be one of: "openai", "gemini", "none", "log", or "".
func NewProvider(providerName, apiKey string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("ai: api key required for openai provider")
		}
		return NewOpenAIProvider(apiKey), nil

	case "gemini":
		if apiKey == "" {
			return nil, fmt.Errorf("ai: api key required for gemini provider")
		}
		return NewGeminiProvider(apiKey)

	case "none", "", "log":
		return NewLogProvider(), nil

	default:
		return nil, fmt.Errorf("ai: unsupported provider %q", providerName)
	}
}
