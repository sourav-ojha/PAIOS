package main

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicSynth is the paid fallback — opt-in via --provider anthropic,
// never the default. See docs/04-open-source-evaluation.md for cost tradeoffs.
type anthropicSynth struct {
	apiKey string
}

func (a *anthropicSynth) synthesize(ctx context.Context, contextBlock, question string) (string, error) {
	client := anthropic.NewClient(option.WithAPIKey(a.apiKey))

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_5,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(synthPrompt(contextBlock, question))),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic call: %w", err)
	}

	var out string
	for _, block := range msg.Content {
		if block.Type == "text" {
			out += block.Text
		}
	}
	return out, nil
}
