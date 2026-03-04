package usecase

import (
	"errors"
	"strings"
	"testing"

	authportout "lifebase/internal/auth/port/out"
)

func TestMapDeleteGoogleTaskListError_NotFoundIgnored(t *testing.T) {
	err := mapDeleteGoogleTaskListError(&authportout.GoogleAPIError{
		StatusCode: 404,
		Message:    "not found",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestMapDeleteGoogleTaskListError_DefaultListMessage(t *testing.T) {
	err := mapDeleteGoogleTaskListError(&authportout.GoogleAPIError{
		StatusCode: 400,
		Reason:     "invalid",
		Message:    "Invalid Value",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "기본 Tasks 목록") {
		t.Fatalf("expected default list message, got %q", err.Error())
	}
}

func TestMapDeleteGoogleTaskListError_WrapsUnknown(t *testing.T) {
	base := errors.New("network error")
	err := mapDeleteGoogleTaskListError(base)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "delete google task list") {
		t.Fatalf("expected wrapped message, got %q", err.Error())
	}
}
