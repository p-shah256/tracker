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
	slog.Info("Starting resume rendering process",
		"company", jobCompany,
		"position", jobPosition)

	userName := os.Getenv("USER_NAME")
	cvFileName := userName + "_CV" + ".yaml"
	slog.Info("Created CV filename", "filename", cvFileName)

	err := os.WriteFile(cvFileName, tailoredResumeYAML, 0644)
	if err != nil {
		slog.Error("Failed to write tailored YAML", "error", err)
		return "", fmt.Errorf("failed to write tailored YAML: %w", err)
	}
	slog.Info("Successfully wrote YAML file to disk", "path", cvFileName)

	cmd := exec.Command("rendercv", "render", cvFileName)
	slog.Info("Executing rendercv command", "command", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("rendercv command failed",
			"error", err,
			"output", string(output))
		return "", fmt.Errorf("rendercv command failed: %w, output: %s", err, string(output))
	}
	slog.Info("rendercv command executed successfully", "output_length", len(output))

	pdfName := filepath.Base(cvFileName)
	pdfName = pdfName[:len(pdfName)-len(filepath.Ext(pdfName))] + ".pdf"
	pdfPath := filepath.Join("rendercv_output", pdfName)
	slog.Info("Generated PDF path", "path", pdfPath)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		slog.Error("Expected PDF not found", "path", pdfPath)
		return "", fmt.Errorf("expected PDF not found at %s", pdfPath)
	}
	slog.Info("Successfully verified PDF exists", "path", pdfPath)

	return pdfPath, nil
}
