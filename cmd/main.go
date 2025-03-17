package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"dbot/internal"
)

var bot *discordgo.Session

func main() {
	initLogger()
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
		handleError(s, m, fmt.Errorf("failed to create temp directory: %w", err))
		return
	}

	filePath, err := downloadFile(url, "temp")
	if err != nil {
		handleError(s, m, err)
		return
	}
	slog.Info("Downloaded file", "path", filePath)
	defer os.Remove(filePath)

	jobData, err := internal.ParseJobDesc(filePath)
	if err != nil {
		handleError(s, m, fmt.Errorf("job parsing failed: %w", err))
		return
	}
	slog.Info("Parsed job description successfully")

	resumePath := "./Pranchal_Shah_CV.yaml"
	resumeData, err := os.ReadFile(resumePath)
	if err != nil {
		handleError(s, m, fmt.Errorf("failed to read resume: %w", err))
		return
	}

	// For now, we'll use a simple map structure for the resume
	// In a real implementation, you'd parse the YAML properly
	resume := map[string]interface{}{
		"sections": resumeData,
	}
	slog.Info("Loaded resume", "path", resumePath)

	tailoredData, err := internal.GetTailored(jobData, resume)
	if err != nil {
		handleError(s, m, fmt.Errorf("resume tailoring failed: %w", err))
		return
	}

	s.MessageReactionRemove(m.ChannelID, m.ID, "⏳", s.State.User.ID)
	s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
	s.ChannelMessageSend(m.ChannelID, tailoredData)
}
