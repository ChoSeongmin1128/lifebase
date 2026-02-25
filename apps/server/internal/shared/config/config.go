package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Google   GoogleConfig
	JWT      JWTConfig
	Storage  StorageConfig
}

type ServerConfig struct {
	Port   int
	Env    string
	WebOrigin string
}

func (s ServerConfig) WebURL() string {
	return s.WebOrigin
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type StorageConfig struct {
	DataPath  string
	ThumbPath string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, _ := strconv.Atoi(getEnv("SERVER_PORT", "38117"))
	accessExpiry, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	refreshExpiry, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "720h"))

	return &Config{
		Server: ServerConfig{
			Port:      port,
			Env:       getEnv("SERVER_ENV", "development"),
			WebOrigin: getEnv("WEB_URL", "http://localhost:39001"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://seongmin@localhost:5432/lifebase?sslmode=disable"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		Google: GoogleConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", ""),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
		Storage: StorageConfig{
			DataPath:  getEnv("STORAGE_DATA_PATH", "/Volumes/WDRedPlus/LifeBase/data"),
			ThumbPath: getEnv("STORAGE_THUMB_PATH", "/Users/seongmin/lifebase-cache/thumbs"),
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
