package postgres

import (
	"errors"
	"strings"

	portout "lifebase/internal/auth/port/out"
)

type googleAPIErrorPolicy struct {
	StatusCode         int
	Reason             string
	Retryable          bool
	RequiresReauth     bool
	RequiresFullResync bool
	UserMessage        string
}

func classifyGoogleAPIError(err error) googleAPIErrorPolicy {
	policy := googleAPIErrorPolicy{
		Retryable:   true,
		UserMessage: "Google API 요청 처리 중 일시 오류가 발생했습니다. 잠시 후 다시 시도해 주세요.",
	}
	if err == nil {
		return policy
	}

	var apiErr *portout.GoogleAPIError
	if !errors.As(err, &apiErr) {
		return policy
	}

	reason := strings.ToLower(strings.TrimSpace(apiErr.Reason))
	policy.StatusCode = apiErr.StatusCode
	policy.Reason = reason

	switch apiErr.StatusCode {
	case 400:
		policy.Retryable = false
		if reason == "timerangeempty" {
			policy.UserMessage = "조회 기간이 올바르지 않습니다. 시작/종료 시간을 확인해 주세요."
			return policy
		}
		policy.UserMessage = "Google 요청 값이 올바르지 않습니다. 입력값을 확인해 주세요."
		return policy
	case 401:
		policy.Retryable = false
		policy.RequiresReauth = true
		policy.UserMessage = "Google 계정 인증이 만료되었습니다. 계정을 다시 연결해 주세요."
		return policy
	case 403:
		switch reason {
		case "autherror", "invalidcredentials":
			policy.Retryable = false
			policy.RequiresReauth = true
			policy.UserMessage = "Google 계정 인증이 만료되었습니다. 계정을 다시 연결해 주세요."
		case "ratelimitexceeded", "userratelimitexceeded", "quotaexceeded":
			policy.Retryable = true
			policy.UserMessage = "Google API 요청 한도에 도달했습니다. 잠시 후 자동 재시도합니다."
		case "forbiddenfornonorganizer":
			policy.Retryable = false
			policy.UserMessage = "주최자 권한이 없어 일정 속성을 변경할 수 없습니다."
		default:
			policy.Retryable = false
			policy.UserMessage = "Google 리소스 접근 권한이 없습니다."
		}
		return policy
	case 404:
		policy.Retryable = false
		policy.UserMessage = "대상 Google 리소스를 찾을 수 없습니다."
		return policy
	case 409:
		switch reason {
		case "duplicate":
			policy.Retryable = false
			policy.UserMessage = "동일 ID 리소스가 이미 존재합니다. 다른 식별자로 다시 시도해 주세요."
		default:
			policy.Retryable = true
			policy.UserMessage = "Google 리소스 충돌이 발생했습니다. 잠시 후 다시 시도합니다."
		}
		return policy
	case 410:
		policy.Retryable = false
		if reason == "" || reason == "fullsyncrequired" || reason == "updatedmintoolongago" || reason == "deleted" {
			policy.RequiresFullResync = true
			policy.UserMessage = "동기화 토큰이 만료되어 전체 재동기화가 필요합니다."
			return policy
		}
		policy.UserMessage = "Google 동기화 상태가 만료되었습니다. 전체 재동기화가 필요할 수 있습니다."
		return policy
	case 412:
		policy.Retryable = true
		policy.UserMessage = "원격 데이터 버전이 변경되었습니다. 최신 상태를 반영해 다시 시도합니다."
		return policy
	case 429:
		policy.Retryable = true
		policy.UserMessage = "요청이 너무 많습니다. 잠시 후 자동 재시도합니다."
		return policy
	}

	if apiErr.StatusCode >= 500 && apiErr.StatusCode <= 599 {
		policy.Retryable = true
		policy.UserMessage = "Google 서버 일시 오류가 발생했습니다. 잠시 후 자동 재시도합니다."
		return policy
	}

	policy.Retryable = false
	if apiErr.Message != "" {
		policy.UserMessage = apiErr.Message
	} else {
		policy.UserMessage = "Google API 요청이 실패했습니다."
	}
	return policy
}

func googleSyncErrorMessage(err error) string {
	policy := classifyGoogleAPIError(err)
	return policy.UserMessage
}

func shortenText(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= 512 {
		return trimmed
	}
	return trimmed[:512]
}

func isRetryableGoogleError(err error) bool {
	return classifyGoogleAPIError(err).Retryable
}

func isGoogleAuthError(err error) bool {
	return classifyGoogleAPIError(err).RequiresReauth
}

func shouldResetGoogleSyncToken(err error) bool {
	return classifyGoogleAPIError(err).RequiresFullResync
}

func isGoogleStatus(err error, status int) bool {
	var apiErr *portout.GoogleAPIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}

func isGoogleReason(err error, reason string) bool {
	var apiErr *portout.GoogleAPIError
	return errors.As(err, &apiErr) && strings.EqualFold(apiErr.Reason, reason)
}
