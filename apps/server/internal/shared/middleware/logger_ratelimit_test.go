package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoggerDefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "127.0.0.1:1000"

	Logger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLoggerWriteHeaderStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/teapot", nil)
	req.RemoteAddr = "127.0.0.1:1001"

	Logger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("teapot"))
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected 418, got %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "teapot" {
		t.Fatalf("expected body teapot, got %q", got)
	}
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(120)
	if rl == nil {
		t.Fatal("expected rate limiter")
	}
	if rl.rate != 2 {
		t.Fatalf("expected 2 req/s, got %f", rl.rate)
	}
	if rl.burst != 120 {
		t.Fatalf("expected burst 120, got %f", rl.burst)
	}
}

func TestRateLimiterAllowsRequest(t *testing.T) {
	rl := &RateLimiter{
		visitors: map[string]*visitor{},
		rate:     1,
		burst:    2,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.RemoteAddr = "1.2.3.4:9999"

	called := false
	rl.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if !called {
		t.Fatal("next was not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestRateLimiterRejectsWhenNoToken(t *testing.T) {
	rl := &RateLimiter{
		visitors: map[string]*visitor{
			"1.2.3.4:9999": {
				tokens:   0,
				lastSeen: time.Now(),
			},
		},
		rate:  0,
		burst: 1,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "1.2.3.4:9999"

	rl.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimiterCapsRefilledTokensToBurst(t *testing.T) {
	rl := &RateLimiter{
		visitors: map[string]*visitor{
			"1.2.3.4:9999": {
				tokens:   1,
				lastSeen: time.Now().Add(-10 * time.Second),
			},
		},
		rate:  100,
		burst: 3,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/burst", nil)
	req.RemoteAddr = "1.2.3.4:9999"

	rl.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	got := rl.visitors["1.2.3.4:9999"].tokens
	if got < 1.9 || got > 2.1 {
		t.Fatalf("expected remaining tokens near 2 after capped refill, got %f", got)
	}
}

func TestRateLimiterCleanupExpired(t *testing.T) {
	now := time.Now()
	rl := &RateLimiter{
		visitors: map[string]*visitor{
			"old": {tokens: 1, lastSeen: now.Add(-6 * time.Minute)},
			"new": {tokens: 1, lastSeen: now.Add(-2 * time.Minute)},
		},
	}

	rl.cleanupExpired(now)

	if _, ok := rl.visitors["old"]; ok {
		t.Fatal("expected expired visitor to be removed")
	}
	if _, ok := rl.visitors["new"]; !ok {
		t.Fatal("expected recent visitor to remain")
	}
}
