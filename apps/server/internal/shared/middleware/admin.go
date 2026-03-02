package middleware

import (
	"context"
	"net/http"

	"lifebase/internal/shared/response"
)

type AdminChecker interface {
	IsActiveAdmin(ctx context.Context, userID string) (bool, error)
}

func Admin(checker AdminChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
				return
			}
			ok, err := checker.IsActiveAdmin(r.Context(), userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "ADMIN_CHECK_FAILED", "admin check failed")
				return
			}
			if !ok {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "admin access denied")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
