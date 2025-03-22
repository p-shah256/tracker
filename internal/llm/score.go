package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/p-shah256/tracker/pkg/types"
)

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

	Return valid JSON without any formatting or tab characters, ensuring all string values are properly escaped:
	{
	  "overall_score": 7.5,
	  "overall_comments": "overall comments on the resume, existing skills, missing skills, etc. (in 3-4 sentences)",
	  "sections": [
		{
	  	"name": "if experience = 'company-position', else 'project name' (ignore others for now)",
		  "score": 8,
		  "score_reasoning": "WHY this scores poorly - be specific about what's missing or weak. Be brutal and honest. Be detailed enough to use this reasoning to optimize the resume. Be detailed enough so that it can be used to optimize the resume.",
		  "original_content": "original content of the item",
		  "missing_skills": [{ "name": "skill1", "importance": 1-10 (same as job requirements) }],
		}
	  ],
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

	logger.Debug("parsed LLM response", "scored_resume", scoredResume)

	return &scoredResume, nil
}
