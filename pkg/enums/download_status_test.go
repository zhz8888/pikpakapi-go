package enums

import (
	"testing"
)

func TestDownloadStatus_String(t *testing.T) {
	tests := []struct {
		status   DownloadStatus
		expected string
	}{
		{DownloadStatusNotDownloading, "not_downloading"},
		{DownloadStatusDownloading, "downloading"},
		{DownloadStatusDone, "done"},
		{DownloadStatusError, "error"},
		{DownloadStatusNotFound, "not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestParseDownloadStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected DownloadStatus
	}{
		{"not_downloading", DownloadStatusNotDownloading},
		{"downloading", DownloadStatusDownloading},
		{"done", DownloadStatusDone},
		{"error", DownloadStatusError},
		{"not_found", DownloadStatusNotFound},
		{"unknown", DownloadStatusNotFound},
		{"", DownloadStatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseDownloadStatus(tt.input); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestDownloadStatusConstants(t *testing.T) {
	if DownloadStatusNotDownloading != "not_downloading" {
		t.Error("DownloadStatusNotDownloading should be 'not_downloading'")
	}
	if DownloadStatusDownloading != "downloading" {
		t.Error("DownloadStatusDownloading should be 'downloading'")
	}
	if DownloadStatusDone != "done" {
		t.Error("DownloadStatusDone should be 'done'")
	}
	if DownloadStatusError != "error" {
		t.Error("DownloadStatusError should be 'error'")
	}
	if DownloadStatusNotFound != "not_found" {
		t.Error("DownloadStatusNotFound should be 'not_found'")
	}
}
