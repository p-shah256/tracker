package jobprocessor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/p-shah256/tracker/internal/llm"
	"github.com/p-shah256/tracker/internal/rendercv"
	"github.com/p-shah256/tracker/pkg/types"
)

// ProcessHTML handles the full job processing pipeline:
// 1. Parse job description
// 2. Tailor resume
// 3. Render PDF
// Returns the path to the generated PDF
func ProcessHTML(jobFilePath string) (string, error) {
	// ================= get JOB DESC JSON =================
	jdJson, err := llm.ParseJD(jobFilePath)
	if err != nil {
		return "", fmt.Errorf("job parsing failed: %w", err)
	}
	slog.Info("Parsed job description successfully",
		"company", jdJson.Company,
		"position", jdJson.Position.Name,
		"skills_count", len(jdJson.Skills))

	cvPath := filepath.Join(".", "configs", "Master_CV.yaml")
	resumeData, err := os.ReadFile(cvPath)
	if err != nil {
		return "", fmt.Errorf("failed to read resume: %w", err)
	}

	// ================= get tailored resume =================
	var fullResumeMap map[string]any
	if err = yaml.Unmarshal(resumeData, &fullResumeMap); err != nil {
		return "", fmt.Errorf("failed to parse full resume YAML: %w", err)
	}
	var partialResume types.Resume
	if err = yaml.Unmarshal(resumeData, &partialResume); err != nil {
		return "", fmt.Errorf("failed to parse partial resume YAML: %w", err)
	}

	tailoredPartial, err := llm.GetTailored(jdJson, partialResume)
	if err != nil {
		return "", fmt.Errorf("resume tailoring failed: %w", err)
	}

	fullResumeMap["cv"].(map[string]any)["sections"].(map[string]any)["professional_experience"] =
		tailoredPartial.CV.Sections.ProfessionalExperience
	fullResumeMap["cv"].(map[string]any)["sections"].(map[string]any)["projects"] =
		tailoredPartial.CV.Sections.Projects
	fullResumeMap["cv"].(map[string]any)["sections"].(map[string]any)["technical_skills"] =
		tailoredPartial.CV.Sections.TechnicalSkills

	updatedYAML, err := yaml.Marshal(fullResumeMap)
	if err != nil {
		return "", fmt.Errorf("failed to convert updated resume to YAML: %w", err)
	}

	pdfPath, err := rendercv.RenderTailoredResume(
		updatedYAML,
		jdJson.Company,
		jdJson.Position.Name,
	)
	if err != nil {
		return "", fmt.Errorf("failed to render tailored resume: %w", err)
	}

	return pdfPath, nil
}
