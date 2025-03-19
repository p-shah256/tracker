package transformation

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

// TransformHighScoringEntries transforms high-scoring resume entries to emphasize job requirements
// Uses goroutines to prevent blocking
func TransformHighScoringEntries(scoredResume *types.ScoredResume, extractedSkills *types.ExtractedSkills, minScore int) (*types.TransformedResume, error) {
	// Create a channel to receive the result
	resultCh := make(chan struct {
		transformedResume *types.TransformedResume
		err               error
	})

	// Run the CPU-bound operation in a separate goroutine
	go func() {
		transformedResume, err := transformHighScoringEntriesSync(scoredResume, extractedSkills, minScore)
		resultCh <- struct {
			transformedResume *types.TransformedResume
			err               error
		}{transformedResume, err}
	}()

	// Wait for the result
	result := <-resultCh
	return result.transformedResume, result.err
}

// transformHighScoringEntriesSync is the synchronous implementation that will run in a goroutine
func transformHighScoringEntriesSync(scoredResume *types.ScoredResume, extractedSkills *types.ExtractedSkills, minScore int) (*types.TransformedResume, error) {
	// Filter high-scoring entries
	var highScoringExperiences []types.ScoredExperienceItem
	for _, exp := range scoredResume.ProfessionalExperience {
		if exp.Score >= minScore {
			highScoringExperiences = append(highScoringExperiences, exp)
		}
	}

	var highScoringProjects []types.ScoredProjectItem
	for _, proj := range scoredResume.Projects {
		if proj.Score >= minScore {
			highScoringProjects = append(highScoringProjects, proj)
		}
	}

	// If no high-scoring entries, return error
	if len(highScoringExperiences) == 0 && len(highScoringProjects) == 0 {
		return nil, fmt.Errorf("no high-scoring entries found with score >= %d", minScore)
	}

	// Convert inputs to JSON for the prompt
	filteredResumeData := struct {
		ProfessionalExperience []types.ScoredExperienceItem `json:"professional_experience"`
		Projects               []types.ScoredProjectItem    `json:"projects"`
	}{
		ProfessionalExperience: highScoringExperiences,
		Projects:               highScoringProjects,
	}

	filteredResumeJSON, err := json.Marshal(filteredResumeData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filtered resume data: %w", err)
	}

	skillsJSON, err := json.Marshal(extractedSkills)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal skills data: %w", err)
	}

	// Create the transformation prompt
	prompt := fmt.Sprintf(`Rewrite these high-scoring resume bullet points to emphasize these exact keywords from the job description.
Every bullet must:
1. Include a metric or quantifiable achievement
2. Front-load with technical achievement
3. Mirror the job's exact terminology
4. Make it clear the candidate has done exactly what the employer wants
5. Be concise and impactful

Job Requirements:
%s

High-Scoring Resume Entries:
%s

Return the result as a JSON object with the following structure:
{
  "professional_experience": [
    {
      "company": "company name",
      "position": "position title",
      "highlights": [
        {
          "original": "original bullet point",
          "transformed": "rewritten bullet point",
          "emphasized_skills": ["skill1", "skill2"]
        }
      ]
    }
  ],
  "projects": [
    {
      "name": "project name",
      "highlights": [
        {
          "original": "original bullet point",
          "transformed": "rewritten bullet point",
          "emphasized_skills": ["skill1", "skill3"]
        }
      ]
    }
  ]
}`, string(skillsJSON), string(filteredResumeJSON))

	// Call the LLM API
	content, err := callGeminiAPI("You are a resume optimization assistant. Rewrite resume bullet points to emphasize job-specific skills and include metrics.", prompt)
	if err != nil {
		return nil, fmt.Errorf("resume transformation failed: %w", err)
	}

	// Clean and parse the response
	cleanResponse := clean.CleanLlmResponse(content)
	
	var transformedResume types.TransformedResume
	if err := json.Unmarshal([]byte(cleanResponse), &transformedResume); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return &transformedResume, nil
}

// GenerateAlternative generates an alternative version of a specific bullet point
func GenerateAlternative(bulletPoint string, matchingSkills []string) (string, error) {
	// Create a channel to receive the result
	resultCh := make(chan struct {
		alternative string
		err         error
	})

	// Run the CPU-bound operation in a separate goroutine
	go func() {
		alternative, err := generateAlternativeSync(bulletPoint, matchingSkills)
		resultCh <- struct {
			alternative string
			err         error
		}{alternative, err}
	}()

	// Wait for the result
	result := <-resultCh
	return result.alternative, result.err
}

// generateAlternativeSync is the synchronous implementation that will run in a goroutine
func generateAlternativeSync(bulletPoint string, matchingSkills []string) (string, error) {
	skillsJSON, err := json.Marshal(matchingSkills)
	if err != nil {
		return "", fmt.Errorf("failed to marshal skills data: %w", err)
	}

	prompt := fmt.Sprintf(`Generate an alternative version of this resume bullet point that emphasizes these skills: %s

Original bullet point:
%s

The alternative version must:
1. Include a specific metric or quantifiable achievement
2. Front-load with technical achievement
3. Use the exact terminology from the skills list
4. Be concise and impactful
5. Demonstrate the same experience but with different wording

Return only the rewritten bullet point with no additional text.`, string(skillsJSON), bulletPoint)

	// Call the LLM API
	content, err := callGeminiAPI("You are a resume optimization assistant. Generate alternative bullet points that emphasize specific skills.", prompt)
	if err != nil {
		return "", fmt.Errorf("alternative generation failed: %w", err)
	}

	// Clean the response
	cleanResponse := clean.CleanLlmResponse(content)
	
	return cleanResponse, nil
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
