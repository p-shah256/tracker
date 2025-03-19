package helper

import (
	"log/slog"
	"os"
)

func InitLogger() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

//
// func DownloadFile(url, filename string) (string, error) {
//
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to download file: %v", err)
// 	}
// 	defer resp.Body.Close()
//
// 	if resp.StatusCode != http.StatusOK {
// 		return "", fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
// 	}
//
// 	filePath := filepath.Join(os.TempDir(), filename)
// 	file, err := os.Create(filePath)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create file: %v", err)
// 	}
// 	defer file.Close()
//
// 	_, err = io.Copy(file, resp.Body)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to save file: %v", err)
// 	}
//
// 	log.Println("File downloaded successfully")
// 	return filePath, nil
// }
//
// func HandleError(s *discordgo.Session, m *discordgo.MessageCreate, err error) {
// 	slog.Error("Processing error", "error", err)
// 	s.MessageReactionRemove(m.ChannelID, m.ID, "⏳", s.State.User.ID)
// 	s.MessageReactionAdd(m.ChannelID, m.ID, "❌")
// 	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %v", err))
// }
