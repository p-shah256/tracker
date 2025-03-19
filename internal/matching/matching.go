package matching

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/p-shah256/tracker/internal/cleaner"
	"github.com/p-shah256/tracker/pkg/types"
	"google.golang.org/api/option"
)

var clean = cleaner.NewCleaner()

func ScoreResume(extractedSkills *types.ExtractedSkills, resumeText string) (*types.ScoredResume, error) {
	resultCh := make(chan struct {
		scoredResume *types.ScoredResume
		err          error
	})

	go func() {
		scoredResume, err := scoreResumeSync(extractedSkills, resumeText)
		resultCh <- struct {
			scoredResume *types.ScoredResume
			err          error
		}{scoredResume, err}
	}()

	result := <-resultCh
	return result.scoredResume, result.err
}

func scoreResumeSync(extractedSkills *types.ExtractedSkills, resumeText string) (*types.ScoredResume, error) {
	skillsJSON, err := json.Marshal(extractedSkills)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal skills data: %w", err)
	}

	prompt := fmt.Sprintf(`For each experience entry in my resume, identify which skills/requirements from the job description it addresses. 
Score each entry 1-10 on relevance, where 10 means it perfectly matches what the employer is looking for.

Job Requirements:
%s

My Resume:
%s

Return the result as a JSON object with the following structure:
{
  "professional_experience": [
    {
      "company": "company name",
      "position": "position title",
      "score": 8,
      "matching_skills": ["skill1", "skill2"],
      "highlights": [
        {
          "text": "original bullet point",
          "score": 7,
          "matching_skills": ["skill1"]
        }
      ]
    }
  ],
  "projects": [
    {
      "name": "project name",
      "score": 9,
      "matching_skills": ["skill1", "skill3"],
      "highlights": [
        {
          "text": "original bullet point",
          "score": 8,
          "matching_skills": ["skill1", "skill3"]
        }
      ]
    }
  ]
}`, string(skillsJSON), resumeText)

	content, err := callGeminiAPI("You are a resume evaluation assistant. Score how well each resume entry matches the job requirements.", prompt)
	if err != nil {
		return nil, fmt.Errorf("resume scoring failed: %w", err)
	}

	cleanResponse := clean.CleanLlmResponse(content)

	var scoredResume types.ScoredResume
	if err := json.Unmarshal([]byte(cleanResponse), &scoredResume); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return &scoredResume, nil
}

func callGeminiAPI(systemPrompt, userPrompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_KEY environment variable not set")
	}

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
