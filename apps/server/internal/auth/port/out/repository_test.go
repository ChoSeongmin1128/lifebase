package out

import "testing"

func TestGoogleAPIErrorError(t *testing.T) {
	var nilErr *GoogleAPIError
	if got := nilErr.Error(); got != "google api error" {
		t.Fatalf("expected default message, got %q", got)
	}
	e := &GoogleAPIError{Message: "boom"}
	if got := e.Error(); got != "boom" {
		t.Fatalf("expected message passthrough, got %q", got)
	}
}

