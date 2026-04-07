package ai

import (
	"context"
	"log"
)

// LogProvider is the dev/fallback AI implementation.
// It echoes the prompt with a prefix so it's obvious in logs.
type LogProvider struct{}

func NewLogProvider() *LogProvider { return &LogProvider{} }

func (p *LogProvider) Generate(_ context.Context, input string) (string, error) {
	response := "[AI LOG PROVIDER] Received prompt: " + input
	log.Println(response)
	return "This is a simulated AI response. Configure a real AI provider (OpenAI or Gemini) in Settings → AI Configuration.", nil
}
