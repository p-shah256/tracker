package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/p-shah256/tracker/pkg/types"
)

func (l *LLM) ExtractSkills(jobDescContent string) (*types.ExtractedSkills, error) {
	logger := slog.With(
		"component", "llm",
		"operation", "extract_skills",
	)
	logger.Info("starting skill extraction", "content_length", len(jobDescContent))

	relevantContent := clean.CleanHTML(jobDescContent)
	logger.Debug("cleaned HTML content", "original_length", len(jobDescContent), "cleaned_length", len(relevantContent))

	prompt := `Parse this job description and extract EVERY keyword that could help match a candidate. Be aggressive and thorough:
		1. Technical skills (both stated and implied)
		2. Software/tools 
		3. Methodologies/processes
		4. Domain expertise areas
		5. Industry terminology

		Format as JSON:
		{
		  "required_skills": [
			{"name": "skill", "context": "exact text where mentioned", "importance": 1-10}
		  ],
		  "nice_to_have_skills": [
			{"name": "skill", "context": "exact text where mentioned", "importance": 1-10}
		  ],
		  "company_info": {
			"name": "company name",
			"position": "job title",
			"level": "seniority level"
		  }
		}` + relevantContent

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	logger.Debug("sending prompt to LLM", "prompt_length", len(prompt))
	startTime := time.Now()

	content, err := l.Generate(ctx, "You are a precise skill extraction assistant. Extract only skills explicitly mentioned in the job description.", prompt)
	if err != nil {
		logger.Error("skill extraction failed", "error", err, "duration_ms", time.Since(startTime).Milliseconds())
		return nil, fmt.Errorf("skill extraction failed: %w", err)
	}
	logger.Info("received LLM response",
		"duration_ms", time.Since(startTime).Milliseconds(),
		"response_length", len(content))

	cleanResponse := clean.CleanLlmResponse(content)
	logger.Debug("cleaned LLM response",
		"original_length", len(content),
		"cleaned_length", len(cleanResponse))

	var extractedSkills types.ExtractedSkills
	if err := json.Unmarshal([]byte(cleanResponse), &extractedSkills); err != nil {
		logger.Error("JSON parsing failed", "error", err, "content", cleanResponse)
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	logger.Info("skill extraction completed",
		"required_skills_count", len(extractedSkills.RequiredSkills),
		"nice_to_have_skills_count", len(extractedSkills.NiceToHaveSkills),
		"company_name", extractedSkills.CompanyInfo.Name)

	return &extractedSkills, nil
}

func (l *LLM) ScoreResume(extractedSkills *types.ExtractedSkills, resumeText string) (*types.ScoredResume, error) {
	logger := slog.With(
		"component", "llm",
		"operation", "score_resume",
	)

	logger.Info("starting resume scoring",
		"resume_length", len(resumeText),
		"required_skills", len(extractedSkills.RequiredSkills),
		"nice_to_have_skills", len(extractedSkills.NiceToHaveSkills))

	skillsJSON, err := json.Marshal(extractedSkills)
	if err != nil {
		logger.Error("failed to marshal skills data", "error", err)
		return nil, fmt.Errorf("failed to marshal skills data: %w", err)
	}

	logger.Debug("skills data serialized", "json_size", len(skillsJSON))

	// TODO: add other items here, maybe just take everything like technical skills as well
	prompt := fmt.Sprintf(`Score how each part of this resume matches the job requirements. Be brutally honest about what's missing or weak.
	Job Requirements:
	%s

	Resume:
	%s

	Return as JSON:
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
			  "matching_skills": ["skill1"],
			  "reasoning": "WHY this scores poorly - be specific about what's missing or weak"
			}
		  ]
		}
	  ],
	  "projects": [{
		"name": "project name",
		"score": 8,
		"matching_skills": ["skill1", "skill2"],
		"highlights": [
			{
			  "text": "original bullet point",
			  "score": 7,
			  "matching_skills": ["skill1"],
			  "reasoning": "WHY this scores poorly - be specific about what's missing or weak"
			}
		  ]
		}],
	  "overall_score": 7.5
	}`, string(skillsJSON), resumeText)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	content, err := l.Generate(ctx, "You are a resume evaluation assistant. Score how well each resume entry matches the job requirements.", prompt)
	if err != nil {
		logger.Error("resume scoring failed",
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds())
		return nil, fmt.Errorf("resume scoring failed: %w", err)
	}

	logger.Info("received LLM response",
		"duration_ms", time.Since(startTime).Milliseconds(),
		"response_length", len(content))

	cleanResponse := clean.CleanLlmResponse(content)
	var scoredResume types.ScoredResume
	if err := json.Unmarshal([]byte(cleanResponse), &scoredResume); err != nil {
		logger.Error("JSON parsing failed",
			"error", err,
			"content_preview", cleanResponse[:min(100, len(cleanResponse))])
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return &scoredResume, nil
}

func (l *LLM) TransformResumeBullets(extractedSkills *types.ExtractedSkills, items []types.TransformItem) ([]types.TransformItem, error) {
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

	prompt := fmt.Sprintf(`Rewrite these resume bullets to make them job-targeted weapons. For each:
		1. Address the specific weakness identified in "reasoning"
		2. Stay within ±25%% character count
		3. Keep original metrics but make them relevant
		4. Front-load with technical achievements using job-specific language
		5. Sound like an actual human professional, not HR-speak

		Job Requirements:
		%s

		Bullets to Transform:
		%s

		Return as JSON array:
		{
		  "id": "original id",
		  "original_text": "original text",
		  "transformed_text": "rewritten text", 
		  "char_count_original": 120,
		  "char_count_new": 115,
		  "original_skills": ["skills already in bullet"],
		  "added_skills": ["new skills emphasized"],
		  "original_score": 5,
		  "new_score": 8,
		  "reasoning": "original reasoning for low score",
		  "improvement_explanation": "how this rewrite addresses the weaknesses"
		}`, string(skillsJSON), string(itemsJSON))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	content, err := l.Generate(ctx, "You are a resume optimization expert who helps tailor resumes to specific job descriptions.", prompt)
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
