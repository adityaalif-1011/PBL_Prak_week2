package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// NewLogger membuat logger ke stdout dan file
func NewLogger() *slog.Logger {
	// Buat folder logs
	os.MkdirAll("logs", 0755)

	// Buat file log
	logFile, err := os.OpenFile(
		filepath.Join("logs", "app.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		panic("failed to create log file: " + err.Error())
	}

	// Tulis ke stdout dan file
	writer := io.MultiWriter(os.Stdout, logFile)

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: parseLevel(GetEnv("LOG_LEVEL", "info")),
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
