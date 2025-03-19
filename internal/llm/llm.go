package llm

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

func ExtractSkills(jobDescContent string) (*types.ExtractedSkills, error) {
	relevantContent := clean.CleanHTML(jobDescContent)

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

	content, err := callGeminiAPI("You are a precise skill extraction assistant. Extract only skills explicitly mentioned in the job description.", prompt)
	if err != nil {
		return nil, fmt.Errorf("skill extraction failed: %w", err)
	}
	cleanResponse := clean.CleanLlmResponse(content)

	var extractedSkills types.ExtractedSkills
	if err := json.Unmarshal([]byte(cleanResponse), &extractedSkills); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return &extractedSkills, nil
}

func ScoreResume(extractedSkills *types.ExtractedSkills, resumeText string) (*types.ScoredResume, error) {
	skillsJSON, err := json.Marshal(extractedSkills)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal skills data: %w", err)
	}
	prompt := fmt.Sprintf(`For each experience entry in my resume, identify which skills/requirements from the job description it addresses. 
	Score each entry 1-10 on relevance, where 10 means it perfectly matches what the employer is looking for.
	Also calculate an overall match score for the entire resume.
	
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
	  ],
	  "overall_score": 7.5
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

// LLM function to transform resume bullets
func TransformResumeBullets(extractedSkills *types.ExtractedSkills, items []types.TransformItem, emphasisLevel string) ([]types.TransformItem, error) {
	// Add unique IDs to items if they don't have them
	for i := range items {
		if items[i].ID == "" {
			items[i].ID = fmt.Sprintf("%d", i+1)
		}
	}

	skillsJSON, err := json.Marshal(extractedSkills)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal skills data: %w", err)
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal items data: %w", err)
	}

	prompt := fmt.Sprintf(`Rewrite these resume bullet points to better match the job requirements.
	For each bullet point:
	1. Emphasize the skills/requirements from the job description that match
	2. Every bullet must include a quantifiable metric or achievement
	3. Front-load with technical achievements
	4. Mirror the job's terminology exactly
	5. Keep the same basic information but optimize it for this specific job
	6. The emphasis level requested is: %s
	
	Job Requirements:
	%s
	
	Bullet Points to Transform:
	%s
	
	Return the result as a JSON array with the following structure for each item:
	{
		"id": "original id",
		"original_text": "the original text",
		"transformed_text": "the rewritten text",
		"matching_skills": ["skill1", "skill2"],
		"section": "original section",
		"company": "original company if available",
		"position": "original position if available",
		"name": "original name if available"
	}
	
	Don't abbreviate or use acronyms unless they appear in the original text or job description.
	Focus on making each bullet sound natural while incorporating the required skills.`,
		emphasisLevel, string(skillsJSON), string(itemsJSON))

	content, err := callGeminiAPI("You are a resume optimization expert who helps tailor resumes to specific job descriptions.", prompt)
	if err != nil {
		return nil, fmt.Errorf("resume transformation failed: %w", err)
	}

	cleanResponse := clean.CleanLlmResponse(content)

	var transformedItems []types.TransformItem
	if err := json.Unmarshal([]byte(cleanResponse), &transformedItems); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return transformedItems, nil
}

// LLM function to generate alternative bullet
func GenerateAlternativeBullet(extractedSkills *types.ExtractedSkills, originalText string, matchingSkills []string, emphasisLevel string) (string, error) {
	skillsJSON, err := json.Marshal(extractedSkills)
	if err != nil {
		return "", fmt.Errorf("failed to marshal skills data: %w", err)
	}

	matchingSkillsJSON, err := json.Marshal(matchingSkills)
	if err != nil {
		return "", fmt.Errorf("failed to marshal matching skills data: %w", err)
	}

	prompt := fmt.Sprintf(`Generate a different version of this resume bullet point that better matches the job requirements.
	The new version should:
	1. Emphasize the skills/requirements from the job description that match
	2. Include a quantifiable metric or achievement
	3. Front-load with technical achievements
	4. Mirror the job's terminology exactly
	5. Keep the same basic information but optimize it for this specific job
	6. Be distinctly different from the original in structure and phrasing
	7. The emphasis level requested is: %s
	
	Job Requirements:
	%s
	
	Original Bullet Point:
	%s
	
	Matching Skills to Emphasize:
	%s
	
	Return ONLY the alternative bullet text with no additional commentary or explanation.
	Don't abbreviate or use acronyms unless they appear in the original text or job description.
	Focus on making the bullet sound natural while incorporating the required skills.`,
		emphasisLevel, string(skillsJSON), originalText, string(matchingSkillsJSON))

	content, err := callGeminiAPI("You are a resume optimization expert who helps tailor resumes to specific job descriptions.", prompt)
	if err != nil {
		return "", fmt.Errorf("alternative generation failed: %w", err)
	}

	return clean.CleanLlmResponse(content), nil
}
