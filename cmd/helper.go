package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
	"log"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

func initLogger() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func loadResume(path string) (Resume, error) {
	log.Printf("Loading resume from: %s", path)
	var resume Resume
	file, err := os.Open(path)
	if err != nil {
		return resume, fmt.Errorf("resume file not found at %s", path)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err = decoder.Decode(&resume); err != nil {
		return resume, errors.New("failed to parse resume file")
	}

	if len(resume.CV.Sections.TechnicalSkills) == 0 ||
		len(resume.CV.Sections.ProfessionalExperience) == 0 ||
		len(resume.CV.Sections.Projects) == 0 {
		return resume, errors.New("resume sections not found")
	}

	log.Println("Resume loaded and verified successfully")
	return resume, nil
}

func downloadFile(url, filename string) (string, error) {
	log.Printf("Downloading file from: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	filePath := filepath.Join(os.TempDir(), filename)
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	log.Println("File downloaded successfully")
	return filePath, nil
}

func handleError(s *discordgo.Session, m *discordgo.MessageCreate, err error) {
	slog.Error("Processing error", "error", err)
	s.MessageReactionRemove(m.ChannelID, m.ID, "⏳", s.State.User.ID)
	s.MessageReactionAdd(m.ChannelID, m.ID, "❌")
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %v", err))
}
