package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func setRequiredLoadEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("STATE_HMAC_KEY", "state-hmac")
	t.Setenv("JWT_ACCESS_EXPIRY", "1h")
	t.Setenv("JWT_REFRESH_EXPIRY", "48h")
}

func TestGetEnv(t *testing.T) {
	t.Setenv("CONFIG_TEST_ENV", "value")
	if got := getEnv("CONFIG_TEST_ENV", "fallback"); got != "value" {
		t.Fatalf("expected value, got %q", got)
	}
	if got := getEnv("CONFIG_TEST_ENV_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestDetectServerEnvFromProcess(t *testing.T) {
	t.Setenv("SERVER_ENV", "production")
	if got := detectServerEnv(t.TempDir()); got != "production" {
		t.Fatalf("expected production, got %q", got)
	}
}

func TestDetectServerEnvFromFiles(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	cwd := t.TempDir()
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SERVER_ENV=root\n"), 0o600); err != nil {
		t.Fatalf("write root env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".env.local"), []byte("SERVER_ENV=local\n"), 0o600); err != nil {
		t.Fatalf("write cwd env: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	_ = os.Unsetenv("SERVER_ENV")

	if got := detectServerEnv(root); got != "local" {
		t.Fatalf("expected local, got %q", got)
	}
}

func TestDetectServerEnvFallsBackToRootAndDefault(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	cwd := t.TempDir()
	root := t.TempDir()
	_ = os.Unsetenv("SERVER_ENV")

	if err := os.WriteFile(filepath.Join(root, ".env.local"), []byte("SERVER_ENV=root-local\n"), 0o600); err != nil {
		t.Fatalf("write root env.local: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if got := detectServerEnv(root); got != "root-local" {
		t.Fatalf("expected root-local, got %q", got)
	}

	if err := os.Remove(filepath.Join(root, ".env.local")); err != nil {
		t.Fatalf("remove root env.local: %v", err)
	}
	if got := detectServerEnv(root); got != "development" {
		t.Fatalf("expected development fallback, got %q", got)
	}
}

func TestDetectServerEnvFromCwdEnvAndRootEnv(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	cwd := t.TempDir()
	root := t.TempDir()
	_ = os.Unsetenv("SERVER_ENV")

	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".env"), []byte("SERVER_ENV=cwd-env\n"), 0o600); err != nil {
		t.Fatalf("write cwd env: %v", err)
	}
	if got := detectServerEnv(root); got != "cwd-env" {
		t.Fatalf("expected cwd-env, got %q", got)
	}

	if err := os.Remove(filepath.Join(cwd, ".env")); err != nil {
		t.Fatalf("remove cwd env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SERVER_ENV=root-env\n"), 0o600); err != nil {
		t.Fatalf("write root env: %v", err)
	}
	if got := detectServerEnv(root); got != "root-env" {
		t.Fatalf("expected root-env, got %q", got)
	}
}

func TestLoadFromDirsPrefersFirstFileAndDedupDir(t *testing.T) {
	dir := t.TempDir()
	key := "CONFIG_TEST_PRIORITY"
	_ = os.Unsetenv(key)

	if err := os.WriteFile(filepath.Join(dir, ".env.first"), []byte(key+"=first\n"), 0o600); err != nil {
		t.Fatalf("write first env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.second"), []byte(key+"=second\n"), 0o600); err != nil {
		t.Fatalf("write second env: %v", err)
	}

	loadFromDirs([]string{".env.first", ".env.second"}, dir, dir)
	if got := os.Getenv(key); got != "first" {
		t.Fatalf("expected first, got %q", got)
	}
}

func TestLoadFromDirsFallsBackWhenAbsFails(t *testing.T) {
	fileDir := t.TempDir()
	key := "CONFIG_TEST_ABS_FALLBACK"
	_ = os.Unsetenv(key)
	if err := os.WriteFile(filepath.Join(fileDir, ".env.only"), []byte(key+"=set\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	prevAbs := absPathFn
	t.Cleanup(func() { absPathFn = prevAbs })
	absPathFn = func(path string) (string, error) {
		if path == fileDir {
			return "", errors.New("abs failed")
		}
		return prevAbs(path)
	}

	loadFromDirs([]string{".env.only"}, fileDir)
	if got := os.Getenv(key); got != "set" {
		t.Fatalf("expected env to load despite abs fallback, got %q", got)
	}
}

func TestDetectServerEnvAbsFallbackAndDuplicateCandidates(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SERVER_ENV=dup\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	_ = os.Unsetenv("SERVER_ENV")

	prevAbs := absPathFn
	t.Cleanup(func() { absPathFn = prevAbs })
	absPathFn = func(path string) (string, error) {
		return "", errors.New("abs failed")
	}

	if got := detectServerEnv("."); got != "dup" {
		t.Fatalf("expected dup via fallback path, got %q", got)
	}
}

func TestDetectServerEnvSkipsDuplicateCandidates(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	_ = os.Unsetenv("SERVER_ENV")

	if got := detectServerEnv("."); got != "development" {
		t.Fatalf("expected development fallback with duplicate candidates, got %q", got)
	}
}

func TestServerConfigWebURL(t *testing.T) {
	cfg := ServerConfig{WebOrigin: "https://web.example.com"}
	if got := cfg.WebURL(); got != "https://web.example.com" {
		t.Fatalf("expected web origin, got %q", got)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	setRequiredLoadEnv(t)
	t.Setenv("SERVER_PORT", "4000")
	t.Setenv("SERVER_ENV", "staging")
	t.Setenv("DOMAIN", "example.com")
	t.Setenv("WEB_URL", "https://web.example.com")
	t.Setenv("ADMIN_URL", "https://admin.example.com")
	t.Setenv("API_URL", "https://api.example.com")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("REDIS_URL", "redis://example")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("KASI_HOLIDAY_SERVICE_KEY", "hkey")
	t.Setenv("KASI_HOLIDAY_ENDPOINT", "https://holiday.example.com")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("JWT_ACCESS_EXPIRY", "1h")
	t.Setenv("JWT_REFRESH_EXPIRY", "48h")
	t.Setenv("STORAGE_DATA_PATH", "/data")
	t.Setenv("STORAGE_THUMB_PATH", "/thumbs")
	t.Setenv("STATE_HMAC_KEY", "hmac")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.Port != 4000 {
		t.Fatalf("expected port 4000, got %d", cfg.Server.Port)
	}
	if cfg.Server.Env != "staging" {
		t.Fatalf("expected staging env, got %q", cfg.Server.Env)
	}
	if cfg.JWT.AccessExpiry.Hours() != 1 {
		t.Fatalf("expected 1h access expiry, got %s", cfg.JWT.AccessExpiry)
	}
	if cfg.JWT.RefreshExpiry.Hours() != 48 {
		t.Fatalf("expected 48h refresh expiry, got %s", cfg.JWT.RefreshExpiry)
	}
	if cfg.StateHMACKey != "hmac" {
		t.Fatalf("expected hmac key, got %q", cfg.StateHMACKey)
	}
}

func TestLoadInvalidNumericAndDurationReturnsError(t *testing.T) {
	setRequiredLoadEnv(t)
	t.Setenv("SERVER_PORT", "not-a-number")
	t.Setenv("JWT_ACCESS_EXPIRY", "invalid")
	t.Setenv("JWT_REFRESH_EXPIRY", "invalid")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestLoadDatabaseDefaultUsesDevelopmentDatabase(t *testing.T) {
	setRequiredLoadEnv(t)
	t.Setenv("DATABASE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Database.URL != "postgres://seongmin@localhost:5432/lifebase_dev?sslmode=disable" {
		t.Fatalf("expected development database fallback, got %q", cfg.Database.URL)
	}
}

func TestLoadFailsWithoutRequiredSecrets(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("STATE_HMAC_KEY", "")
	t.Setenv("JWT_ACCESS_EXPIRY", "1h")
	t.Setenv("JWT_REFRESH_EXPIRY", "48h")

	if _, err := Load(); err == nil {
		t.Fatal("expected missing secrets error")
	}
}
