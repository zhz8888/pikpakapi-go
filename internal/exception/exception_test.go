package exception

import (
	"errors"
	"testing"
)

func TestPikpakException(t *testing.T) {
	e := NewPikpakExceptionWithMessage(ErrCodeUnknownError, "test error")
	if e.Error() != "[1006] test error" {
		t.Errorf("Expected '[1006] test error', got '%s'", e.Error())
	}
}

func TestPikpakExceptionWithError(t *testing.T) {
	originalErr := errors.New("original error")
	e := NewPikpakExceptionWithError(ErrCodeUnknownError, originalErr)
	if e.Error() != "[1006] unknown error: original error" {
		t.Errorf("Expected '[1006] unknown error: original error', got '%s'", e.Error())
	}
}

func TestIsPikpakException(t *testing.T) {
	e := NewPikpakExceptionWithMessage(ErrCodeUnknownError, "test")
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
		{"InvalidUsernamePassword", ErrInvalidUsernamePassword, "[1001] invalid username or password"},
		{"InvalidEncodedToken", ErrInvalidEncodedToken, "[1002] invalid encoded token"},
		{"CaptchaTokenFailed", ErrCaptchaTokenFailed, "[1003] captcha token get failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}
