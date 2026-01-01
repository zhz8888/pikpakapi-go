package enums

import "strings"

type DownloadStatus string

const (
	DownloadStatusNotDownloading DownloadStatus = "not_downloading"
	DownloadStatusDownloading    DownloadStatus = "downloading"
	DownloadStatusDone           DownloadStatus = "done"
	DownloadStatusError          DownloadStatus = "error"
	DownloadStatusNotFound       DownloadStatus = "not_found"
)

func (s DownloadStatus) String() string {
	return string(s)
}

func ParseDownloadStatus(status string) DownloadStatus {
	switch status {
	case "not_downloading":
		return DownloadStatusNotDownloading
	case "downloading":
		return DownloadStatusDownloading
	case "done":
		return DownloadStatusDone
	case "error":
		return DownloadStatusError
	case "not_found":
		return DownloadStatusNotFound
	default:
		return DownloadStatusNotFound
	}
}

type DownloadPhase string

const (
	DownloadPhaseRunning   DownloadPhase = "PHASE_TYPE_RUNNING"
	DownloadPhaseError     DownloadPhase = "PHASE_TYPE_ERROR"
	DownloadPhaseComplete  DownloadPhase = "PHASE_TYPE_COMPLETE"
	DownloadPhasePending   DownloadPhase = "PHASE_TYPE_PENDING"
	DownloadPhasePaused    DownloadPhase = "PHASE_TYPE_PAUSED"
	DownloadPhaseWaiting   DownloadPhase = "PHASE_TYPE_WAITING"
	DownloadPhaseExtracting DownloadPhase = "PHASE_TYPE_EXTRACTING"
	DownloadPhaseConverting DownloadPhase = "PHASE_TYPE_CONVERTING"
	DownloadPhaseTe601     DownloadPhase = "PHASE_TYPE_TE601"
	DownloadPhaseChecking  DownloadPhase = "PHASE_TYPE_CHECKING"
)

func (p DownloadPhase) String() string {
	return string(p)
}

func ParseDownloadPhase(phase string) DownloadPhase {
	switch phase {
	case "PHASE_TYPE_RUNNING":
		return DownloadPhaseRunning
	case "PHASE_TYPE_ERROR":
		return DownloadPhaseError
	case "PHASE_TYPE_COMPLETE":
		return DownloadPhaseComplete
	case "PHASE_TYPE_PENDING":
		return DownloadPhasePending
	case "PHASE_TYPE_PAUSED":
		return DownloadPhasePaused
	case "PHASE_TYPE_WAITING":
		return DownloadPhaseWaiting
	case "PHASE_TYPE_EXTRACTING":
		return DownloadPhaseExtracting
	case "PHASE_TYPE_CONVERTING":
		return DownloadPhaseConverting
	case "PHASE_TYPE_TE601":
		return DownloadPhaseTe601
	case "PHASE_TYPE_CHECKING":
		return DownloadPhaseChecking
	default:
		return DownloadPhase(phase)
	}
}

type FileKind string

const (
	FileKindFile   FileKind = "drive#file"
	FileKindFolder FileKind = "drive#folder"
)

func (k FileKind) IsFolder() bool {
	return k == FileKindFolder
}

func (k FileKind) String() string {
	return string(k)
}

func ParseFileKind(kind string) FileKind {
	switch kind {
	case "drive#file":
		return FileKindFile
	case "drive#folder":
		return FileKindFolder
	default:
		return FileKind(kind)
	}
}

func (s *DownloadStatus) UnmarshalJSON(data []byte) error {
	unquoted := strings.Trim(string(data), `"`)
	*s = ParseDownloadStatus(unquoted)
	return nil
}

func (s DownloadStatus) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(s) + `"`), nil
}

func (p *DownloadPhase) UnmarshalJSON(data []byte) error {
	unquoted := strings.Trim(string(data), `"`)
	*p = ParseDownloadPhase(unquoted)
	return nil
}

func (p DownloadPhase) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(p) + `"`), nil
}

func (k *FileKind) UnmarshalJSON(data []byte) error {
	unquoted := strings.Trim(string(data), `"`)
	*k = ParseFileKind(unquoted)
	return nil
}

func (k FileKind) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(k) + `"`), nil
}
