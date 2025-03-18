package bot

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"

	"github.com/p-shah256/tracker/internal/helper"
	"github.com/p-shah256/tracker/internal/jobprocessor"
)

type Bot struct {
	session *discordgo.Session
}

func New(token string) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}
	bot := &Bot{
		session: session,
	}
	session.AddHandler(bot.onMessageCreate)
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	return bot, nil
}

func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("error opening Discord session: %w", err)
	}
	slog.Info("Bot is running...")
	return nil
}

func (b *Bot) Close() error {
	return b.session.Close()
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	slog.Info("Received message", "content", m.Content, "author", m.Author.Username)
	for _, att := range m.Attachments {
		if filepath.Ext(att.Filename) == ".html" {
			go b.processJobPosting(s, m, att.URL)
			break
		}
	}
}

func (b *Bot) processJobPosting(s *discordgo.Session, m *discordgo.MessageCreate, url string) {
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

	pdfPath, err := jobprocessor.ProcessHTML(filePath)
	if err != nil {
		helper.HandleError(s, m, err)
		return
	}

	pdfFile, err := os.Open(pdfPath)
	if err != nil {
		helper.HandleError(s, m, fmt.Errorf("failed to open PDF file: %w", err))
		return
	}
	defer pdfFile.Close()

	slog.Info("Received a Pdf path successfully")
	_, err = s.ChannelFileSend(m.ChannelID, filepath.Base(pdfPath), pdfFile)
	if err != nil {
		helper.HandleError(s, m, fmt.Errorf("failed to send PDF file: %w", err))
		return
	}

	s.MessageReactionsRemoveAll(m.ChannelID, m.ID)
	s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
	slog.Info("Done processing!")
}
