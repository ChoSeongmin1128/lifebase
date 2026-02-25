package config

import (
	"os"
	"path/filepath"
	"runtime"
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
	Port      int
	Env       string
	Domain    string
	WebOrigin string
	APIOrigin string
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
	// 프로젝트 루트 .env 로드 (CWD 기준 또는 소스 파일 기준)
	_ = godotenv.Load()
	_, thisFile, _, _ := runtime.Caller(0)
	rootEnv := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", ".env")
	_ = godotenv.Load(rootEnv)

	port, _ := strconv.Atoi(getEnv("SERVER_PORT", "38117"))
	accessExpiry, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	refreshExpiry, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "720h"))

	return &Config{
		Server: ServerConfig{
			Port:      port,
			Env:       getEnv("SERVER_ENV", "development"),
			Domain:    getEnv("DOMAIN", "lifebase.cc"),
			WebOrigin: getEnv("WEB_URL", "http://localhost:39001"),
			APIOrigin: getEnv("API_URL", "http://localhost:38117"),
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
