package undotoken

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestGenerateAndVerify(t *testing.T) {
	token, err := GenerateMoveFile("u1", "f1", strPtr("p1"), "secret")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := Verify(token, "secret")
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.Action != ActionMoveFile || claims.UserID != "u1" || claims.ItemID != "f1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if claims.ParentID == nil || *claims.ParentID != "p1" {
		t.Fatalf("unexpected parent id: %#v", claims.ParentID)
	}
}

func TestVerifyRejectsTamperedAndExpired(t *testing.T) {
	token, err := GenerateCopyFile("u1", "f1", time.Now().UnixNano(), "secret")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if _, err := Verify(token+"x", "secret"); err == nil {
		t.Fatal("expected tampered token error")
	}

	payload := Claims{
		Action:       ActionCopyFile,
		UserID:       "u1",
		ItemID:       "f1",
		StateVersion: time.Now().UnixNano(),
		IssuedAt:     time.Now().Add(-time.Minute).Unix(),
		ExpiresAt:    time.Now().Add(-time.Second).Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	expired := encoded + "." + sign(encoded, "secret")
	if _, err := Verify(expired, "secret"); err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestVerifyRejectsCopyTokenWithoutStateVersion(t *testing.T) {
	payload := Claims{
		Action:    ActionCopyFile,
		UserID:    "u1",
		ItemID:    "f1",
		IssuedAt:  time.Now().Add(-time.Second).Unix(),
		ExpiresAt: time.Now().Add(time.Second).Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	token := encoded + "." + sign(encoded, "secret")
	if _, err := Verify(token, "secret"); err == nil {
		t.Fatal("expected missing state version error")
	}
}

func strPtr(v string) *string { return &v }
