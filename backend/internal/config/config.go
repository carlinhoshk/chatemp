package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           string
	DBPath         string
	UploadDir      string
	StaticDir      string
	RoomTTL        time.Duration
	EphemeralTTL   time.Duration
	MediaSweep     time.Duration
	RoomPurge      time.Duration
	MaxUploadBytes int64
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getInt64(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func Load() Config {
	return Config{
		Port:           getenv("PORT", "8080"),
		DBPath:         getenv("DB_PATH", "./data/chatemp.db"),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		StaticDir:      getenv("STATIC_DIR", ""),
		RoomTTL:        getDuration("ROOM_TTL", 24*time.Hour),
		EphemeralTTL:   getDuration("EPHEMERAL_TTL", 15*time.Second),
		MediaSweep:     getDuration("MEDIA_SWEEP", time.Minute),
		RoomPurge:      getDuration("ROOM_PURGE", 30*time.Minute),
		MaxUploadBytes: getInt64("MAX_UPLOAD_BYTES", 50<<20),
	}
}
