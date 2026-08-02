package main

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// geminiSynth uses Google AI Studio's free tier — Gemini Flash, 1M token
// context, generous free quota. See docs/04-open-source-evaluation.md for
// the cost/quality landscape that led here after local Ollama models
// (1B/7B/12B) all failed grounded extraction on the same test question.
type geminiSynth struct {
	apiKey string
	model  string
}

func (g *geminiSynth) synthesize(ctx context.Context, contextBlock, question string) (string, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("gemini client: %w", err)
	}

	result, err := client.Models.GenerateContent(
		ctx,
		g.model,
		genai.Text(synthPrompt(contextBlock, question)),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("gemini call: %w", err)
	}
	return result.Text(), nil
}
