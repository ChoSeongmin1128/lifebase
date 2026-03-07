package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	portout "lifebase/internal/auth/port/out"
)

func TestGooglePushProcessorPureHelpers(t *testing.T) {
	if d := nextRetryDelay(0); d != 10*time.Second {
		t.Fatalf("attempt 0 should map to 10s, got %s", d)
	}
	if d := nextRetryDelay(3); d != 40*time.Second {
		t.Fatalf("attempt 3 should map to 40s, got %s", d)
	}
	if d := nextRetryDelay(100); d != 1280*time.Second {
		t.Fatalf("attempt cap should map to 1280s, got %s", d)
	}

	if got := shortenError(nil); got != "" {
		t.Fatalf("nil error should return empty string, got %q", got)
	}
	if got := shortenError(errors.New(" x ")); got != "x" {
		t.Fatalf("shortenError trim failed: %q", got)
	}
	longErr := errors.New(strings.Repeat("a", 600))
	if got := shortenError(longErr); len(got) != 512 {
		t.Fatalf("shortenError long len mismatch: %d", len(got))
	}

	if got := pgInterval(500 * time.Millisecond); got != "1 seconds" {
		t.Fatalf("interval lower bound mismatch: %q", got)
	}
	if got := pgInterval(5 * time.Second); got != "5 seconds" {
		t.Fatalf("interval formatting mismatch: %q", got)
	}
}

func TestGoogleSyncCoordinatorPureHelpers(t *testing.T) {
	if minIntervalForReason("hourly") != time.Hour {
		t.Fatal("hourly interval mismatch")
	}
	if minIntervalForReason("background") != 10*time.Minute {
		t.Fatal("background interval mismatch")
	}
	if minIntervalForReason("tab_heartbeat") != 10*time.Minute {
		t.Fatal("tab_heartbeat interval mismatch")
	}
	if minIntervalForReason("page_action") != 15*time.Second {
		t.Fatal("page_action interval mismatch")
	}
	if minIntervalForReason("page_enter") != 0 || minIntervalForReason("manual") != 0 {
		t.Fatal("page_enter/manual interval mismatch")
	}
	if minIntervalForReason("unknown") != 15*time.Second {
		t.Fatal("default interval mismatch")
	}

	k1 := advisoryLockKey("same-key")
	k2 := advisoryLockKey("same-key")
	k3 := advisoryLockKey("other-key")
	if k1 != k2 {
		t.Fatal("advisoryLockKey should be deterministic")
	}
	if k1 == k3 {
		t.Fatal("different keys should produce different lock values")
	}
}

func TestGoogleSyncerCompletedAt(t *testing.T) {
	now := time.Now()
	if got := completedAt(portout.OAuthTask{IsDone: false}, now); got != nil {
		t.Fatal("not done task should not have completed_at")
	}
	existing := now.Add(-time.Hour)
	if got := completedAt(portout.OAuthTask{IsDone: true, CompletedAt: &existing}, now); got != &existing {
		t.Fatal("existing completed_at should be reused")
	}
	if got := completedAt(portout.OAuthTask{IsDone: true}, now); got == nil || !got.Equal(now) {
		t.Fatalf("done task should fallback to now, got %v", got)
	}
}

