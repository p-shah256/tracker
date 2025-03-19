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

func GenAlternative(bulletPoint string, matchingSkills []string) (string, error) {
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

	content, err := callGeminiAPI("You are a resume optimization assistant. Generate alternative bullet points that emphasize specific skills.", prompt)
	if err != nil {
		return "", fmt.Errorf("alternative generation failed: %w", err)
	}
	cleanResponse := clean.CleanLlmResponse(content)
	return cleanResponse, nil
}

// func TransformHighScoring(scoredResume *types.ScoredResume, extractedSkills *types.ExtractedSkills, minScore int) (*types.TransformedResume, error) {
// 	var highScoringExperiences []types.ScoredExperienceItem
// 	for _, exp := range scoredResume.ProfessionalExperience {
// 		if exp.Score >= minScore {
// 			highScoringExperiences = append(highScoringExperiences, exp)
// 		}
// 	}
//
// 	var highScoringProjects []types.ScoredProjectItem
// 	for _, proj := range scoredResume.Projects {
// 		if proj.Score >= minScore {
// 			highScoringProjects = append(highScoringProjects, proj)
// 		}
// 	}
//
// 	if len(highScoringExperiences) == 0 && len(highScoringProjects) == 0 {
// 		return nil, fmt.Errorf("no high-scoring entries found with score >= %d", minScore)
// 	}
//
// 	filteredResumeData := struct {
// 		ProfessionalExperience []types.ScoredExperienceItem `json:"professional_experience"`
// 		Projects               []types.ScoredProjectItem    `json:"projects"`
// 	}{
// 		ProfessionalExperience: highScoringExperiences,
// 		Projects:               highScoringProjects,
// 	}
//
// 	filteredResumeJSON, err := json.Marshal(filteredResumeData)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to marshal filtered resume data: %w", err)
// 	}
//
// 	skillsJSON, err := json.Marshal(extractedSkills)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to marshal skills data: %w", err)
// 	}
//
// 	prompt := fmt.Sprintf(`Rewrite these high-scoring resume bullet points to emphasize these exact keywords from the job description.
// 	Every bullet must:
// 	1. Include a metric or quantifiable achievement
// 	2. Front-load with technical achievement
// 	3. Mirror the job's exact terminology
// 	4. Make it clear the candidate has done exactly what the employer wants
// 	5. Be concise and impactful
//
// 	Job Requirements:
// 	%s
//
// 	High-Scoring Resume Entries:
// 	%s
//
// 	Return the result as a JSON object with the following structure:
// 	{
// 	  "professional_experience": [
// 		{
// 		  "company": "company name",
// 		  "position": "position title",
// 		  "highlights": [
// 			{
// 			  "original": "original bullet point",
// 			  "transformed": "rewritten bullet point",
// 			  "emphasized_skills": ["skill1", "skill2"]
// 			}
// 		  ]
// 		}
// 	  ],
// 	  "projects": [
// 		{
// 		  "name": "project name",
// 		  "highlights": [
// 			{
// 			  "original": "original bullet point",
// 			  "transformed": "rewritten bullet point",
// 			  "emphasized_skills": ["skill1", "skill3"]
// 			}
// 		  ]
// 		}
// 	  ]
// 	}`, string(skillsJSON), string(filteredResumeJSON))
//
// 	content, err := callGeminiAPI("You are a resume optimization assistant. Rewrite resume bullet points to emphasize job-specific skills and include metrics.", prompt)
// 	if err != nil {
// 		return nil, fmt.Errorf("resume transformation failed: %w", err)
// 	}
// 	cleanResponse := clean.CleanLlmResponse(content)
// 	var transformedResume types.TransformedResume
// 	if err := json.Unmarshal([]byte(cleanResponse), &transformedResume); err != nil {
// 		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
// 	}
//
// 	return &transformedResume, nil
// }
