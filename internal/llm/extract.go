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

func (l *LLM) TransformResumeBullets(extractedSkills *types.ExtractedSkills, items []types.TransformItem, emphasisLevel string) ([]types.TransformItem, error) {
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

func (l *LLM) GenerateAlternativeBullet(extractedSkills *types.ExtractedSkills, originalText string, matchingSkills []string, emphasisLevel string) (string, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	content, err := l.Generate(ctx, "You are a resume optimization expert who helps tailor resumes to specific job descriptions.", prompt)
	if err != nil {
		return "", fmt.Errorf("alternative generation failed: %w", err)
	}

	return clean.CleanLlmResponse(content), nil
}
