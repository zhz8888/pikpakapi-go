package exception

import (
	"errors"
	"fmt"
)

type ErrorCode int

const (
	ErrCodeSuccess ErrorCode = 1000 + iota
	ErrCodeInvalidUsernamePassword
	ErrCodeInvalidEncodedToken
	ErrCodeCaptchaTokenFailed
	ErrCodeUsernamePasswordRequired
	ErrCodeMaxRetriesReached
	ErrCodeUnknownError
	ErrCodeEmptyJSONData
	ErrCodeInvalidFileID
	ErrCodeInvalidFileName
	ErrCodeEmptyFileIDs
	ErrCodeInvalidURL
	ErrCodeInvalidAccessToken
	ErrCodeInvalidCredentials
	ErrCodeInvalidShareURL
	ErrCodeInvalidPassCode
	ErrCodeNetworkError
	ErrCodeServerError
	ErrCodeTimeout
	ErrCodeUnauthorized
	ErrCodeForbidden
	ErrCodeNotFound
	ErrCodeConflict
	ErrCodeInternalServerError
	ErrCodeServiceUnavailable
	ErrCodeInvalidParameter
	ErrCodeInvalidMediaFormat
	ErrCodeMarshalFailed
	ErrCodeCreateRequestFailed
	ErrCodeReadResponseFailed
	ErrCodeUnmarshalFailed
	ErrCodeMaxRetriesExceeded
	ErrCodeOpenFileFailed
	ErrCodeGetFileInfoFailed
	ErrCodeReadFileFailed
	ErrCodeCreateFormFileFailed
	ErrCodeWriteFileContentFailed
	ErrCodeReadChunkFailed
	ErrCodeDownloadFailed
	ErrCodeCreateDirectoryFailed
	ErrCodeCreateFileFailed
	ErrCodeWriteFileFailed
)

func (e ErrorCode) String() string {
	switch e {
	case ErrCodeSuccess:
		return "success"
	case ErrCodeInvalidUsernamePassword:
		return "invalid username or password"
	case ErrCodeInvalidEncodedToken:
		return "invalid encoded token"
	case ErrCodeCaptchaTokenFailed:
		return "captcha token get failed"
	case ErrCodeUsernamePasswordRequired:
		return "username and password are required"
	case ErrCodeMaxRetriesReached:
		return "max retries reached"
	case ErrCodeUnknownError:
		return "unknown error"
	case ErrCodeEmptyJSONData:
		return "empty JSON data"
	case ErrCodeInvalidFileID:
		return "invalid file id"
	case ErrCodeInvalidFileName:
		return "invalid file name"
	case ErrCodeEmptyFileIDs:
		return "file ids is empty"
	case ErrCodeInvalidURL:
		return "invalid url"
	case ErrCodeInvalidAccessToken:
		return "invalid access token"
	case ErrCodeInvalidCredentials:
		return "invalid credentials"
	case ErrCodeInvalidShareURL:
		return "invalid share url"
	case ErrCodeInvalidPassCode:
		return "invalid pass code"
	case ErrCodeNetworkError:
		return "network error"
	case ErrCodeServerError:
		return "server error"
	case ErrCodeTimeout:
		return "timeout"
	case ErrCodeUnauthorized:
		return "unauthorized"
	case ErrCodeForbidden:
		return "forbidden"
	case ErrCodeNotFound:
		return "not found"
	case ErrCodeConflict:
		return "conflict"
	case ErrCodeInternalServerError:
		return "internal server error"
	case ErrCodeServiceUnavailable:
		return "service unavailable"
	case ErrCodeInvalidParameter:
		return "invalid parameter"
	case ErrCodeInvalidMediaFormat:
		return "invalid media format"
	case ErrCodeMarshalFailed:
		return "marshal failed"
	case ErrCodeCreateRequestFailed:
		return "create request failed"
	case ErrCodeReadResponseFailed:
		return "read response failed"
	case ErrCodeUnmarshalFailed:
		return "unmarshal failed"
	case ErrCodeMaxRetriesExceeded:
		return "max retries exceeded"
	case ErrCodeOpenFileFailed:
		return "open file failed"
	case ErrCodeGetFileInfoFailed:
		return "get file info failed"
	case ErrCodeReadFileFailed:
		return "read file failed"
	case ErrCodeCreateFormFileFailed:
		return "create form file failed"
	case ErrCodeWriteFileContentFailed:
		return "write file content failed"
	case ErrCodeReadChunkFailed:
		return "read chunk failed"
	case ErrCodeDownloadFailed:
		return "download failed"
	case ErrCodeCreateDirectoryFailed:
		return "create directory failed"
	case ErrCodeCreateFileFailed:
		return "create file failed"
	case ErrCodeWriteFileFailed:
		return "write file failed"
	default:
		return "unknown error"
	}
}

