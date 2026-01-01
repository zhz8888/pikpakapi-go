package token

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	accessToken := "test_access_token"
	refreshToken := "test_refresh_token"

	encoded, err := Encode(accessToken, refreshToken)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if encoded == "" {
		t.Error("Encoded token should not be empty")
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.AccessToken != accessToken {
		t.Errorf("AccessToken mismatch: expected %s, got %s", accessToken, decoded.AccessToken)
	}

	if decoded.RefreshToken != refreshToken {
		t.Errorf("RefreshToken mismatch: expected %s, got %s", refreshToken, decoded.RefreshToken)
	}
}

func TestDecodeInvalidToken(t *testing.T) {
	_, err := Decode("invalid_base64_token")
	if err == nil {
		t.Error("Decode should fail for invalid token")
	}
}

func TestDecodeEmptyFields(t *testing.T) {
	data := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		AccessToken:  "",
		RefreshToken: "",
	}

	jsonData, _ := json.Marshal(data)
	encoded := base64.StdEncoding.EncodeToString(jsonData)

	_, err := Decode(encoded)
	if err == nil {
		t.Error("Decode should fail for token with empty fields")
	}
}
