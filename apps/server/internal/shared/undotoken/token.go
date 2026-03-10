package undotoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	ActionMoveFile   = "move_file"
	ActionMoveFolder = "move_folder"
	ActionCopyFile   = "copy_file"

	defaultTTL = 15 * time.Second
)

var errInvalidToken = errors.New("invalid undo token")

type Claims struct {
	Action    string  `json:"action"`
	UserID    string  `json:"user_id"`
	ItemID    string  `json:"item_id"`
	ParentID  *string `json:"parent_id,omitempty"`
	IssuedAt  int64   `json:"iat"`
	ExpiresAt int64   `json:"exp"`
}

func GenerateMoveFile(userID, fileID string, parentID *string, hmacKey string) (string, error) {
	return generate(Claims{
		Action:   ActionMoveFile,
		UserID:   userID,
		ItemID:   fileID,
		ParentID: parentID,
	}, hmacKey, defaultTTL)
}

func GenerateMoveFolder(userID, folderID string, parentID *string, hmacKey string) (string, error) {
	return generate(Claims{
		Action:   ActionMoveFolder,
		UserID:   userID,
		ItemID:   folderID,
		ParentID: parentID,
	}, hmacKey, defaultTTL)
}

func GenerateCopyFile(userID, fileID string, hmacKey string) (string, error) {
	return generate(Claims{
		Action: ActionCopyFile,
		UserID: userID,
		ItemID: fileID,
	}, hmacKey, defaultTTL)
}

func Verify(token, hmacKey string) (*Claims, error) {
	if token == "" || hmacKey == "" {
		return nil, errInvalidToken
	}

	parts := splitToken(token)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, errInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errInvalidToken
	}
	if !hmac.Equal([]byte(sign(parts[0], hmacKey)), []byte(parts[1])) {
		return nil, errInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errInvalidToken
	}
	if claims.Action == "" || claims.UserID == "" || claims.ItemID == "" {
		return nil, errInvalidToken
	}

	now := time.Now()
	issuedAt := time.Unix(claims.IssuedAt, 0)
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	if issuedAt.After(now.Add(time.Minute)) || now.After(expiresAt) {
		return nil, errInvalidToken
	}

	switch claims.Action {
	case ActionMoveFile, ActionMoveFolder, ActionCopyFile:
		return &claims, nil
	default:
		return nil, errInvalidToken
	}
}

func generate(base Claims, hmacKey string, ttl time.Duration) (string, error) {
	if hmacKey == "" {
		return "", fmt.Errorf("empty hmac key")
	}
	if base.Action == "" || base.UserID == "" || base.ItemID == "" {
		return "", fmt.Errorf("invalid undo claims")
	}
	if ttl <= 0 {
		ttl = defaultTTL
	}

	now := time.Now()
	base.IssuedAt = now.Unix()
	base.ExpiresAt = now.Add(ttl).Unix()

	payload, err := json.Marshal(base)
	if err != nil {
		return "", fmt.Errorf("marshal undo token: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + sign(encoded, hmacKey), nil
}

func sign(payload, hmacKey string) string {
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func splitToken(token string) []string {
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			return []string{token[:i], token[i+1:]}
		}
	}
	return []string{token}
}
