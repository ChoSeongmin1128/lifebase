package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var absPathFn = filepath.Abs

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	Google       GoogleConfig
	PublicData   PublicDataConfig
	JWT          JWTConfig
	Storage      StorageConfig
	StateHMACKey string
}

type ServerConfig struct {
	Port        int
	Env         string
	Domain      string
	WebOrigin   string
	AdminOrigin string
	APIOrigin   string
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

type PublicDataConfig struct {
	HolidayServiceKey string
	HolidayEndpoint   string
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
	_, thisFile, _, _ := runtime.Caller(0)
	// config.go 기준 프로젝트 루트는 5단계 상위다.
	rootDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	loadEnvFiles(rootDir)

	port, _ := strconv.Atoi(getEnv("SERVER_PORT", "38117"))
	accessExpiry, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "1h"))
	refreshExpiry, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "720h"))

	return &Config{
		Server: ServerConfig{
			Port:        port,
			Env:         getEnv("SERVER_ENV", "development"),
			Domain:      getEnv("DOMAIN", "lifebase.cc"),
			WebOrigin:   getEnv("WEB_URL", "http://localhost:39001"),
			AdminOrigin: getEnv("ADMIN_URL", "http://localhost:39001"),
			APIOrigin:   getEnv("API_URL", "http://localhost:38117"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://seongmin@localhost:5432/lifebase_dev?sslmode=disable"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		Google: GoogleConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		},
		PublicData: PublicDataConfig{
			HolidayServiceKey: getEnv("KASI_HOLIDAY_SERVICE_KEY", ""),
			HolidayEndpoint:   getEnv("KASI_HOLIDAY_ENDPOINT", "https://apis.data.go.kr/B090041/openapi/service/SpcdeInfoService/getRestDeInfo"),
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
		StateHMACKey: getEnv("STATE_HMAC_KEY", "dev-hmac-key"),
	}, nil
}

func loadEnvFiles(rootDir string) {
	env := detectServerEnv(rootDir)

	// 우선순위:
	// process env > .env.<env>.local > .env.local > .env.<env> > .env
	// (godotenv.Load는 이미 설정된 값을 덮어쓰지 않으므로 우선순위 높은 파일을 먼저 로드)
	loadFromDirs([]string{
		".env." + env + ".local",
		".env.local",
		".env." + env,
		".env",
	}, ".", rootDir)
}

func loadFromDirs(files []string, dirs ...string) {
	seen := map[string]struct{}{}

	for _, dir := range dirs {
		abs, err := absPathFn(dir)
		if err != nil {
			abs = dir
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}

		for _, file := range files {
			_ = godotenv.Load(filepath.Join(abs, file))
		}
	}
}

func detectServerEnv(rootDir string) string {
	if v := os.Getenv("SERVER_ENV"); v != "" {
		return v
	}

	// SERVER_ENV가 파일에만 있을 수 있으므로 최소 파일 집합에서 먼저 탐지한다.
	candidates := []string{
		filepath.Join(".", ".env.local"),
		filepath.Join(".", ".env"),
		filepath.Join(rootDir, ".env.local"),
		filepath.Join(rootDir, ".env"),
	}

	seen := map[string]struct{}{}
	for _, file := range candidates {
		abs, err := absPathFn(file)
		if err != nil {
			abs = file
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}

		values, err := godotenv.Read(abs)
		if err != nil {
			continue
		}
		if v := values["SERVER_ENV"]; v != "" {
			return v
		}
	}

	return "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
