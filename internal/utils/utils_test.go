package utils

import (
	"testing"
)

func TestGetTimestamp(t *testing.T) {
	ts1 := GetTimestamp()
	if ts1 == 0 {
		t.Error("Expected non-zero timestamp")
	}

	ts2 := GetTimestamp()
	if ts2 < ts1 {
		t.Error("Expected timestamp to be increasing")
	}
}

func TestCaptchaSign(t *testing.T) {
	deviceID := "test_device_id"
	timestamp := "1234567890"

	sign := CaptchaSign(deviceID, timestamp)

	if sign == "" {
		t.Error("Expected non-empty sign")
	}

	if sign[:2] != "1." {
		t.Error("Expected sign to start with '1.'")
	}

	sign2 := CaptchaSign(deviceID, timestamp)
	if sign != sign2 {
		t.Error("Expected same inputs to produce same sign")
	}
}

func TestGenerateDeviceSign(t *testing.T) {
	deviceID := "test_device_id"
	packageName := "com.pikcloud.pikpak"

	sign := GenerateDeviceSign(deviceID, packageName)

	if sign == "" {
		t.Error("Expected non-empty device sign")
	}

	if sign[:7] != "div101." {
		t.Error("Expected device sign to start with 'div101.'")
	}

	sign2 := GenerateDeviceSign(deviceID, packageName)
	if sign != sign2 {
		t.Error("Expected same inputs to produce same device sign")
	}
}

func TestBuildCustomUserAgent(t *testing.T) {
	deviceID := "test_device_id"
	userID := "test_user_id"

	ua := BuildCustomUserAgent(deviceID, userID)

	if ua == "" {
		t.Error("Expected non-empty user agent")
	}

	if len(ua) < 100 {
		t.Error("Expected user agent to be reasonably long")
	}

	if !contains(ua, "ANDROID-") {
		t.Error("Expected user agent to contain 'ANDROID-'")
	}

	if !contains(ua, "clientid/") {
		t.Error("Expected user agent to contain 'clientid/'")
	}

	if !contains(ua, "deviceid/") {
		t.Error("Expected user agent to contain 'deviceid/'")
	}
}

func TestConstants(t *testing.T) {
	if ClientID == "" {
		t.Error("Expected ClientID to be non-empty")
	}
	if ClientSecret == "" {
		t.Error("Expected ClientSecret to be non-empty")
	}
	if ClientVersion == "" {
		t.Error("Expected ClientVersion to be non-empty")
	}
	if PackageName == "" {
		t.Error("Expected PackageName to be non-empty")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
