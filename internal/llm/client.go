package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/p-shah256/tracker/internal/cleaner"
)

var clean = cleaner.NewCleaner()

type LLM struct {
	client *genai.Client
	model  string
}

func New(apiKey string) (*LLM, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &LLM{
		client: client,
		model:  "gemini-2.0-flash",
	}, nil
}

func (l *LLM) Close() {
	if l.client != nil {
		l.client.Close()
	}
}

func (l *LLM) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	model := l.client.GenerativeModel(l.model)

	if systemPrompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(systemPrompt)},
		}
	}

	prompt := []genai.Part{genai.Text(userPrompt)}

	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	slog.Info("LLM API call",
		"input_tokens", resp.UsageMetadata.PromptTokenCount,
		"output_tokens", resp.UsageMetadata.CandidatesTokenCount,
		"total_tokens", resp.UsageMetadata.TotalTokenCount)


	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}

	response, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return "", fmt.Errorf("unexpected response format from LLM")
	}

	return string(response), nil
}
