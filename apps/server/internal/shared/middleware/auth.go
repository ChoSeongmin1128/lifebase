package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"lifebase/internal/shared/response"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const AuthAppKey contextKey = "auth_app"

var parseJWTToken = jwt.Parse

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken, app, headerErr := resolveAuthToken(r)
			if headerErr != nil {
				response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid authorization header format")
				return
			}
			if rawToken == "" {
				response.Error(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header is required")
				return
			}

			token, err := parseJWTToken(rawToken, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token claims")
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "missing user ID in token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			if app != "" {
				ctx = context.WithValue(ctx, AuthAppKey, app)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func resolveAuthToken(r *http.Request) (string, string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return "", "", jwt.ErrSignatureInvalid
		}
		return parts[1], "", nil
	}

	if cookie, app := authCookieForRequest(r); cookie != nil {
		return cookie.Value, app, nil
	}

	return "", "", nil
}

func authCookieForRequest(r *http.Request) (*http.Cookie, string) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/admin") {
		if cookie, err := r.Cookie("lifebase_admin_access_token"); err == nil && cookie.Value != "" {
			return cookie, "admin"
		}
		return nil, ""
	}

	if cookie, err := r.Cookie("lifebase_access_token"); err == nil && cookie.Value != "" {
		return cookie, "web"
	}
	if path == "/api/v1/auth/logout" {
		if cookie, err := r.Cookie("lifebase_admin_access_token"); err == nil && cookie.Value != "" {
			return cookie, "admin"
		}
	}

	return nil, ""
}

func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

func GetAuthApp(ctx context.Context) string {
	if v, ok := ctx.Value(AuthAppKey).(string); ok {
		return v
	}
	return ""
}
