package postgres

import (
	"testing"

	portout "lifebase/internal/auth/port/out"
)

func TestClassifyGoogleAPIError(t *testing.T) {
	tests := []struct {
		name                string
		err                 error
		retryable           bool
		requiresReauth      bool
		requiresFullResync  bool
		expectedUserMessage string
	}{
		{
			name: "400 timeRangeEmpty",
			err: &portout.GoogleAPIError{
				StatusCode: 400,
				Reason:     "timeRangeEmpty",
				Message:    "Bad Request",
			},
			retryable:           false,
			requiresReauth:      false,
			requiresFullResync:  false,
			expectedUserMessage: "조회 기간이 올바르지 않습니다. 시작/종료 시간을 확인해 주세요.",
		},
		{
			name: "401 auth error",
			err: &portout.GoogleAPIError{
				StatusCode: 401,
				Reason:     "authError",
				Message:    "Invalid Credentials",
			},
			retryable:           false,
			requiresReauth:      true,
			requiresFullResync:  false,
			expectedUserMessage: "Google 계정 인증이 만료되었습니다. 계정을 다시 연결해 주세요.",
		},
		{
			name: "403 rate limit",
			err: &portout.GoogleAPIError{
				StatusCode: 403,
				Reason:     "rateLimitExceeded",
				Message:    "Rate limit exceeded",
			},
			retryable:           true,
			requiresReauth:      false,
			requiresFullResync:  false,
			expectedUserMessage: "Google API 요청 한도에 도달했습니다. 잠시 후 자동 재시도합니다.",
		},
		{
			name: "410 full sync required",
			err: &portout.GoogleAPIError{
				StatusCode: 410,
				Reason:     "fullSyncRequired",
				Message:    "sync token is no longer valid",
			},
			retryable:           false,
			requiresReauth:      false,
			requiresFullResync:  true,
			expectedUserMessage: "동기화 토큰이 만료되어 전체 재동기화가 필요합니다.",
		},
		{
			name: "429 too many requests",
			err: &portout.GoogleAPIError{
				StatusCode: 429,
				Reason:     "rateLimitExceeded",
				Message:    "Too Many Requests",
			},
			retryable:           true,
			requiresReauth:      false,
			requiresFullResync:  false,
			expectedUserMessage: "요청이 너무 많습니다. 잠시 후 자동 재시도합니다.",
		},
		{
			name: "500 backend error",
			err: &portout.GoogleAPIError{
				StatusCode: 500,
				Reason:     "backendError",
				Message:    "Backend Error",
			},
			retryable:           true,
			requiresReauth:      false,
			requiresFullResync:  false,
			expectedUserMessage: "Google 서버 일시 오류가 발생했습니다. 잠시 후 자동 재시도합니다.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyGoogleAPIError(tt.err)
			if got.Retryable != tt.retryable {
				t.Fatalf("retryable mismatch: got=%v want=%v", got.Retryable, tt.retryable)
			}
			if got.RequiresReauth != tt.requiresReauth {
				t.Fatalf("requiresReauth mismatch: got=%v want=%v", got.RequiresReauth, tt.requiresReauth)
			}
			if got.RequiresFullResync != tt.requiresFullResync {
				t.Fatalf("requiresFullResync mismatch: got=%v want=%v", got.RequiresFullResync, tt.requiresFullResync)
			}
			if got.UserMessage != tt.expectedUserMessage {
				t.Fatalf("user message mismatch:\n got: %q\nwant: %q", got.UserMessage, tt.expectedUserMessage)
			}
		})
	}
}
