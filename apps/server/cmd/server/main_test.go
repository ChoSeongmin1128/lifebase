package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/golang-jwt/jwt/v5"

	authportin "lifebase/internal/auth/port/in"
	holidaydomain "lifebase/internal/holiday/domain"
	holidayportin "lifebase/internal/holiday/port/in"
	"lifebase/internal/shared/config"
)

type authUseCaseStub struct {
	runHourlyFn      func(context.Context) (int, error)
	processOutboxFn  func(context.Context, int) (int, error)
}

func (s *authUseCaseStub) GetAuthURL(string) string { return "" }
func (s *authUseCaseStub) GetAuthURLForApp(string, string) string { return "" }
func (s *authUseCaseStub) HandleCallback(context.Context, string) (*authportin.LoginResult, error) {
	return nil, nil
}
func (s *authUseCaseStub) HandleCallbackForApp(context.Context, string, string) (*authportin.LoginResult, error) {
	return nil, nil
}
func (s *authUseCaseStub) ListGoogleAccounts(context.Context, string) ([]authportin.GoogleAccountSummary, error) {
	return nil, nil
}
func (s *authUseCaseStub) LinkGoogleAccount(context.Context, string, string, string) error { return nil }
func (s *authUseCaseStub) SyncGoogleAccount(context.Context, string, string, authportin.SyncGoogleAccountInput) error {
	return nil
}
func (s *authUseCaseStub) TriggerGoogleSync(context.Context, string, authportin.TriggerGoogleSyncInput) (int, error) {
	return 0, nil
}
func (s *authUseCaseStub) RunHourlyGoogleSync(ctx context.Context) (int, error) {
	if s.runHourlyFn != nil {
		return s.runHourlyFn(ctx)
	}
	return 0, nil
}
func (s *authUseCaseStub) ProcessGooglePushOutbox(ctx context.Context, limit int) (int, error) {
	if s.processOutboxFn != nil {
		return s.processOutboxFn(ctx, limit)
	}
	return 0, nil
}
func (s *authUseCaseStub) RefreshAccessToken(context.Context, string) (*authportin.LoginResult, error) {
	return nil, nil
}
func (s *authUseCaseStub) Logout(context.Context, string) error { return nil }

type holidayUseCaseStub struct {
	refreshFn func(context.Context, holidayportin.RefreshRangeInput) (*holidayportin.RefreshRangeResult, error)
}

func (s *holidayUseCaseStub) ListRange(context.Context, time.Time, time.Time) ([]holidaydomain.Holiday, error) {
	return nil, nil
}
func (s *holidayUseCaseStub) RefreshRange(ctx context.Context, input holidayportin.RefreshRangeInput) (*holidayportin.RefreshRangeResult, error) {
	if s.refreshFn != nil {
		return s.refreshFn(ctx, input)
	}
	return &holidayportin.RefreshRangeResult{}, nil
}

