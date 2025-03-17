package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	
	"github.com/shah256/tracker/internal/cleaner"
)

var clean = cleaner.NewCleaner()

func ParseJobDesc(htmlFilePath string) (map[string]any, error) {
	parsingRulesPath := filepath.Join(".", "configs", "prompts", "1_parsing.txt")
	parsingRules, err := os.ReadFile(parsingRulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read parsing rules: %w", err)
	}

	htmlContent, err := os.ReadFile(htmlFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML file: %w", err)
	}
	relevantContent := clean.CleanHTML(string(htmlContent))
	slog.Info("Reduced HTML size",
		"originalSize", len(htmlContent),
		"reducedSize", len(relevantContent))

	prompt := fmt.Sprintf("%s\n\nParse the following job description HTML, maintaining maximum detail while ensuring clean, normalized data:\n\n%s",
		string(parsingRules), relevantContent)
	slog.Debug("Sending prompt to Gemini",
		"promptLength", len(prompt),
		"promptPreview", prompt)

	content, err := callGeminiAPI("You are a helpful assistant.", prompt)
	if err != nil {
		return nil, fmt.Errorf("job parsing failed: %w", err)
	}
	cleanResponse := clean.CleanLLMResponse(content)

	var parsedData map[string]any
	if err := json.Unmarshal([]byte(cleanResponse), &parsedData); err != nil {
		slog.Error("Failed to parse LLM response as JSON",
			"error", err,
			"invalidJSON", cleanResponse[:min(500, len(cleanResponse))])
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}
	slog.Debug("Response received and cleaned", "parsedData", parsedData)
	return parsedData, nil
}

func GetTailored(dbFriendly map[string]any, relevantYAML map[string]any) (string, error) {
	tailorPath := filepath.Join(".", "configs", "prompts", "3_tailor.txt")
	tailorPrompt, err := os.ReadFile(tailorPath)
	if err != nil {
		return "", fmt.Errorf("failed to read tailor prompt: %w", err)
	}

	dbFriendlyJSON, err := json.Marshal(dbFriendly)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job data: %w", err)
	}

	slog.Debug("Data sizes for tailoring",
		"jobDataSize", len(dbFriendlyJSON),
		"resumeDataSize", len(relevantYAML))

	relevantYAMLJSON, err := json.Marshal(relevantYAML)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resume data: %w", err)
	}

	userMessage := fmt.Sprintf(
		"I need to tailor a resume for a job. Here's the job data and resume segments:\n\n"+
			"JOB DATA: %s\n\n"+
			"RESUME SECTIONS: %s\n\n"+
			"Please tailor the resume based on your tailoring instructions.",
		string(dbFriendlyJSON), string(relevantYAMLJSON))

	slog.Debug("Sending tailoring request to Gemini", "messageLength", len(userMessage))

	return callGeminiAPI(string(tailorPrompt), userMessage)
}

func callGeminiAPI(systemPrompt, userPrompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	if systemPrompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(systemPrompt)},
		}
	}
	prompt := []genai.Part{genai.Text(userPrompt)}
	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return "", fmt.Errorf("Gemini API call failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini API")
	}
	response, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return "", fmt.Errorf("unexpected response format from Gemini API")
	}

	return string(response), nil
}
