package signer

import (
	"strconv"
	"strings"
	"testing"
)

func TestGetTimestamp(t *testing.T) {
	timestamp := GetTimestamp()
	if timestamp <= 0 {
		t.Errorf("GetTimestamp() = %d, want > 0", timestamp)
	}
}

func TestGetTimestamp_Increases(t *testing.T) {
	ts1 := GetTimestamp()
	ts2 := GetTimestamp()
	if ts2 < ts1 {
		t.Errorf("GetTimestamp() = %d, want >= %d", ts2, ts1)
	}
}

func TestCaptchaSign_Format(t *testing.T) {
	deviceID := "test_device_id"
	timestamp := "1234567890"

	sign := CaptchaSign(deviceID, timestamp)

	if !strings.HasPrefix(sign, "1.") {
		t.Errorf("CaptchaSign() = %s, want prefix '1.'", sign)
	}
}

func TestCaptchaSign_Deterministic(t *testing.T) {
	deviceID := "test_device_id"
	timestamp := "1234567890"

	sign1 := CaptchaSign(deviceID, timestamp)
	sign2 := CaptchaSign(deviceID, timestamp)

	if sign1 != sign2 {
		t.Errorf("CaptchaSign() not deterministic: %s != %s", sign1, sign2)
	}
}

func TestCaptchaSign_DifferentInputs(t *testing.T) {
	sign1 := CaptchaSign("device1", "1234567890")
	sign2 := CaptchaSign("device2", "1234567890")
	sign3 := CaptchaSign("device1", "0987654321")

	if sign1 == sign2 {
		t.Error("CaptchaSign() should produce different results for different device IDs")
	}
	if sign1 == sign3 {
		t.Error("CaptchaSign() should produce different results for different timestamps")
	}
}

func TestGenerateDeviceSign_Format(t *testing.T) {
	deviceID := "test_device_id"
	packageName := "com.pikpak"

	sign := GenerateDeviceSign(deviceID, packageName)

	if !strings.HasPrefix(sign, "div101.") {
		t.Errorf("GenerateDeviceSign() = %s, want prefix 'div101.'", sign)
	}
}

func TestGenerateDeviceSign_ContainsDeviceID(t *testing.T) {
	deviceID := "test_device_id"
	packageName := "com.pikpak"

	sign := GenerateDeviceSign(deviceID, packageName)

	if !strings.Contains(sign, deviceID) {
		t.Errorf("GenerateDeviceSign() = %s, should contain deviceID %s", sign, deviceID)
	}
}

func TestGenerateDeviceSign_Deterministic(t *testing.T) {
	deviceID := "test_device_id"
	packageName := "com.pikpak"

	sign1 := GenerateDeviceSign(deviceID, packageName)
	sign2 := GenerateDeviceSign(deviceID, packageName)

	if sign1 != sign2 {
		t.Errorf("GenerateDeviceSign() not deterministic: %s != %s", sign1, sign2)
	}
}

func TestGenerateDeviceSign_DifferentInputs(t *testing.T) {
	sign1 := GenerateDeviceSign("device1", "com.pikpak")
	sign2 := GenerateDeviceSign("device2", "com.pikpak")
	sign3 := GenerateDeviceSign("device1", "com.pikpak2")

	if sign1 == sign2 {
		t.Error("GenerateDeviceSign() should produce different results for different device IDs")
	}
	if sign1 == sign3 {
		t.Error("GenerateDeviceSign() should produce different results for different package names")
	}
}

func TestConstants(t *testing.T) {
	if ClientID == "" {
		t.Error("ClientID should not be empty")
	}
	if ClientVersion == "" {
		t.Error("ClientVersion should not be empty")
	}
	if PackageName == "" {
		t.Error("PackageName should not be empty")
	}
}

func TestSalts_Length(t *testing.T) {
	if len(salts) == 0 {
		t.Error("Salts slice should not be empty")
	}
}

func TestCaptchaSign_OutputLength(t *testing.T) {
	sign := CaptchaSign("test_device", "1234567890")
	expectedPrefix := "1."
	md5Length := 32

	if !strings.HasPrefix(sign, expectedPrefix) {
		t.Errorf("CaptchaSign() should start with '%s', got %s", expectedPrefix, sign)
	}

	withoutPrefix := strings.TrimPrefix(sign, expectedPrefix)
	if len(withoutPrefix) != md5Length {
		t.Errorf("CaptchaSign() MD5 part length = %d, want %d", len(withoutPrefix), md5Length)
	}
}

func TestGenerateDeviceSign_OutputLength(t *testing.T) {
	sign := GenerateDeviceSign("test_device_id", "com.pikpak")
	expectedPrefix := "div101."
	md5Length := 32

	if !strings.HasPrefix(sign, expectedPrefix) {
		t.Errorf("GenerateDeviceSign() should start with '%s', got %s", expectedPrefix, sign)
	}

	withoutPrefix := strings.TrimPrefix(sign, expectedPrefix)
	expectedLen := len("test_device_id") + md5Length
	if len(withoutPrefix) != expectedLen {
		t.Errorf("GenerateDeviceSign() part after prefix length = %d, want %d", len(withoutPrefix), expectedLen)
	}
}

func TestGetTimestamp_NotZero(t *testing.T) {
	ts := GetTimestamp()
	if ts == 0 {
		t.Error("GetTimestamp() should not return 0")
	}
}

func TestGetTimestamp_ReasonableRange(t *testing.T) {
	ts := GetTimestamp()
	now := strconv.FormatInt(ts, 10)
	if len(now) < 10 || len(now) > 14 {
		t.Errorf("GetTimestamp() = %s, length should be 10-14 digits (Unix timestamp in milliseconds)", now)
	}
}
