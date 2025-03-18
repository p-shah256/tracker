package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/p-shah256/tracker/internal/bot"
	"github.com/p-shah256/tracker/internal/helper"
)

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

	discordBot, err := bot.New(botToken)
	if err != nil {
		slog.Error("Error creating Discord bot", "error", err)
		os.Exit(1)
	}

	if err = discordBot.Start(); err != nil {
		slog.Error("Error starting Discord bot", "error", err)
		os.Exit(1)
	}

	defer discordBot.Close()

	select {}
}