func TestRunGoogleBackgroundPullSync(t *testing.T) {
	oldInterval := googlePullSyncInterval
	oldStartup := googlePullSyncStartupWait
	googlePullSyncInterval = 5 * time.Millisecond
	googlePullSyncStartupWait = 5 * time.Millisecond
	t.Cleanup(func() {
		googlePullSyncInterval = oldInterval
		googlePullSyncStartupWait = oldStartup
	})

	var called int32
	done := make(chan struct{})
	uc := &authUseCaseStub{
		runHourlyFn: func(context.Context) (int, error) {
			if atomic.AddInt32(&called, 1) == 1 {
				close(done)
			}
			return 1, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runGoogleBackgroundPullSync(ctx, uc)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected RunHourlyGoogleSync to be called")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("background pull sync did not stop on context cancel")
	}
}

func TestRunGooglePushOutboxWorker(t *testing.T) {
	oldInterval := googlePushOutboxInterval
	googlePushOutboxInterval = 5 * time.Millisecond
	t.Cleanup(func() {
		googlePushOutboxInterval = oldInterval
	})

	var called int32
	done := make(chan struct{})
	uc := &authUseCaseStub{
		processOutboxFn: func(context.Context, int) (int, error) {
			if atomic.AddInt32(&called, 1) == 1 {
				close(done)
			}
			return 1, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runGooglePushOutboxWorker(ctx, uc)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected ProcessGooglePushOutbox to be called")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("push outbox worker did not stop on context cancel")
	}
}

func TestRunHolidayBackgroundRefresh(t *testing.T) {
	oldInterval := holidayRefreshInterval
	oldStartup := holidayRefreshStartupWait
	holidayRefreshInterval = 5 * time.Millisecond
	holidayRefreshStartupWait = 5 * time.Millisecond
	t.Cleanup(func() {
		holidayRefreshInterval = oldInterval
		holidayRefreshStartupWait = oldStartup
	})

	var called int32
	done := make(chan struct{})
	uc := &holidayUseCaseStub{
		refreshFn: func(context.Context, holidayportin.RefreshRangeInput) (*holidayportin.RefreshRangeResult, error) {
			if atomic.AddInt32(&called, 1) == 1 {
				close(done)
			}
			return &holidayportin.RefreshRangeResult{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runHolidayBackgroundRefresh(ctx, uc)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected RefreshRange to be called")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("holiday refresh worker did not stop on context cancel")
	}
}

func TestRunHolidayBackgroundRefresh_StartupDelayBranch(t *testing.T) {
	oldInterval := holidayRefreshInterval
	oldStartup := holidayRefreshStartupWait
	holidayRefreshInterval = time.Hour
	holidayRefreshStartupWait = 5 * time.Millisecond
	t.Cleanup(func() {
		holidayRefreshInterval = oldInterval
		holidayRefreshStartupWait = oldStartup
	})

	var called int32
	done := make(chan struct{})
	uc := &holidayUseCaseStub{
		refreshFn: func(context.Context, holidayportin.RefreshRangeInput) (*holidayportin.RefreshRangeResult, error) {
			if atomic.AddInt32(&called, 1) == 1 {
				close(done)
			}
			return &holidayportin.RefreshRangeResult{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runHolidayBackgroundRefresh(ctx, uc)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected startup delay branch to trigger RefreshRange")
	}

	cancel()
	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("holiday refresh worker did not stop on context cancel")
	}
}

func TestRunGoogleBackgroundPullSync_Error(t *testing.T) {
	oldInterval := googlePullSyncInterval
	oldStartup := googlePullSyncStartupWait
	googlePullSyncInterval = 5 * time.Millisecond
	googlePullSyncStartupWait = 5 * time.Millisecond
	t.Cleanup(func() {
		googlePullSyncInterval = oldInterval
		googlePullSyncStartupWait = oldStartup
	})

	var called int32
	done := make(chan struct{})
	uc := &authUseCaseStub{
		runHourlyFn: func(context.Context) (int, error) {
			if atomic.AddInt32(&called, 1) == 1 {
				close(done)
			}
			return 0, fmt.Errorf("boom")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runGoogleBackgroundPullSync(ctx, uc)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected RunHourlyGoogleSync to be called")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("background pull sync did not stop on context cancel")
	}
}

func TestRunGooglePushOutboxWorker_Error(t *testing.T) {
	oldInterval := googlePushOutboxInterval
	googlePushOutboxInterval = 5 * time.Millisecond
	t.Cleanup(func() {
		googlePushOutboxInterval = oldInterval
	})

	var called int32
	done := make(chan struct{})
	uc := &authUseCaseStub{
		processOutboxFn: func(context.Context, int) (int, error) {
			if atomic.AddInt32(&called, 1) == 1 {
				close(done)
			}
			return 0, fmt.Errorf("boom")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runGooglePushOutboxWorker(ctx, uc)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected ProcessGooglePushOutbox to be called")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("push outbox worker did not stop on context cancel")
	}
}

func TestRunHolidayBackgroundRefresh_Error(t *testing.T) {
	oldInterval := holidayRefreshInterval
	oldStartup := holidayRefreshStartupWait
	holidayRefreshInterval = 5 * time.Millisecond
	holidayRefreshStartupWait = 5 * time.Millisecond
	t.Cleanup(func() {
		holidayRefreshInterval = oldInterval
		holidayRefreshStartupWait = oldStartup
	})

	var called int32
	done := make(chan struct{})
	uc := &holidayUseCaseStub{
		refreshFn: func(context.Context, holidayportin.RefreshRangeInput) (*holidayportin.RefreshRangeResult, error) {
			if atomic.AddInt32(&called, 1) == 1 {
				close(done)
			}
			return nil, fmt.Errorf("boom")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runHolidayBackgroundRefresh(ctx, uc)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected RefreshRange to be called")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("holiday refresh worker did not stop on context cancel")
	}
}

func TestMainBootHealthAndMe(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("close reserved listener: %v", err)
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}

	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("SERVER_PORT", port)
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("STORAGE_DATA_PATH", t.TempDir())
	t.Setenv("STORAGE_THUMB_PATH", t.TempDir())
	t.Setenv("WEB_URL", "http://localhost:39001")
	t.Setenv("ADMIN_URL", "http://localhost:39002")
	t.Setenv("API_URL", "http://localhost:"+port)

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()

	baseURL := "http://127.0.0.1:" + port
	var healthResp *http.Response
	for i := 0; i < 60; i++ {
		healthResp, err = http.Get(baseURL + "/api/v1/health")
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected health status: %d", healthResp.StatusCode)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/me", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+signed)

	meResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("me request failed: %v", err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected me status: %d", meResp.StatusCode)
	}
	var payload map[string]string
	if err := json.NewDecoder(meResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if payload["user_id"] != "user-123" {
		t.Fatalf("unexpected user_id: %q", payload["user_id"])
	}

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("main did not exit after SIGINT")
	}
}

func TestMain_ConfigLoadError_Exits(t *testing.T) {
	oldLoadConfig := loadConfig
	oldExit := exitProcess
	loadConfig = func() (*config.Config, error) {
		return nil, fmt.Errorf("load failed")
	}
	var exitCode int
	exitProcess = func(code int) { exitCode = code }
	t.Cleanup(func() {
		loadConfig = oldLoadConfig
		exitProcess = oldExit
	})

	main()

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestMain_NewDBPoolError_Exits(t *testing.T) {
	oldLoadConfig := loadConfig
	oldNewDBPool := newDBPool
	oldExit := exitProcess
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Database: config.DatabaseConfig{URL: "postgres://invalid"},
		}, nil
	}
	newDBPool = func(context.Context, string) (*pgxpool.Pool, error) {
		return nil, fmt.Errorf("connect failed")
	}
	var exitCode int
	exitProcess = func(code int) { exitCode = code }
	t.Cleanup(func() {
		loadConfig = oldLoadConfig
		newDBPool = oldNewDBPool
		exitProcess = oldExit
	})

	main()

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestMain_PingError_Exits(t *testing.T) {
	oldLoadConfig := loadConfig
	oldNewDBPool := newDBPool
	oldPing := pingDBPool
	oldClose := closeDBPool
	oldExit := exitProcess

	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			Database: config.DatabaseConfig{URL: "postgres://ok"},
		}, nil
	}
	newDBPool = func(context.Context, string) (*pgxpool.Pool, error) {
		return &pgxpool.Pool{}, nil
	}
	pingDBPool = func(context.Context, *pgxpool.Pool) error {
		return fmt.Errorf("ping failed")
	}
	closeDBPool = func(*pgxpool.Pool) {}
	var exitCode int
	exitProcess = func(code int) { exitCode = code }
	t.Cleanup(func() {
		loadConfig = oldLoadConfig
		newDBPool = oldNewDBPool
		pingDBPool = oldPing
		closeDBPool = oldClose
		exitProcess = oldExit
	})

	main()

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestMain_ListenAndServeError_Exits(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	oldListen := listenAndServeHTTPServer
	oldShutdown := shutdownHTTPServer
	oldExit := exitProcess
	listenAndServeHTTPServer = func(*http.Server) error { return fmt.Errorf("listen failed") }
	shutdownHTTPServer = func(*http.Server, context.Context) error { return nil }
	var exitCode int32
	exitProcess = func(code int) { atomic.StoreInt32(&exitCode, int32(code)) }
	t.Cleanup(func() {
		listenAndServeHTTPServer = oldListen
		shutdownHTTPServer = oldShutdown
		exitProcess = oldExit
	})

	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("REDIS_URL", "://invalid")
	t.Setenv("SERVER_PORT", "0")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("STORAGE_DATA_PATH", t.TempDir())
	t.Setenv("STORAGE_THUMB_PATH", t.TempDir())
	t.Setenv("WEB_URL", "http://localhost:39001")
	t.Setenv("ADMIN_URL", "http://localhost:39002")
	t.Setenv("API_URL", "http://localhost:38117")

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("main did not exit after SIGINT")
	}
	if atomic.LoadInt32(&exitCode) != 1 {
		t.Fatalf("expected exit code 1 from listen error, got %d", atomic.LoadInt32(&exitCode))
	}
}

func TestMain_ShutdownError_Exits(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	oldListen := listenAndServeHTTPServer
	oldShutdown := shutdownHTTPServer
	oldExit := exitProcess
	listenAndServeHTTPServer = func(*http.Server) error { return http.ErrServerClosed }
	shutdownHTTPServer = func(*http.Server, context.Context) error { return fmt.Errorf("shutdown failed") }
	var exitCode int32
	exitProcess = func(code int) { atomic.StoreInt32(&exitCode, int32(code)) }
	t.Cleanup(func() {
		listenAndServeHTTPServer = oldListen
		shutdownHTTPServer = oldShutdown
		exitProcess = oldExit
	})

	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("REDIS_URL", "://invalid")
	t.Setenv("SERVER_PORT", "0")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("STORAGE_DATA_PATH", t.TempDir())
	t.Setenv("STORAGE_THUMB_PATH", t.TempDir())
	t.Setenv("WEB_URL", "http://localhost:39001")
	t.Setenv("ADMIN_URL", "http://localhost:39002")
	t.Setenv("API_URL", "http://localhost:38117")

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("main did not exit after SIGINT")
	}
	if atomic.LoadInt32(&exitCode) != 1 {
		t.Fatalf("expected exit code 1 from shutdown error, got %d", atomic.LoadInt32(&exitCode))
	}
}