func (e ErrorCode) Message() string {
	return e.String()
}

type PikpakException struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *PikpakException) Error() string {
	if e.Err != nil {
		if e.Message != "" {
			return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
		}
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Code.String(), e.Err)
	}
	if e.Message != "" {
		return fmt.Sprintf("[%d] %s", e.Code, e.Message)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Code.String())
}

func (e *PikpakException) Unwrap() error {
	return e.Err
}

func (e *PikpakException) Is(target error) bool {
	if t, ok := target.(*PikpakException); ok {
		return e.Code == t.Code
	}
	return false
}

func NewPikpakException(code ErrorCode) *PikpakException {
	return &PikpakException{Code: code, Message: code.String()}
}

func NewPikpakExceptionWithMessage(code ErrorCode, message string) *PikpakException {
	return &PikpakException{Code: code, Message: message}
}

func NewPikpakExceptionWithError(code ErrorCode, err error) *PikpakException {
	return &PikpakException{Code: code, Message: code.String(), Err: err}
}

func NewPikpakExceptionFull(code ErrorCode, message string, err error) *PikpakException {
	return &PikpakException{Code: code, Message: message, Err: err}
}

func IsPikpakException(err error) bool {
	var pe *PikpakException
	return errors.As(err, &pe)
}

func GetErrorCode(err error) ErrorCode {
	var pe *PikpakException
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ErrCodeUnknownError
}

var (
	ErrInvalidUsernamePassword  = NewPikpakException(ErrCodeInvalidUsernamePassword)
	ErrInvalidEncodedToken      = NewPikpakException(ErrCodeInvalidEncodedToken)
	ErrCaptchaTokenFailed       = NewPikpakException(ErrCodeCaptchaTokenFailed)
	ErrUsernamePasswordRequired = NewPikpakException(ErrCodeUsernamePasswordRequired)
	ErrMaxRetriesReached        = NewPikpakException(ErrCodeMaxRetriesReached)
	ErrUnknownError             = NewPikpakException(ErrCodeUnknownError)
	ErrEmptyJSONData            = NewPikpakException(ErrCodeEmptyJSONData)
	ErrInvalidFileID            = NewPikpakException(ErrCodeInvalidFileID)
	ErrInvalidFileName          = NewPikpakException(ErrCodeInvalidFileName)
	ErrEmptyFileIDs             = NewPikpakException(ErrCodeEmptyFileIDs)
	ErrInvalidURL               = NewPikpakException(ErrCodeInvalidURL)
	ErrInvalidAccessToken       = NewPikpakException(ErrCodeInvalidAccessToken)
	ErrInvalidCredentials       = NewPikpakException(ErrCodeInvalidCredentials)
	ErrInvalidShareURL          = NewPikpakException(ErrCodeInvalidShareURL)
	ErrInvalidPassCode          = NewPikpakException(ErrCodeInvalidPassCode)
	ErrNetworkError             = NewPikpakException(ErrCodeNetworkError)
	ErrServerError              = NewPikpakException(ErrCodeServerError)
	ErrTimeout                  = NewPikpakException(ErrCodeTimeout)
	ErrUnauthorized             = NewPikpakException(ErrCodeUnauthorized)
	ErrForbidden                = NewPikpakException(ErrCodeForbidden)
	ErrNotFound                 = NewPikpakException(ErrCodeNotFound)
	ErrConflict                 = NewPikpakException(ErrCodeConflict)
	ErrInternalServerError      = NewPikpakException(ErrCodeInternalServerError)
	ErrServiceUnavailable       = NewPikpakException(ErrCodeServiceUnavailable)
)
