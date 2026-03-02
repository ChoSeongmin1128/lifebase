package oauthstate

import "testing"

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
