package oauthstate

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	stateTTL = 10 * time.Minute
)

var (
	errInvalidState = errors.New("invalid oauth state")
)

func Generate(app, hmacKey string) (string, error) {
	if !isAllowedApp(app) {
		return "", fmt.Errorf("invalid app: %s", app)
	}
	if hmacKey == "" {
		return "", fmt.Errorf("empty hmac key")
	}

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	payload := strings.Join([]string{app, nonce, ts}, ".")
	sig := sign(payload, hmacKey)
	return payload + "." + sig, nil
}

func Verify(state, hmacKey string) (string, error) {
	if hmacKey == "" {
		return "", fmt.Errorf("empty hmac key")
	}

	parts := strings.Split(state, ".")
	if len(parts) != 4 {
		return "", errInvalidState
	}
	app, nonce, ts, gotSig := parts[0], parts[1], parts[2], parts[3]
	if app == "" || nonce == "" || ts == "" || gotSig == "" {
		return "", errInvalidState
	}
	if !isAllowedApp(app) {
		return "", errInvalidState
	}

	issuedAt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return "", errInvalidState
	}
	now := time.Now()
	issuedAtTime := time.Unix(issuedAt, 0)
	if issuedAtTime.After(now.Add(1 * time.Minute)) {
		return "", errInvalidState
	}
	if now.Sub(issuedAtTime) > stateTTL {
		return "", errInvalidState
	}

	payload := strings.Join([]string{app, nonce, ts}, ".")
	expected := sign(payload, hmacKey)
	if !hmac.Equal([]byte(expected), []byte(gotSig)) {
		return "", errInvalidState
	}

	return app, nil
}

func sign(payload, hmacKey string) string {
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func isAllowedApp(app string) bool {
	return app == "web" || app == "admin"
}
