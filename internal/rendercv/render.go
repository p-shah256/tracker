package rendercv

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

// 1. Write the tailored YAML to disk
// 2. Call rendercv command to generate PDF
// 3. Return the path to the generated PDF
func RenderTailoredResume(tailoredResumeYAML []byte, jobCompany, jobPosition string) (string, error) {
	// Get the current working directory for absolute paths
	workDir, err := os.Getwd()
	if err != nil {
		slog.Error("Failed to get working directory", "error", err)
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	slog.Info("Starting resume rendering process",
		"company", jobCompany,
		"position", jobPosition,
		"workingDir", workDir)

	userName := os.Getenv("USER_NAME")
	cvFileName := userName + "_CV" + ".yaml"
	absYamlPath := filepath.Join(workDir, cvFileName)
	err = os.WriteFile(absYamlPath, tailoredResumeYAML, 0644)
	if err != nil {
		slog.Error("Failed to write tailored YAML", "error", err, "absolutePath", absYamlPath)
		return "", fmt.Errorf("failed to write tailored YAML to %s: %w", absYamlPath, err)
	}

	cmd := exec.Command("rendercv", "render", absYamlPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("rendercv command failed",
			"error", err,
			"output", string(output),
			"workingDir", workDir)
		return "", fmt.Errorf("rendercv command failed: %w, output: %s", err, string(output))
	}

	pdfName := filepath.Base(cvFileName)
	pdfName = pdfName[:len(pdfName)-len(filepath.Ext(pdfName))] + ".pdf"
	renderOutputDir := filepath.Join(workDir, "rendercv_output")
	absPdfPath := filepath.Join(renderOutputDir, pdfName)
	if _, err := os.Stat(absPdfPath); os.IsNotExist(err) {
		slog.Error("Expected PDF not found", "absolutePath", absPdfPath)
		return "", fmt.Errorf("expected PDF not found at %s", absPdfPath)
	}

	return absPdfPath, nil
}
