package ai

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GeminiProvider calls the Google Gemini API.
type GeminiProvider struct {
	client    *genai.Client
	modelName string
}

func NewGeminiProvider(apiKey string) (*GeminiProvider, error) {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to create client: %w", err)
	}
	return &GeminiProvider{client: client, modelName: "gemini-2.0-flash"}, nil
}

func (p *GeminiProvider) Generate(ctx context.Context, input string) (string, error) {
	result, err := p.client.Models.GenerateContent(ctx, p.modelName, genai.Text(input), nil)
	if err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}

	if result == nil || len(result.Candidates) == 0 {
		return "", fmt.Errorf("gemini: no candidates returned")
	}

	candidate := result.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response content")
	}

	return fmt.Sprintf("%v", candidate.Content.Parts[0]), nil
}
