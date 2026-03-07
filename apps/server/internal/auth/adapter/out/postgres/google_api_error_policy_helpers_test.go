package postgres

import (
	"errors"
	"strings"
	"testing"

	portout "lifebase/internal/auth/port/out"
)

func TestGoogleAPIErrorPolicyHelpers(t *testing.T) {
	p := classifyGoogleAPIError(nil)
	if !p.Retryable || p.UserMessage == "" {
		t.Fatalf("nil error policy mismatch: %#v", p)
	}

	plain := errors.New("plain")
	p = classifyGoogleAPIError(plain)
	if !p.Retryable {
		t.Fatalf("non-google error should keep default retryable policy: %#v", p)
	}

	auth403 := &portout.GoogleAPIError{StatusCode: 403, Reason: "authError"}
	if !isGoogleAuthError(auth403) {
		t.Fatal("authError should require reauth")
	}
	if isRetryableGoogleError(auth403) {
		t.Fatal("authError should not be retryable")
	}

	unknown403 := &portout.GoogleAPIError{StatusCode: 403, Reason: "somethingElse"}
	if isRetryableGoogleError(unknown403) {
		t.Fatal("unknown 403 should not be retryable")
	}

	dup409 := &portout.GoogleAPIError{StatusCode: 409, Reason: "duplicate"}
	if isRetryableGoogleError(dup409) {
		t.Fatal("duplicate 409 should not be retryable")
	}

	resync410 := &portout.GoogleAPIError{StatusCode: 410, Reason: "updatedMinTooLongAgo"}
	if !shouldResetGoogleSyncToken(resync410) {
		t.Fatal("410 updatedMinTooLongAgo should require full resync")
	}
	other410 := &portout.GoogleAPIError{StatusCode: 410, Reason: "other"}
	if shouldResetGoogleSyncToken(other410) {
		t.Fatal("410 other reason should not require hard reset")
	}

	precond412 := &portout.GoogleAPIError{StatusCode: 412}
	if !isRetryableGoogleError(precond412) {
		t.Fatal("412 should be retryable")
	}
	unknown := &portout.GoogleAPIError{StatusCode: 499, Message: "custom message"}
	if msg := googleSyncErrorMessage(unknown); msg != "custom message" {
		t.Fatalf("expected custom fallback message, got %q", msg)
	}

	if !isGoogleStatus(auth403, 403) || isGoogleStatus(auth403, 404) {
		t.Fatal("isGoogleStatus mismatch")
	}
	if !isGoogleReason(auth403, "AUTHERROR") || isGoogleReason(auth403, "quotaExceeded") {
		t.Fatal("isGoogleReason mismatch")
	}

	short := shortenText("  abc  ")
	if short != "abc" {
		t.Fatalf("expected trimmed short text, got %q", short)
	}
	long := strings.Repeat("x", 700)
	if got := shortenText(long); len(got) != 512 {
		t.Fatalf("expected truncated text len 512, got %d", len(got))
	}
}

func TestClassifyGoogleAPIErrorStatusMatrix(t *testing.T) {
	cases := []struct {
		name              string
		err               error
		retryable         bool
		reauth            bool
		fullResync        bool
		userMessagePart   string
	}{
		{name: "400_time_range_empty", err: &portout.GoogleAPIError{StatusCode: 400, Reason: "timeRangeEmpty"}, retryable: false, userMessagePart: "조회 기간"},
		{name: "400_generic", err: &portout.GoogleAPIError{StatusCode: 400, Reason: "badRequest"}, retryable: false, userMessagePart: "입력값"},
		{name: "401_reauth", err: &portout.GoogleAPIError{StatusCode: 401, Reason: "authError"}, retryable: false, reauth: true, userMessagePart: "다시 연결"},
		{name: "403_rate_limit", err: &portout.GoogleAPIError{StatusCode: 403, Reason: "quotaExceeded"}, retryable: true, userMessagePart: "자동 재시도"},
		{name: "403_forbidden_non_organizer", err: &portout.GoogleAPIError{StatusCode: 403, Reason: "forbiddenForNonOrganizer"}, retryable: false, userMessagePart: "주최자 권한"},
		{name: "404_missing", err: &portout.GoogleAPIError{StatusCode: 404, Reason: "notFound"}, retryable: false, userMessagePart: "찾을 수 없습니다"},
		{name: "409_conflict", err: &portout.GoogleAPIError{StatusCode: 409, Reason: "conflict"}, retryable: true, userMessagePart: "충돌"},
		{name: "410_deleted", err: &portout.GoogleAPIError{StatusCode: 410, Reason: "deleted"}, retryable: false, fullResync: true, userMessagePart: "전체 재동기화"},
		{name: "410_other", err: &portout.GoogleAPIError{StatusCode: 410, Reason: "other"}, retryable: false, userMessagePart: "전체 재동기화가 필요할 수 있습니다"},
		{name: "429_rate_limit", err: &portout.GoogleAPIError{StatusCode: 429}, retryable: true, userMessagePart: "요청이 너무 많습니다"},
		{name: "500_server", err: &portout.GoogleAPIError{StatusCode: 500, Reason: "backendError"}, retryable: true, userMessagePart: "서버 일시 오류"},
		{name: "fallback_default_message", err: &portout.GoogleAPIError{StatusCode: 418}, retryable: false, userMessagePart: "Google API 요청이 실패했습니다."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := classifyGoogleAPIError(tc.err)
			if p.Retryable != tc.retryable || p.RequiresReauth != tc.reauth || p.RequiresFullResync != tc.fullResync {
				t.Fatalf("policy mismatch: %#v", p)
			}
			if !strings.Contains(p.UserMessage, tc.userMessagePart) {
				t.Fatalf("expected message containing %q, got %q", tc.userMessagePart, p.UserMessage)
			}
		})
	}
}
