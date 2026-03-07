package oauthstate

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGenerateAndVerify(t *testing.T) {
	state, err := Generate("admin", "test-secret")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	app, err := Verify(state, "test-secret")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if app != "admin" {
		t.Fatalf("expected app=admin, got %s", app)
	}
}

func TestVerifyTamperedState(t *testing.T) {
	state, err := Generate("web", "test-secret")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	tampered := state + "x"
	if _, err := Verify(tampered, "test-secret"); err == nil {
		t.Fatal("expected verification error for tampered state")
	}
}

func TestGenerateInvalidInput(t *testing.T) {
	if _, err := Generate("mobile", "test-secret"); err == nil {
		t.Fatal("expected invalid app error")
	}
	if _, err := Generate("web", ""); err == nil {
		t.Fatal("expected empty hmac key error")
	}
}

func TestGenerateNonceError(t *testing.T) {
	prev := randRead
	t.Cleanup(func() { randRead = prev })
	randRead = func([]byte) (int, error) {
		return 0, errors.New("rand failed")
	}

	if _, err := Generate("web", "test-secret"); err == nil {
		t.Fatal("expected nonce generation error")
	}
}

func TestVerifyInvalidInput(t *testing.T) {
	if _, err := Verify("", "test-secret"); err == nil {
		t.Fatal("expected invalid state error for empty state")
	}
	if _, err := Verify("web.a.b.c", ""); err == nil {
		t.Fatal("expected empty hmac key error")
	}
}

func TestVerifyRejectsFutureAndExpiredState(t *testing.T) {
	key := "test-secret"
	now := time.Now()
	futureTS := strconv.FormatInt(now.Add(2*time.Minute).Unix(), 10)
	expiredTS := strconv.FormatInt(now.Add(-11*time.Minute).Unix(), 10)

	baseFuture := strings.Join([]string{"web", "nonce", futureTS}, ".")
	stateFuture := baseFuture + "." + sign(baseFuture, key)
	if _, err := Verify(stateFuture, key); err == nil {
		t.Fatal("expected future-issued state to fail")
	}

	baseExpired := strings.Join([]string{"admin", "nonce", expiredTS}, ".")
	stateExpired := baseExpired + "." + sign(baseExpired, key)
	if _, err := Verify(stateExpired, key); err == nil {
		t.Fatal("expected expired state to fail")
	}
}

func TestVerifyRejectsInvalidTimestampAndApp(t *testing.T) {
	key := "test-secret"
	baseBadTS := "web.nonce.notint"
	stateBadTS := baseBadTS + "." + sign(baseBadTS, key)
	if _, err := Verify(stateBadTS, key); err == nil {
		t.Fatal("expected invalid timestamp state to fail")
	}

	baseBadApp := "mobile.nonce.1700000000"
	stateBadApp := baseBadApp + "." + sign(baseBadApp, key)
	if _, err := Verify(stateBadApp, key); err == nil {
		t.Fatal("expected invalid app state to fail")
	}

	if _, err := Verify("web..1700000000.sig", key); err == nil {
		t.Fatal("expected empty state part to fail")
	}
}

func TestIsAllowedApp(t *testing.T) {
	if !isAllowedApp("web") {
		t.Fatal("web should be allowed")
	}
	if !isAllowedApp("admin") {
		t.Fatal("admin should be allowed")
	}
	if isAllowedApp("mobile") {
		t.Fatal("mobile should not be allowed")
	}
}
