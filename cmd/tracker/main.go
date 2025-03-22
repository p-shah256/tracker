package main

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/p-shah256/tracker/internal/api"
	"github.com/p-shah256/tracker/pkg/logger"
)

func main() {
	logger.Setup()

	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file", "error", err)
	}

	slog.Info("Starting Resume Tailor web application...")

	geminiKey := os.Getenv("GEMINI_KEY")
	if geminiKey == "" {
		slog.Error("Gemini API key not found in environment variables")
		os.Exit(1)
	}

	port := 8080
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil {
			port = p
		}
	}

	server, err := api.NewServer(port)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	slog.Info("Server initialized", "port", port)
	if err := server.Start(); err != nil {
		slog.Error("Error starting API server", "error", err)
		os.Exit(1)
	}
}
