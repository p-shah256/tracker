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

	"github.com/p-shah256/tracker/internal/cleaner"
	"github.com/p-shah256/tracker/pkg/types"
)

var clean = cleaner.NewCleaner()

func ParseJD(htmlFilePath string) (*types.JdJson, error) {
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
	cleanResponse := clean.CleanLlmResponse(content)

	var response struct {
		DBFriendly types.JdJson `json:"db_friendly"`
	}

	if err := json.Unmarshal([]byte(cleanResponse), &response); err != nil {
		slog.Error("Failed to parse LLM response as JSON",
			"error", err,
			"invalidJSON", cleanResponse[:min(500, len(cleanResponse))])
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	slog.Info("Response received and cleaned", "parsedData", response.DBFriendly)
	return &response.DBFriendly, nil
}

func GetTailored(dbFriendly *types.JdJson, resume types.Resume) (string, error) {
	tailorPath := filepath.Join(".", "configs", "prompts", "3_tailor.txt")
	tailorPrompt, err := os.ReadFile(tailorPath)
	if err != nil {
		return "", fmt.Errorf("failed to read tailor prompt: %w", err)
	}
	dbFriendlyJSON, err := json.Marshal(dbFriendly)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job data: %w", err)
	}
	resumeJSON, err := json.Marshal(resume)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resume data: %w", err)
	}

	userMessage := fmt.Sprintf(string(tailorPrompt), string(dbFriendlyJSON), string(resumeJSON))
	slog.Debug("GET TAILORED, sending this to Gemini:", "userMessage", userMessage)

	content, err := callGeminiAPI(string(tailorPrompt), userMessage)
	if err != nil {
		return "", fmt.Errorf("resume tailoring failed: %w", err)
	}

	cleanResponse := clean.CleanLlmResponse(content)

	return cleanResponse, nil
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

	model := client.GenerativeModel("gemini-2.0-flash")
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
