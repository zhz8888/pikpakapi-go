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

func TestDownloadStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		status   DownloadStatus
		expected string
	}{
		{DownloadStatusDone, `"done"`},
		{DownloadStatusDownloading, `"downloading"`},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			data, err := tt.status.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}
			if string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %s, want %s", string(data), tt.expected)
			}
		})
	}
}

func TestDownloadStatus_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected DownloadStatus
	}{
		{`"done"`, DownloadStatusDone},
		{`"downloading"`, DownloadStatusDownloading},
		{`""`, DownloadStatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var status DownloadStatus
			err := status.UnmarshalJSON([]byte(tt.input))
			if err != nil {
				t.Errorf("UnmarshalJSON() error = %v", err)
				return
			}
			if status != tt.expected {
				t.Errorf("UnmarshalJSON() = %s, want %s", status, tt.expected)
			}
		})
	}
}

func TestDownloadPhase_String(t *testing.T) {
	tests := []struct {
		phase    DownloadPhase
		expected string
	}{
		{DownloadPhaseRunning, "PHASE_TYPE_RUNNING"},
		{DownloadPhaseError, "PHASE_TYPE_ERROR"},
		{DownloadPhaseComplete, "PHASE_TYPE_COMPLETE"},
		{DownloadPhasePending, "PHASE_TYPE_PENDING"},
		{DownloadPhasePaused, "PHASE_TYPE_PAUSED"},
		{DownloadPhaseWaiting, "PHASE_TYPE_WAITING"},
		{DownloadPhaseExtracting, "PHASE_TYPE_EXTRACTING"},
		{DownloadPhaseConverting, "PHASE_TYPE_CONVERTING"},
		{DownloadPhaseTe601, "PHASE_TYPE_TE601"},
		{DownloadPhaseChecking, "PHASE_TYPE_CHECKING"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.phase.String(); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestParseDownloadPhase(t *testing.T) {
	tests := []struct {
		input    string
		expected DownloadPhase
	}{
		{"PHASE_TYPE_RUNNING", DownloadPhaseRunning},
		{"PHASE_TYPE_ERROR", DownloadPhaseError},
		{"PHASE_TYPE_COMPLETE", DownloadPhaseComplete},
		{"PHASE_TYPE_PENDING", DownloadPhasePending},
		{"PHASE_TYPE_PAUSED", DownloadPhasePaused},
		{"PHASE_TYPE_WAITING", DownloadPhaseWaiting},
		{"PHASE_TYPE_EXTRACTING", DownloadPhaseExtracting},
		{"PHASE_TYPE_CONVERTING", DownloadPhaseConverting},
		{"PHASE_TYPE_TE601", DownloadPhaseTe601},
		{"PHASE_TYPE_CHECKING", DownloadPhaseChecking},
		{"unknown_phase", DownloadPhase("unknown_phase")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseDownloadPhase(tt.input); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestDownloadPhase_MarshalJSON(t *testing.T) {
	data, err := DownloadPhaseRunning.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
		return
	}
	expected := `"PHASE_TYPE_RUNNING"`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, want %s", string(data), expected)
	}
}

func TestDownloadPhase_UnmarshalJSON(t *testing.T) {
	var phase DownloadPhase
	err := phase.UnmarshalJSON([]byte(`"PHASE_TYPE_RUNNING"`))
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}
	if phase != DownloadPhaseRunning {
		t.Errorf("UnmarshalJSON() = %s, want %s", phase, DownloadPhaseRunning)
	}
}

func TestFileKind_IsFolder(t *testing.T) {
	tests := []struct {
		kind     FileKind
		expected bool
	}{
		{FileKindFolder, true},
		{FileKindFile, false},
		{FileKind("unknown"), false},
		{FileKind(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := tt.kind.IsFolder(); got != tt.expected {
				t.Errorf("IsFolder() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFileKind_String(t *testing.T) {
	tests := []struct {
		kind     FileKind
		expected string
	}{
		{FileKindFile, "drive#file"},
		{FileKindFolder, "drive#folder"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestParseFileKind(t *testing.T) {
	tests := []struct {
		input    string
		expected FileKind
	}{
		{"drive#file", FileKindFile},
		{"drive#folder", FileKindFolder},
		{"unknown_kind", FileKind("unknown_kind")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseFileKind(tt.input); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestFileKind_MarshalJSON(t *testing.T) {
	data, err := FileKindFile.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
		return
	}
	expected := `"drive#file"`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, want %s", string(data), expected)
	}
}

func TestFileKind_UnmarshalJSON(t *testing.T) {
	var kind FileKind
	err := kind.UnmarshalJSON([]byte(`"drive#file"`))
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}
	if kind != FileKindFile {
		t.Errorf("UnmarshalJSON() = %s, want %s", kind, FileKindFile)
	}
}

func TestEnums_DefaultValues(t *testing.T) {
	if FileKindFile != "drive#file" {
		t.Error("FileKindFile should be 'drive#file'")
	}
	if FileKindFolder != "drive#folder" {
		t.Error("FileKindFolder should be 'drive#folder'")
	}

	var defaultStatus DownloadStatus
	if defaultStatus != "" {
		t.Errorf("Default DownloadStatus should be empty string, got %s", defaultStatus)
	}

	var defaultPhase DownloadPhase
	if defaultPhase != "" {
		t.Errorf("Default DownloadPhase should be empty string, got %s", defaultPhase)
	}

	var defaultKind FileKind
	if defaultKind != "" {
		t.Errorf("Default FileKind should be empty string, got %s", defaultKind)
	}
}

func TestDownloadStatus_ParseEmpty(t *testing.T) {
	status := ParseDownloadStatus("")
	if status != DownloadStatusNotFound {
		t.Errorf("ParseDownloadStatus(\"\") should return DownloadStatusNotFound, got %s", status)
	}
}

func TestDownloadStatus_Compare(t *testing.T) {
	status1 := DownloadStatusDone
	status2 := DownloadStatusDone
	status3 := DownloadStatusError

	if status1 != status2 {
		t.Error("Same DownloadStatus values should be equal")
	}

	if status1 == status3 {
		t.Error("Different DownloadStatus values should not be equal")
	}
}

func TestFileKind_Compare(t *testing.T) {
	kind1 := FileKindFile
	kind2 := FileKindFile
	kind3 := FileKindFolder

	if kind1 != kind2 {
		t.Error("Same FileKind values should be equal")
	}

	if kind1 == kind3 {
		t.Error("Different FileKind values should not be equal")
	}
}
