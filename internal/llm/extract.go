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
	logger.Info("starting skill extraction")

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
		"duration_ms", time.Since(startTime).Milliseconds())

	cleanResponse := clean.CleanLlmResponse(content)
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
		"required_skills", len(extractedSkills.RequiredSkills),
		"nice_to_have_skills", len(extractedSkills.NiceToHaveSkills))

	skillsJSON, err := json.Marshal(extractedSkills)
	if err != nil {
		logger.Error("failed to marshal skills data", "error", err)
		return nil, fmt.Errorf("failed to marshal skills data: %w", err)
	}

	// TODO: maybe later you can remove reasoning for each hightlight and just have a single score reasoning for the entire section
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
		  "score_reasoning": "WHY this scores poorly - be specific about what's missing or weak. Be brutal and honest around 2-3 sentences",
		  "highlights": [
			{
			  "text": "original bullet point",
			  "score": 7,
			  "matching_skills": ["skill1"],
			  "reasoning": "WHY this scores poorly - be specific about what's missing or weak",
			  "missing_skills": ["skill2"]
			}
		  ]
		}
	  ],
	  "projects": [{
		"name": "project name",
		"score": 8,
		"matching_skills": ["skill1", "skill2"],
		"score_reasoning": "WHY this scores poorly - be specific about what's missing or weak. Be brutal and honest around 2-3 sentences",
		"highlights": [
			{
			  "text": "original bullet point",
			  "score": 7,
			  "matching_skills": ["skill1"],
			  "reasoning": "WHY this scores poorly - be specific about what's missing or weak",
			  "missing_skills": ["skill2"]
			}
		  ]
		}],
	  "overall_score": 7.5,
	  "overall_comments": "overall comments on the resume, existing skills, missing skills, etc. (in 2-3 sentences)",	
	  "missing_skills": [{
		  "name": "skill1",
		  "context": "exact text where mentioned",
		  "importance": 1-10
		}],
	  "existing_skills": [{
		"name": "skill1",
		"context": "exact text where mentioned",
		"importance": 1-10
	  }],
	}`, string(skillsJSON), resumeText)

	logger.Debug("prompt", "prompt", prompt)

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
		"duration_ms", time.Since(startTime).Milliseconds())

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

func (l *LLM) TransformResumeBullets(scored *types.ScoredResume, items []types.TransformItem) ([]types.TransformItem, error) {
	for i := range items {
		if items[i].ID == "" {
			items[i].ID = fmt.Sprintf("%d", i+1)
		}
	}

	// only send missing skills and existing skills, and overall comments instead of sending all the extracted skills
	scoredJSON, err := json.Marshal(types.ScoredResume{
		MissingSkills:   scored.MissingSkills,
		ExistingSkills:  scored.ExistingSkills,
		OverallComments: scored.OverallComments,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal skills data: %w", err)
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal items data: %w", err)
	}

	prompt := fmt.Sprintf(`Transform these resume bullets to exactly match the job requirements, regardless of original content:
		1. Replace original skills with required job skills from the missing_skills list
		2. Keep metrics (numbers, percentages) but apply them to new context
		3. Use direct, simple language with job-specific terms
		4. Stay within ±25%% of original character count
		5. Start with strong action verbs

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
		}`, string(scoredJSON), string(itemsJSON))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// lets print the prompt nicely like a json object with indentations
	slog.Debug("prompt", "prompt", prompt)
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
