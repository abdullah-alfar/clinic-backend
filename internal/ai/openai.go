package ai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIProvider calls the OpenAI Chat Completions API (gpt-4o).
type OpenAIProvider struct {
	client *openai.Client
}


func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIProvider{client: &client}
}

func (p *OpenAIProvider) Generate(ctx context.Context, input string) (string, error) {
	completion, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(input),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices returned")
	}

	return completion.Choices[0].Message.Content, nil
}
