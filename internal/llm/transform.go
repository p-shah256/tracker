package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/p-shah256/tracker/pkg/types"
)

func (l *LLM) TransformResumeBullets(scored *types.Section) (types.TransformResponse, error) {
	// only send missing skills and existing skills, and overall comments instead of sending all the extracted skills
	// section has all the items required
	sectionStr, err := json.Marshal(*scored)
	if err != nil {
		return types.TransformResponse{}, fmt.Errorf("failed to marshal section data: %w", err)
	}

	prompt := fmt.Sprintf(`Transform these resume bullets to exactly match the job requirements, regardless of original content:
		1. Replace original skills with required job skills from the missing_skills list
		2. Keep metrics (numbers, percentages) but apply them to new context
		3. Use direct, simple language with job-specific terms
		4. Stay within ±25%% of original character count
		5. Start with strong action verbs

		Section to transform:
		%s

		Return as JSON array:
		{
		"name": "name of section",
		"items": [{
			"original_bullet": "original text",
			"transformed_bullet": "rewritten text", 
			"char_count_original": 120,
			"char_count_new": 115,
			"original_skills": ["skills already in bullet"],
			"added_skills": ["new skills emphasized"],
			"original_score": 5,
			"new_score": 8,
			}, ...]
		"improvement_explanation": "how this rewrite addresses the weaknesses of this section (2-3 sentences)"
		}`, string(sectionStr))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// lets print the prompt nicely like a json object with indentations
	slog.Debug("prompt", "prompt", prompt)
	content, err := l.Generate(ctx, "You are a resume optimization expert who helps tailor resumes to specific job descriptions.", prompt)
	if err != nil {
		return types.TransformResponse{}, fmt.Errorf("resume transformation failed: %w", err)
	}

	cleanResponse := clean.CleanLlmResponse(content)

	var transformedItems types.TransformResponse
	if err := json.Unmarshal([]byte(cleanResponse), &transformedItems); err != nil {
		return types.TransformResponse{}, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return transformedItems, nil
}
