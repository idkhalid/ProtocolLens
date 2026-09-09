package config

import (
	"log/slog"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr                   string
	DatabasePath           string
	DataDir                string
	LogLevel               slog.Level
	MaxUploadBytes         int64
	ReplayEnabled          bool
	ReplayAllowedPorts     []uint16
	ReplayTimeout          time.Duration
	ReplayMaxRequestBytes  int64
	ReplayMaxResponseBytes int64
	ReplayMaxConcurrent    int
	LocalCaptureEnabled    bool
}

func Load() Config {
	return Config{
		Addr:                   env("PROTOCOLLENS_ADDR", ":8080"),
		DatabasePath:           env("PROTOCOLLENS_DATABASE_PATH", "data/protocollens.db"),
		DataDir:                env("PROTOCOLLENS_DATA_DIR", "data"),
		LogLevel:               logLevel(env("PROTOCOLLENS_LOG_LEVEL", "info")),
		MaxUploadBytes:         envInt64("PROTOCOLLENS_MAX_UPLOAD_SIZE", 50<<20),
		ReplayEnabled:          envBool("PROTOCOLLENS_REPLAY_ENABLED", false),
		ReplayAllowedPorts:     envPorts("PROTOCOLLENS_REPLAY_ALLOWED_PORTS", []uint16{80, 443}),
		ReplayTimeout:          envDuration("PROTOCOLLENS_REPLAY_TIMEOUT", 10*time.Second, 60*time.Second),
		ReplayMaxRequestBytes:  envInt64("PROTOCOLLENS_REPLAY_MAX_REQUEST_SIZE", 1<<20),
		ReplayMaxResponseBytes: envInt64("PROTOCOLLENS_REPLAY_MAX_RESPONSE_SIZE", 2<<20),
		ReplayMaxConcurrent:    int(envInt64("PROTOCOLLENS_REPLAY_MAX_CONCURRENT", 4)),
		LocalCaptureEnabled:    envBool("PROTOCOLLENS_LOCAL_CAPTURE_ENABLED", false),
	}
}

func LocalCaptureBindAllowed(addr string) bool {
	host := strings.TrimSpace(addr)
	if host == "" {
		return false
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else if strings.HasPrefix(host, ":") {
		return false
	}
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if host == "localhost" {
		return true
	}
	addrIP, err := netip.ParseAddr(host)
	return err == nil && addrIP.IsLoopback()
}

func env(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1" || value == "yes"
}

func envInt64(name string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(name), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(name string, fallback, max time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func envPorts(name string, fallback []uint16) []uint16 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	var ports []uint16
	for _, part := range strings.Split(value, ",") {
		parsed, err := strconv.ParseUint(strings.TrimSpace(part), 10, 16)
		if err != nil || parsed == 0 {
			return fallback
		}
		ports = append(ports, uint16(parsed))
	}
	if len(ports) == 0 {
		return fallback
	}
	return ports
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
