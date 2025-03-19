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

	prompt := `Analyze this job description as if you were a professional resume writer identifying EVERY POSSIBLE keyword that could be used to match a candidate to this role. Extract:
		1. Technical skills (explicit AND implied from job duties)
		2. Software/tools mentioned
		3. Methodologies/processes described
		4. Soft skills required
		5. Industry-specific terminology
		6. Domain knowledge areas
		7. Responsibilities that imply specific skills
		Be THOROUGH and AGGRESSIVE in your extraction - don't just look for explicit "skill" words, but identify ALL competencies someone would need to do this job well.
		Format as a prioritized JSON with:
		{
		  "required_skills": [
			{"name": "skill", "context": "exact text where this skill was mentioned or implied", "importance": 1-10}
		  ],
		  "nice_to_have_skills": [
			{"name": "skill", "context": "exact text where this skill was mentioned or implied", "importance": 1-10}
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

	prompt := fmt.Sprintf(`Transform these resume bullet points to better match the job requirements while sounding like a real human wrote them.
		For each bullet point:
		1. KEEP THE SAME CHARACTER COUNT (±10 percent) - this is critical
		2. Preserve the original achievement metrics but make them relevant to the job
		3. Use language that demonstrates competence in the SPECIFIC skills needed for this job
		4. Sound like a professional in this field, not an HR robot
		5. Maintain the original voice and style
		6. Score how much better the new version matches the job (1-10)
		Job Requirements:
		%s

		Bullet Points to Transform:
		%s

		Return as a JSON array with:
		{
			"id": "original id",
			"original_text": "the original text",
			"transformed_text": "the rewritten text", 
			"char_count_original": 120,
			"char_count_new": 115,
			"original_skills": ["skills already in the bullet"],
			"added_skills": ["new skills emphasized"],
			"original_score": 5,
			"new_score": 8,
			"section": "original section",
			"company": "original company",
			"position": "original position",
			"name": "original name if available"
		}

		CRUCIAL: The bullet must sound natural and authentic - avoid corporate speak, buzzword salad, or awkward keyword stuffing. It should read like it was written by an actual professional in this field, not an AI.`, string(skillsJSON), string(itemsJSON))

	slog.Info("sending this prompt", "prompt",prompt)
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
