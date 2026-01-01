package crypto

import (
	"testing"
)

func TestMD5Hash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "5d41402abc4b2a76b9719d911017c592"},
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"123456", "e10adc3949ba59abbe56e057f20f883e"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := MD5Hash(tt.input); got != tt.expected {
				t.Errorf("MD5Hash(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMD5HashBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "5d41402abc4b2a76b9719d911017c592"},
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := MD5HashBytes([]byte(tt.input)); got != tt.expected {
				t.Errorf("MD5HashBytes(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSHA1Hash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"},
		{"", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"123456", "7c4a8d09ca3762af61e59520943dc26494f8941b"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := SHA1Hash(tt.input); got != tt.expected {
				t.Errorf("SHA1Hash(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDoubleHash(t *testing.T) {
	input := "hello"
	expected := "e69d7e620e82be5eb414d1f8d1d4b9d9"
	if got := DoubleHash(input); got != expected {
		t.Errorf("DoubleHash(%s) = %s, want %s", input, got, expected)
	}
}

func TestMD5Hash_EmptyString(t *testing.T) {
	result := MD5Hash("")
	expected := "d41d8cd98f00b204e9800998ecf8427e"
	if result != expected {
		t.Errorf("MD5Hash(\"\") = %s, want %s", result, expected)
	}
}

func TestSHA1Hash_EmptyString(t *testing.T) {
	result := SHA1Hash("")
	expected := "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	if result != expected {
		t.Errorf("SHA1Hash(\"\") = %s, want %s", result, expected)
	}
}
