package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	Addr           string
	DatabasePath   string
	DataDir        string
	LogLevel       slog.Level
	MaxUploadBytes int64
}

func Load() Config {
	return Config{
		Addr:           env("PROTOCOLLENS_ADDR", ":8080"),
		DatabasePath:   env("PROTOCOLLENS_DATABASE_PATH", "data/protocollens.db"),
		DataDir:        env("PROTOCOLLENS_DATA_DIR", "data"),
		LogLevel:       logLevel(env("PROTOCOLLENS_LOG_LEVEL", "info")),
		MaxUploadBytes: envInt64("PROTOCOLLENS_MAX_UPLOAD_SIZE", 50<<20),
	}
}

func env(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func envInt64(name string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(name), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func logLevel(value string) slog.Level {
	switch value {
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
