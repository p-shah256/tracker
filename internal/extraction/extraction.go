package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/p-shah256/tracker/internal/cleaner"
	"github.com/p-shah256/tracker/pkg/types"
)

var clean = cleaner.NewCleaner()

// ExtractSkills extracts skills from a job description
// It uses asyncio.to_thread equivalent in Go (goroutines) to prevent blocking
func ExtractSkills(jobDescContent string) (*types.ExtractedSkills, error) {
	// Create a channel to receive the result
	resultCh := make(chan struct {
		skills *types.ExtractedSkills
		err    error
	})

	// Run the CPU-bound operation in a separate goroutine
	go func() {
		skills, err := extractSkillsSync(jobDescContent)
		resultCh <- struct {
			skills *types.ExtractedSkills
			err    error
		}{skills, err}
	}()

	// Wait for the result
	result := <-resultCh
	return result.skills, result.err
}

// extractSkillsSync is the synchronous implementation that will run in a goroutine
func extractSkillsSync(jobDescContent string) (*types.ExtractedSkills, error) {
	// Clean HTML content if needed
	relevantContent := clean.CleanHTML(jobDescContent)

	// Create the extraction prompt
	prompt := `Extract every technical skill, tool, platform, methodology, and metric mentioned in this job description. 
Format as a prioritized list with required skills first, nice-to-have second.
Return the result as a JSON object with the following structure:
{
  "required_skills": [
    {"name": "skill name", "context": "original text from job description"}
  ],
  "nice_to_have_skills": [
    {"name": "skill name", "context": "original text from job description"}
  ],
  "company_info": {
    "name": "company name if mentioned",
    "position": "job title",
    "level": "seniority level if mentioned"
  }
}

Job Description:
` + relevantContent

	// Call the LLM API
	content, err := callGeminiAPI("You are a precise skill extraction assistant. Extract only skills explicitly mentioned in the job description.", prompt)
	if err != nil {
		return nil, fmt.Errorf("skill extraction failed: %w", err)
	}

	// Clean and parse the response
	cleanResponse := clean.CleanLlmResponse(content)
	
	var extractedSkills types.ExtractedSkills
	if err := json.Unmarshal([]byte(cleanResponse), &extractedSkills); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return &extractedSkills, nil
}

// callGeminiAPI calls the Gemini API with the given prompts
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
