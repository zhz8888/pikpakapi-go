package exception

import (
	"errors"
	"testing"
)

func TestPikpakException(t *testing.T) {
	e := NewPikpakException("test error")
	if e.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", e.Error())
	}
}

func TestPikpakExceptionWithError(t *testing.T) {
	originalErr := errors.New("original error")
	e := NewPikpakExceptionWithError("wrapped error", originalErr)
	if e.Error() != "wrapped error: original error" {
		t.Errorf("Expected 'wrapped error: original error', got '%s'", e.Error())
	}
}

func TestIsPikpakException(t *testing.T) {
	e := NewPikpakException("test")
	if !IsPikpakException(e) {
		t.Error("Expected IsPikpakException to return true")
	}

	regularErr := errors.New("regular error")
	if IsPikpakException(regularErr) {
		t.Error("Expected IsPikpakException to return false for regular error")
	}
}

func TestErrorVariables(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"InvalidUsernamePassword", ErrInvalidUsernamePassword, "invalid username or password"},
		{"InvalidEncodedToken", ErrInvalidEncodedToken, "invalid encoded token"},
		{"CaptchaTokenFailed", ErrCaptchaTokenFailed, "captcha_token get failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}
