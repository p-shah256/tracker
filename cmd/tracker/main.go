package main

import (
	"github.com/joho/godotenv"
	"github.com/p-shah256/tracker/internal/api"
	"github.com/p-shah256/tracker/internal/helper"
	"log/slog"
	"os"
)

func main() {
	helper.InitLogger()
	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file", "error", err)
	}
	slog.Info("Starting Resume Tailor web application...")
	geminiKey := os.Getenv("GEMINI_KEY")
	if geminiKey == "" {
		slog.Error("Gemini API key not found in environment variables")
		os.Exit(1)
	}
	server, _ := api.NewServer(8080)
	if err := server.Start(); err != nil {
		slog.Error("Error starting API server", "error", err)
		os.Exit(1)
	}
}
