package utils

import (
	"testing"
)

func TestEncodeToken(t *testing.T) {
	accessToken := "test_access_token"
	refreshToken := "test_refresh_token"

	encoded, err := EncodeToken(accessToken, refreshToken)
	if err != nil {
		t.Fatalf("Failed to encode token: %v", err)
	}

	if encoded == "" {
		t.Error("Expected non-empty encoded token")
	}

	data, err := DecodeToken(encoded)
	if err != nil {
		t.Fatalf("Failed to decode token: %v", err)
	}

	if data.AccessToken != accessToken {
		t.Errorf("Expected access token '%s', got '%s'", accessToken, data.AccessToken)
	}

	if data.RefreshToken != refreshToken {
		t.Errorf("Expected refresh token '%s', got '%s'", refreshToken, data.RefreshToken)
	}
}

func TestDecodeToken_Invalid(t *testing.T) {
	_, err := DecodeToken("invalid_base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestDecodeToken_MissingFields(t *testing.T) {
	encoded := "eyJ1c2VyX2lkIjoidGVzdCJ9" // {"user_id":"test"}
	_, err := DecodeToken(encoded)
	if err == nil {
		t.Error("Expected error for missing access_token or refresh_token")
	}
}
