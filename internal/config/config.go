package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	MaxUploadBytes  int64
	TempDir         string
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		Addr:            getenv("ADDR", "9005"),
		MaxUploadBytes:  getenvInt64("MAX_UPLOAD_BYTES", 32<<20), // 32 MiB
		TempDir:         getenv("TEMP_DIR", os.TempDir()),
		ShutdownTimeout: 10 * time.Second,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}
