package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"

	"github.com/p-shah256/tracker/internal/helper"
	"github.com/p-shah256/tracker/internal/llm"
	"github.com/p-shah256/tracker/internal/rendercv"
	"github.com/p-shah256/tracker/pkg/types"
)

var bot *discordgo.Session

func main() {
	helper.InitLogger()
	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file", "error", err)
	}
	slog.Info("Starting bot...")

	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		slog.Error("Bot token not found in environment variables")
		os.Exit(1)
	}

	var err error
	bot, err = discordgo.New("Bot " + botToken)
	if err != nil {
		slog.Error("Error creating Discord session", "error", err)
		os.Exit(1)
	}

	bot.AddHandler(onMessageCreate)
	bot.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	if err = bot.Open(); err != nil {
		slog.Error("Error opening Discord session", "error", err)
		os.Exit(1)
	}

	slog.Info("Bot is running...")
	defer bot.Close()
	select {} // Keep the program running
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	slog.Info("Received message", "content", m.Content, "author", m.Author.Username)

	for _, att := range m.Attachments {
		if filepath.Ext(att.Filename) == ".html" {
			slog.Info("Processing HTML attachment", "filename", att.Filename)
			go processJobPosting(s, m, att.URL)
			break
		}
	}
}

func processJobPosting(s *discordgo.Session, m *discordgo.MessageCreate, url string) {
	slog.Info("Processing job posting", "url", url)
	s.MessageReactionAdd(m.ChannelID, m.ID, "⏳")

	if err := os.MkdirAll("temp", 0755); err != nil {
		helper.HandleError(s, m, fmt.Errorf("failed to create temp directory: %w", err))
		return
	}

	filePath, err := helper.DownloadFile(url, "temp")
	if err != nil {
		helper.HandleError(s, m, err)
		return
	}
	slog.Info("Downloaded file", "path", filePath)
	defer os.Remove(filePath)

	// ======================== parse job desc ========================
	jd_json, err := llm.ParseJobDesc(filePath)
	if err != nil {
		helper.HandleError(s, m, fmt.Errorf("job parsing failed: %w", err))
		return
	}
	slog.Info("Parsed job description successfully",
		"company", jd_json.Company,
		"position", jd_json.Position.Name,
		"skills_count", len(jd_json.Skills))

	// ======================== parse resume ========================
	cvPath := "./configs/" + "Master_CV.yaml"
	pdfPath, err := PartiallyTailorResume(jd_json, cvPath)
	pdfFile, err := os.Open(pdfPath)
	if err != nil {
		helper.HandleError(s, m, fmt.Errorf("failed to open PDF file: %w", err))
		return
	}
	defer pdfFile.Close()

	_, err = s.ChannelFileSend(m.ChannelID, filepath.Base(pdfPath), pdfFile)
	if err != nil {
		helper.HandleError(s, m, fmt.Errorf("failed to send PDF file: %w", err))
		return
	}
}

// parses resume
// gets tailored
// replaces and marshalls it
// renders tailored resume and returns path
func PartiallyTailorResume(jdJson *types.JdJson, cvPath string) (string, error) {
	resumeData, err := os.ReadFile(cvPath)
	if err != nil {
		return "", fmt.Errorf("failed to read resume: %w", err)
	}
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

	fullResumeMap["cv"].(map[string]any)["sections"].(map[string]any)["technical_skills"] =
		tailoredPartial.CV.Sections.TechnicalSkills
	fullResumeMap["cv"].(map[string]any)["sections"].(map[string]any)["professional_experience"] =
		tailoredPartial.CV.Sections.ProfessionalExperience
	fullResumeMap["cv"].(map[string]any)["sections"].(map[string]any)["projects"] =
		tailoredPartial.CV.Sections.Projects
	fullResumeMap["cv"].(map[string]any)["sections"].(map[string]any)["open_source_contributions"] =
		tailoredPartial.CV.Sections.OpenSourceContributions

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
