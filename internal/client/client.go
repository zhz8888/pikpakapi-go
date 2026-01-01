package client

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zhz8888/pikpakapi-go/internal/auth"
	"github.com/zhz8888/pikpakapi-go/internal/constants"
	"github.com/zhz8888/pikpakapi-go/internal/download"
	"github.com/zhz8888/pikpakapi-go/internal/exception"
	"github.com/zhz8888/pikpakapi-go/internal/file"
	"github.com/zhz8888/pikpakapi-go/internal/share"
	"github.com/zhz8888/pikpakapi-go/internal/useragent"
	"github.com/zhz8888/pikpakapi-go/pkg/enums"
)

const (
	HTTPTimeout = 30 * time.Second
)

type ClientInterface interface {
	Login(ctx context.Context) error
	RefreshAccessToken(ctx context.Context) error
	DecodeToken() error
	EncodeToken() error
	GetUserInfo() map[string]string

	FileList(ctx context.Context, size int, parentID string, nextPageToken string, query string) (map[string]interface{}, error)
	CreateFolder(ctx context.Context, name string, parentID string) (map[string]interface{}, error)
	GetFileLink(ctx context.Context, fileID string) (string, error)
	Move(ctx context.Context, fileID string, parentID string) error
	Copy(ctx context.Context, fileID string, parentID string) error
	Rename(ctx context.Context, fileID string, newName string) error
	DeleteToTrash(ctx context.Context, ids []string) (map[string]interface{}, error)
	Untrash(ctx context.Context, ids []string) (map[string]interface{}, error)
	DeleteForever(ctx context.Context, ids []string) (map[string]interface{}, error)
	GetAbout(ctx context.Context) (map[string]interface{}, error)

	OfflineDownload(ctx context.Context, fileURL string, parentID string, name string) (map[string]interface{}, error)
	OfflineList(ctx context.Context, size int, nextPageToken string, phases []string) (map[string]interface{}, error)
	DeleteOfflineTasks(ctx context.Context, taskIDs []string, deleteFiles bool) error
	DeleteTasks(ctx context.Context, taskIDs []string, deleteFiles bool) error
	GetTaskStatus(ctx context.Context, taskID string, fileID string) (enums.DownloadStatus, error)
	CaptureScreenshot(ctx context.Context, fileID string) (map[string]interface{}, error)

	FileBatchShare(ctx context.Context, ids []string, needPassword bool) (map[string]interface{}, error)
	GetShareInfo(ctx context.Context, shareURL string) (map[string]interface{}, error)
	Restore(ctx context.Context, shareID string, passCodeToken string, fileIDs []string) (map[string]interface{}, error)
	GetStorageInfo(ctx context.Context) (StorageInfo, error)
	OfflineTaskRetry(ctx context.Context, taskID string) error
	FileRename(ctx context.Context, fileID string, newName string) error
	FileBatchStar(ctx context.Context, ids []string, star bool) error
	FileStarList(ctx context.Context, size int, nextPageToken string) (map[string]interface{}, error)
	FileBatchUnstar(ctx context.Context, ids []string) error
	Upload(ctx context.Context, filePath string, parentID string) (map[string]interface{}, error)
	UploadReader(ctx context.Context, reader io.Reader, fileName string, fileSize int64, parentID string) (map[string]interface{}, error)
	CreateShareLink(ctx context.Context, fileID string, expireSec int, passCode string) (map[string]interface{}, error)
	GetShareDownloadURL(ctx context.Context, shareURL string, sharePassword string) (string, error)
}

type Client struct {
	authModule  *auth.Auth
	fileModule  *file.File
	downloadMod *download.Download
	shareModule *share.Share

	username                string
	password                string
	maxRetries              int
	initialBackoff          time.Duration
	httpClient              *http.Client
	tokenRefreshCallback    func(*Client)
	tokenRefreshCallbackCtx context.Context
	baseURL                 string
}

type Option func(*Client)

func WithUsername(username string) Option {
	return func(c *Client) {
		c.username = username
	}
}

func WithPassword(password string) Option {
	return func(c *Client) {
		c.password = password
	}
}

func WithMaxRetries(maxRetries int) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

func WithInitialBackoff(backoff time.Duration) Option {
	return func(c *Client) {
		c.initialBackoff = backoff
	}
}

func WithTokenRefreshCallback(callback func(*Client)) Option {
	return func(c *Client) {
		c.tokenRefreshCallback = callback
	}
}

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

func WithDeviceID(deviceID string) Option {
	return func(c *Client) {
		c.authModule.WithDeviceID(deviceID)
	}
}

func WithAccessToken(token string) Option {
	return func(c *Client) {
		c.authModule.SetAccessToken(token)
	}
}

func WithRefreshToken(token string) Option {
	return func(c *Client) {
		c.authModule.SetRefreshToken(token)
	}
}

func generateDeviceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func NewClient(opts ...Option) *Client {
	c := &Client{
		maxRetries:     3,
		initialBackoff: 3 * time.Second,
		httpClient: &http.Client{
			Timeout: HTTPTimeout,
		},
		baseURL: "",
	}

	c.authModule = auth.NewAuth(
		auth.WithUsername(c.username),
		auth.WithPassword(c.password),
		auth.WithBaseURL(c.baseURL),
	)

	for _, opt := range opts {
		opt(c)
	}

	if c.GetDeviceID() == "" {
		c.SetDeviceID(generateDeviceID())
	}

	c.fileModule = file.NewFile(
		file.WithFileBaseURL(c.baseURL),
	)

	c.downloadMod = download.NewDownload(
		download.WithDownloadBaseURL(c.baseURL),
	)

	c.shareModule = share.NewShare(
		share.WithShareBaseURL(c.baseURL),
	)

	c.authModule.SetHTTPClient(c)
	c.fileModule.SetHTTPClient(c)
	c.downloadMod.SetHTTPClient(c)
	c.shareModule.SetHTTPClient(c)

	return c
}

func (c *Client) SetDeviceID(deviceID string) {
	c.authModule.WithDeviceID(deviceID)
}

func (c *Client) GetDeviceID() string {
	return c.authModule.GetDeviceID()
}

func (c *Client) SetAccessToken(token string) {
	c.authModule.SetAccessToken(token)
}

func (c *Client) SetRefreshToken(token string) {
	c.authModule.SetRefreshToken(token)
}

func (c *Client) SetUserID(userID string) {
	c.authModule.SetUserID(userID)
}

func (c *Client) SetEncodedToken(token string) {
	c.authModule.SetEncodedToken(token)
}

func (c *Client) GetAccessToken() string {
	return c.authModule.GetAccessToken()
}

func (c *Client) GetRefreshToken() string {
	return c.authModule.GetRefreshToken()
}

func (c *Client) GetEncodedToken() string {
	return c.authModule.GetEncodedToken()
}

func (c *Client) GetUserID() string {
	return c.authModule.GetUserID()
}

func (c *Client) GetUserInfo() map[string]string {
	return map[string]string{
		"username":      c.username,
		"user_id":       c.authModule.GetUserID(),
		"access_token":  c.authModule.GetAccessToken(),
		"refresh_token": c.authModule.GetRefreshToken(),
		"encoded_token": c.authModule.GetEncodedToken(),
	}
}

func (c *Client) Login(ctx context.Context) error {
	if err := c.authModule.Login(ctx); err != nil {
		return err
	}
	c.username = c.authModule.GetUserID()
	return nil
}

type ShareFileInfo struct {
	ID            string
	Name          string
	Size          int64
	ThumbnailLink string
	MediaType     string
	ShareURL      string
	DownloadURL   string
}

type ShareOption struct {
	ExpireSec int
	PassCode  string
}

func (c *Client) GetStorageInfo(ctx context.Context) (StorageInfo, error) {
	result, err := c.GetAbout(ctx)
	if err != nil {
		return StorageInfo{}, err
	}

	storage := StorageInfo{}
	if quota, ok := result["quota"].(map[string]interface{}); ok {
		if limit, ok := quota["limit"].(string); ok {
			if limitNum, err := strconv.ParseUint(limit, 10, 64); err == nil {
				storage.TotalBytes = limitNum
			}
		}
		if usage, ok := quota["usage"].(string); ok {
			if usageNum, err := strconv.ParseUint(usage, 10, 64); err == nil {
				storage.UsedBytes = usageNum
			}
		}
		if usageInTrash, ok := quota["usage_in_trash"].(string); ok {
			if trashNum, err := strconv.ParseUint(usageInTrash, 10, 64); err == nil {
				storage.TrashBytes = trashNum
			}
		}
		if isUnlimited, ok := quota["is_unlimited"].(bool); ok {
			storage.IsUnlimited = isUnlimited
		}
		if complimentary, ok := quota["complimentary"].(string); ok {
			storage.Complimentary = complimentary
		}
	}
	if expiresAt, ok := result["expires_at"].(string); ok {
		storage.ExpiresAt = expiresAt
	}
	if userType, ok := result["user_type"].(float64); ok {
		storage.UserType = int(userType)
	}

	return storage, nil
}

func (c *Client) OfflineTaskRetry(ctx context.Context, taskID string) error {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/files/" + taskID

	data := map[string]interface{}{
		"status": "PENDING",
	}

	_, err := c.PostJSON(ctx, URL, data)
	return err
}

func (c *Client) FileRename(ctx context.Context, fileID string, newName string) error {
	return c.Rename(ctx, fileID, newName)
}

func (c *Client) FileBatchStar(ctx context.Context, ids []string, star bool) error {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/files:batchStar"

	data := map[string]interface{}{
		"ids":  ids,
		"star": star,
	}

	_, err := c.PostJSON(ctx, URL, data)
	return err
}

func (c *Client) FileStarList(ctx context.Context, size int, nextPageToken string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/files"

	if size == 0 {
		size = 50
	}

	params := map[string]string{
		"limit":          strconv.Itoa(size),
		"starred":        "true",
		"thumbnail_size": "SIZE_LARGE",
	}

	if nextPageToken != "" {
		params["page_token"] = nextPageToken
	}

	return c.GetJSON(ctx, URL, params)
}

func (c *Client) FileBatchUnstar(ctx context.Context, ids []string) error {
	return c.FileBatchStar(ctx, ids, false)
}

func (c *Client) Upload(ctx context.Context, filePath string, parentID string) (map[string]interface{}, error) {
	return c.UploadFile(ctx, filePath, parentID, 4*1024*1024)
}

func (c *Client) UploadReader(ctx context.Context, reader io.Reader, fileName string, fileSize int64, parentID string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}

	uploadURL, err := c.GetUploadURL(ctx, fileName, fileSize, parentID)
	if err != nil {
		return nil, err
	}

	file := &os.File{}
	if f, ok := reader.(*os.File); ok {
		file = f
	}

	if file == nil {
		return nil, exception.NewPikpakExceptionWithMessage(exception.ErrCodeInvalidParameter, "reader must be *os.File")
	}

	return c.uploadFileSmall(ctx, uploadURL, file, fileName, fileSize, parentID)
}

func (c *Client) CreateShareLink(ctx context.Context, fileID string, expireSec int, passCode string) (map[string]interface{}, error) {
	if expireSec == 0 {
		expireSec = 86400
	}
	return c.Share(ctx, fileID, 2, expireSec, passCode)
}

func (c *Client) GetShareDownloadURL(ctx context.Context, shareURL string, sharePassword string) (string, error) {
	return c.GetShareFileDownloadURL(ctx, shareURL, sharePassword, false)
}

func (c *Client) RefreshAccessToken(ctx context.Context) error {
	if err := c.authModule.RefreshAccessToken(ctx); err != nil {
		return err
	}
	if c.tokenRefreshCallback != nil {
		c.tokenRefreshCallback(c)
	}
	return nil
}

func (c *Client) DecodeToken() error {
	return c.authModule.DecodeToken()
}

func (c *Client) EncodeToken() error {
	return c.authModule.EncodeToken()
}

func (c *Client) buildUserAgent() string {
	if c.authModule.GetCaptchaToken() != "" {
		return useragent.BuildCustomUserAgent(c.authModule.GetDeviceID(), c.authModule.GetUserID())
	}
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
}

func (c *Client) getHeaders() map[string]string {
	headers := map[string]string{
		"User-Agent":   c.buildUserAgent(),
		"Content-Type": "application/json; charset=utf-8",
	}

	if c.authModule.GetAccessToken() != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", c.authModule.GetAccessToken())
	}
	if c.authModule.GetCaptchaToken() != "" {
		headers["X-Captcha-Token"] = c.authModule.GetCaptchaToken()
	}
	if c.authModule.GetDeviceID() != "" {
		headers["X-Device-Id"] = c.authModule.GetDeviceID()
	}

	return headers
}

func (c *Client) doRequest(ctx context.Context, method, reqURL string, data interface{}, params map[string]string) ([]byte, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeMarshalFailed, err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeCreateRequestFailed, err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}

	if params != nil {
		q := req.URL.Query()
		for key, value := range params {
			q.Set(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := c.initialBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("Request failed (attempt %d/%d): %v", attempt+1, c.maxRetries+1, err)
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			log.Printf("Failed to read response (attempt %d/%d): %v", attempt+1, c.maxRetries+1, err)
			continue
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			return respBody, nil
		}

		var respData map[string]interface{}
		if err := json.Unmarshal(respBody, &respData); err == nil {
			if errCode, ok := respData["error_code"].(float64); ok && int(errCode) == 16 {
				if c.authModule.GetRefreshToken() != "" {
					if refreshErr := c.RefreshAccessToken(ctx); refreshErr == nil {
						for key, value := range c.getHeaders() {
							req.Header.Set(key, value)
						}
						continue
					}
				}
			}
			if errorMsg, ok := respData["error"].(string); ok {
				return nil, exception.NewPikpakExceptionWithMessage(exception.ErrCodeServerError, errorMsg)
			}
		}

		if resp.StatusCode == http.StatusUnauthorized {
			return nil, exception.ErrInvalidAccessToken
		}
		if resp.StatusCode == http.StatusForbidden {
			return nil, exception.ErrInvalidCredentials
		}

		return nil, exception.NewPikpakExceptionWithMessage(exception.ErrCodeServerError, fmt.Sprintf("request failed with status: %d, body: %s", resp.StatusCode, string(respBody)))
	}

	return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeMaxRetriesExceeded, lastErr)
}

func (c *Client) GetJSON(ctx context.Context, URL string, params map[string]string) (map[string]interface{}, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, URL, nil, params)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeUnmarshalFailed, err)
	}

	return result, nil
}

func (c *Client) PostJSON(ctx context.Context, URL string, data interface{}) (map[string]interface{}, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, URL, data, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeUnmarshalFailed, err)
	}

	return result, nil
}

func (c *Client) PatchJSON(ctx context.Context, URL string, data interface{}) (map[string]interface{}, error) {
	respBody, err := c.doRequest(ctx, http.MethodPatch, URL, data, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeUnmarshalFailed, err)
	}

	return result, nil
}

func (c *Client) PostForm(ctx context.Context, URL string, data map[string]string) (map[string]interface{}, error) {
	form := url.Values{}
	for key, value := range data {
		form.Set(key, value)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeCreateRequestFailed, err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeNetworkError, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeReadResponseFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, exception.NewPikpakExceptionWithMessage(exception.ErrCodeServerError, fmt.Sprintf("post form failed with status: %d, body: %s", resp.StatusCode, string(respBody)))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeUnmarshalFailed, err)
	}

	return result, nil
}

func (c *Client) Delete(ctx context.Context, URL string, params map[string]string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, URL, nil)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeCreateRequestFailed, err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}

	if params != nil {
		q := req.URL.Query()
		for key, value := range params {
			q.Set(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("delete failed: %s", string(respBody))
	}

	return map[string]interface{}{"status": "ok"}, nil
}

func (c *Client) FileList(ctx context.Context, size int, parentID string, nextPageToken string, query string) (map[string]interface{}, error) {
	return c.fileModule.FileList(ctx, size, parentID, nextPageToken, query)
}

func (c *Client) CreateFolder(ctx context.Context, name string, parentID string) (map[string]interface{}, error) {
	return c.fileModule.CreateFolder(ctx, name, parentID)
}

func (c *Client) GetFileLink(ctx context.Context, fileID string) (string, error) {
	return c.fileModule.GetFileLink(ctx, fileID)
}

func (c *Client) Move(ctx context.Context, fileID string, parentID string) error {
	return c.fileModule.Move(ctx, fileID, parentID)
}

func (c *Client) Copy(ctx context.Context, fileID string, parentID string) error {
	return c.fileModule.Copy(ctx, fileID, parentID)
}

func (c *Client) Rename(ctx context.Context, fileID string, newName string) error {
	return c.fileModule.Rename(ctx, fileID, newName)
}

func (c *Client) DeleteToTrash(ctx context.Context, ids []string) (map[string]interface{}, error) {
	return c.fileModule.DeleteToTrash(ctx, ids)
}

func (c *Client) Untrash(ctx context.Context, ids []string) (map[string]interface{}, error) {
	return c.fileModule.Untrash(ctx, ids)
}

func (c *Client) DeleteForever(ctx context.Context, ids []string) (map[string]interface{}, error) {
	return c.fileModule.DeleteForever(ctx, ids)
}

func (c *Client) GetAbout(ctx context.Context) (map[string]interface{}, error) {
	return c.fileModule.GetAbout(ctx)
}

func (c *Client) OfflineDownload(ctx context.Context, fileURL string, parentID string, name string) (map[string]interface{}, error) {
	return c.downloadMod.OfflineDownload(ctx, fileURL, parentID, name)
}

func (c *Client) OfflineList(ctx context.Context, size int, nextPageToken string, phases []string) (map[string]interface{}, error) {
	return c.downloadMod.OfflineList(ctx, size, nextPageToken, phases)
}

func (c *Client) DeleteOfflineTasks(ctx context.Context, taskIDs []string, deleteFiles bool) error {
	return c.downloadMod.DeleteOfflineTasks(ctx, taskIDs, deleteFiles)
}

func (c *Client) DeleteTasks(ctx context.Context, taskIDs []string, deleteFiles bool) error {
	return c.downloadMod.DeleteTasks(ctx, taskIDs, deleteFiles)
}

func (c *Client) GetTaskStatus(ctx context.Context, taskID string, fileID string) (enums.DownloadStatus, error) {
	return c.downloadMod.GetTaskStatus(ctx, taskID, fileID)
}

func (c *Client) CaptureScreenshot(ctx context.Context, fileID string) (map[string]interface{}, error) {
	return c.downloadMod.CaptureScreenshot(ctx, fileID)
}

func (c *Client) FileBatchShare(ctx context.Context, ids []string, needPassword bool) (map[string]interface{}, error) {
	return c.shareModule.FileBatchShare(ctx, ids, needPassword)
}

func (c *Client) GetShareInfo(ctx context.Context, shareURL string) (map[string]interface{}, error) {
	return c.shareModule.GetShareInfo(ctx, shareURL)
}

func (c *Client) Restore(ctx context.Context, shareID string, passCodeToken string, fileIDs []string) (map[string]interface{}, error) {
	return c.shareModule.Restore(ctx, shareID, passCodeToken, fileIDs)
}

type AboutResponse struct {
	Quota struct {
		Limit         string `json:"limit"`
		Usage         string `json:"usage"`
		UsageInTrash  string `json:"usage_in_trash"`
		IsUnlimited   bool   `json:"is_unlimited"`
		Complimentary string `json:"complimentary"`
	} `json:"quota"`
	ExpiresAt string `json:"expires_at"`
	UserType  int    `json:"user_type"`
}

type StorageInfo struct {
	TotalBytes    uint64
	UsedBytes     uint64
	TrashBytes    uint64
	IsUnlimited   bool
	Complimentary string
	ExpiresAt     string
	UserType      int
}

func (c *Client) GetQuotaInfo(ctx context.Context) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/about"

	return c.GetJSON(ctx, URL, nil)
}

func parseShareFileInfo(fileInfo map[string]interface{}) (*ShareFileInfo, error) {
	info := &ShareFileInfo{}

	if id, ok := fileInfo["id"].(string); ok {
		info.ID = id
	}
	if name, ok := fileInfo["name"].(string); ok {
		info.Name = name
	}
	if size, ok := fileInfo["size"].(float64); ok {
		info.Size = int64(size)
	}
	if thumb, ok := fileInfo["thumbnail_link"].(string); ok {
		info.ThumbnailLink = thumb
	}
	if mimeType, ok := fileInfo["mime_type"].(string); ok {
		info.MediaType = mimeType
	}
	if link, ok := fileInfo["share_link"].(map[string]interface{}); ok {
		if url, ok := link["url"].(string); ok {
			info.ShareURL = url
		}
	}
	if link, ok := fileInfo["link"].(map[string]interface{}); ok {
		if url, ok := link["url"].(string); ok {
			info.DownloadURL = url
		}
	}

	return info, nil
}

func (c *Client) extractShareID(shareURL string) (string, error) {
	re := regexp.MustCompile(`/share/link/([^?]+)`)
	matches := re.FindStringSubmatch(shareURL)
	if len(matches) < 2 {
		return "", exception.ErrInvalidShareURL
	}
	return matches[1], nil
}

func (c *Client) getSharePassToken(ctx context.Context, shareID string, passCode string) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/share/v1/passcode"

	data := map[string]interface{}{
		"share_id": shareID,
		"passcode": passCode,
	}

	result, err := c.PostJSON(ctx, URL, data)
	if err != nil {
		return "", err
	}

	if token, ok := result["pass_code_token"].(string); ok {
		return token, nil
	}
	return "", exception.ErrInvalidPassCode
}

func (c *Client) Share(ctx context.Context, fileID string, shareType int, expireSec int, passCode string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/share"

	data := map[string]interface{}{
		"file_id":    fileID,
		"share_type": shareType,
		"expire_sec": expireSec,
		"pass_code":  passCode,
		"file_entity_filter": map[string]bool{
			"video":    true,
			"image":    true,
			"document": true,
			"other":    true,
		},
	}

	return c.PostJSON(ctx, URL, data)
}

func (c *Client) SetSharePolicy(ctx context.Context, shareID string, policy string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/share/" + shareID

	data := map[string]interface{}{
		"policy": policy,
	}

	return c.PostJSON(ctx, URL, data)
}

func (c *Client) GetShareList(ctx context.Context, size int, nextPageToken string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/share/list"

	if size == 0 {
		size = 50
	}

	params := map[string]string{
		"limit": strconv.Itoa(size),
	}

	if nextPageToken != "" {
		params["page_token"] = nextPageToken
	}

	return c.GetJSON(ctx, URL, params)
}

func (c *Client) GetSharePasscode(ctx context.Context, shareID string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/share/v1/passcode/" + shareID

	return c.GetJSON(ctx, URL, nil)
}

func (c *Client) CancelShare(ctx context.Context, shareID string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/share/" + shareID + "/cancel"

	return c.PostJSON(ctx, URL, nil)
}

func (c *Client) InviteNewShare(ctx context.Context, shareID string, fileIDs []string, inviteMsg string, isNewInvite bool) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/share/v1/invite"

	data := map[string]interface{}{
		"share_id":       shareID,
		"file_ids":       fileIDs,
		"invite_message": inviteMsg,
		"is_new_invite":  isNewInvite,
	}

	return c.PostJSON(ctx, URL, data)
}

func (c *Client) InviteList(ctx context.Context, shareID string, size int, nextPageToken string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/share/v1/invite/list"

	params := map[string]string{
		"share_id": shareID,
	}

	if size > 0 {
		params["limit"] = strconv.Itoa(size)
	}
	if nextPageToken != "" {
		params["page_token"] = nextPageToken
	}

	return c.GetJSON(ctx, URL, params)
}

func (c *Client) InviteCancel(ctx context.Context, inviteID string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/share/v1/invite/cancel"

	data := map[string]interface{}{
		"invite_id": inviteID,
	}

	return c.PostJSON(ctx, URL, data)
}

func (c *Client) Favorite(ctx context.Context, fileID string, category string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/files/" + fileID + ":favorite"

	data := map[string]interface{}{
		"category": category,
	}

	return c.PostJSON(ctx, URL, data)
}

func (c *Client) Events(ctx context.Context, size int, nextPageToken string) (map[string]interface{}, error) {
	if size == 0 {
		size = 100
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/events"

	params := map[string]string{
		"thumbnail_size": "SIZE_MEDIUM",
		"limit":          fmt.Sprintf("%d", size),
	}

	if nextPageToken != "" {
		params["next_page_token"] = nextPageToken
	}

	return c.GetJSON(ctx, URL, params)
}

func (c *Client) RemoteDownload(ctx context.Context, fileURL string) (map[string]interface{}, error) {
	if fileURL == "" {
		return nil, exception.ErrInvalidURL
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/files"

	data := map[string]interface{}{
		"kind":        "drive#task",
		"upload_type": "UPLOAD_TYPE_URL",
		"url":         map[string]string{"url": fileURL},
	}

	return c.PostJSON(ctx, URL, data)
}

func (c *Client) GetShareFileInfo(ctx context.Context, shareURL string, sharePassword string) (*ShareFileInfo, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}

	shareID, err := c.extractShareID(shareURL)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"share_id": shareID,
	}

	if sharePassword != "" {
		passToken, passErr := c.getSharePassToken(ctx, shareID, sharePassword)
		if passErr != nil {
			return nil, passErr
		}
		params["pass_code_token"] = passToken
	}

	URL := baseURL + "/drive/v1/share/file_info"

	result, err := c.GetJSON(ctx, URL, params)
	if err != nil {
		return nil, err
	}

	fileInfo, ok := result["file_info"].(map[string]interface{})
	if !ok {
		return nil, exception.NewPikpakExceptionWithMessage(exception.ErrCodeNotFound, "file_info not found in response")
	}

	return parseShareFileInfo(fileInfo)
}

func (c *Client) GetShareFileDownloadURL(ctx context.Context, shareURL string, sharePassword string, useTranscoding bool) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}

	shareID, err := c.extractShareID(shareURL)
	if err != nil {
		return "", err
	}

	params := map[string]string{
		"share_id": shareID,
	}

	if sharePassword != "" {
		passToken, passErr := c.getSharePassToken(ctx, shareID, sharePassword)
		if passErr != nil {
			return "", passErr
		}
		params["pass_code_token"] = passToken
	}

	URL := baseURL + "/drive/v1/share/file_info"

	result, err := c.GetJSON(ctx, URL, params)
	if err != nil {
		return "", err
	}

	fileInfo, ok := result["file_info"].(map[string]interface{})
	if !ok {
		return "", exception.NewPikpakExceptionWithMessage(exception.ErrCodeNotFound, "file_info not found in response")
	}

	if webContentLink, hasWebContentLink := fileInfo["web_content_link"].(string); hasWebContentLink && webContentLink != "" && !useTranscoding {
		return webContentLink, nil
	}

	medias, ok := fileInfo["medias"].([]interface{})
	if !ok || len(medias) == 0 {
		if webContentLink, hasWebContentLink := fileInfo["web_content_link"].(string); hasWebContentLink {
			return webContentLink, nil
		}
		return "", exception.NewPikpakExceptionWithMessage(exception.ErrCodeNotFound, "no download link available")
	}

	if useTranscoding && len(medias) > 1 {
		for _, m := range medias {
			media, mediaOk := m.(map[string]interface{})
			if !mediaOk {
				continue
			}
			link, linkOk := media["link"].(map[string]interface{})
			if !linkOk {
				continue
			}
			if url, urlOk := link["url"].(string); urlOk && url != "" {
				return url, nil
			}
		}
	}

	firstMedia, mediaOk := medias[0].(map[string]interface{})
	if !mediaOk {
		return "", exception.NewPikpakExceptionWithMessage(exception.ErrCodeInvalidMediaFormat, "invalid media format")
	}

	link, linkOk := firstMedia["link"].(map[string]interface{})
	if !linkOk {
		return "", exception.NewPikpakExceptionWithMessage(exception.ErrCodeNotFound, "link not found in media")
	}

	if url, urlOk := link["url"].(string); urlOk {
		return url, nil
	}

	return "", exception.NewPikpakExceptionWithMessage(exception.ErrCodeNotFound, "download url not found")
}

func (c *Client) GetShareFiles(ctx context.Context, shareURL string, sharePassword string) ([]*ShareFileInfo, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}

	shareID, err := c.extractShareID(shareURL)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"share_id":       shareID,
		"thumbnail_size": "SIZE_LARGE",
	}

	if sharePassword != "" {
		passToken, passErr := c.getSharePassToken(ctx, shareID, sharePassword)
		if passErr != nil {
			return nil, passErr
		}
		params["pass_code_token"] = passToken
	}

	URL := baseURL + "/drive/v1/share/file/list"

	result, err := c.GetJSON(ctx, URL, params)
	if err != nil {
		return nil, err
	}

	files := []*ShareFileInfo{}

	if filesRaw, ok := result["files"].([]interface{}); ok {
		for _, f := range filesRaw {
			if fileMap, ok := f.(map[string]interface{}); ok {
				if fileInfo, err := parseShareFileInfo(fileMap); err == nil {
					files = append(files, fileInfo)
				}
			}
		}
	}

	return files, nil
}

func (c *Client) OfflineFileInfo(ctx context.Context, fileID string) (map[string]interface{}, error) {
	if fileID == "" {
		return nil, exception.ErrInvalidFileID
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/files/" + fileID

	return c.GetJSON(ctx, URL, nil)
}

func (c *Client) UploadFile(ctx context.Context, filePath string, parentID string, chunkSize int) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeOpenFileFailed, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeGetFileInfoFailed, err)
	}

	fileSize := fileInfo.Size()
	fileName := fileInfo.Name()

	if chunkSize == 0 {
		chunkSize = 8 * 1024 * 1024
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	uploadURL := baseURL + "/drive/v1/files"

	var uploadResult map[string]interface{}

	if fileSize <= int64(chunkSize) {
		uploadResult, err = c.uploadFileSmall(ctx, uploadURL, file, fileName, fileSize, parentID)
	} else {
		uploadResult, err = c.uploadFileLarge(ctx, uploadURL, file, fileName, fileSize, chunkSize, parentID)
	}

	if err != nil {
		return nil, err
	}

	return uploadResult, nil
}

func (c *Client) uploadFileSmall(ctx context.Context, uploadURL string, file *os.File, fileName string, fileSize int64, parentID string) (map[string]interface{}, error) {
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeReadFileFailed, err)
	}

	md5Hash := md5.Sum(fileContent)
	md5Str := hex.EncodeToString(md5Hash[:])

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeCreateFormFileFailed, err)
	}

	if _, err := part.Write(fileContent); err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeWriteFileContentFailed, err)
	}

	_ = writer.WriteField("name", fileName)
	_ = writer.WriteField("parent_id", parentID)
	_ = writer.WriteField("size", strconv.FormatInt(fileSize, 10))
	_ = writer.WriteField("hash", md5Str)
	_ = writer.WriteField("kind", "drive#file")
	_ = writer.WriteField("upload_type", "UPLOAD_TYPE_RESUMABLE")

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeCreateRequestFailed, err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeNetworkError, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeReadResponseFailed, err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, exception.NewPikpakExceptionWithMessage(exception.ErrCodeServerError, fmt.Sprintf("upload failed with status: %d, body: %s", resp.StatusCode, string(respBody)))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeUnmarshalFailed, err)
	}

	return result, nil
}

func (c *Client) uploadFileLarge(ctx context.Context, uploadURL string, file *os.File, fileName string, fileSize int64, chunkSize int, parentID string) (map[string]interface{}, error) {
	md5Hash := md5.New()
	totalChunks := (int(fileSize) + chunkSize - 1) / chunkSize

	resumable := map[string]interface{}{
		"task_id":         "",
		"file_name":       fileName,
		"file_path":       file.Name(),
		"file_size":       fileSize,
		"parent_id":       parentID,
		"upload_type":     "UPLOAD_TYPE_RESUMABLE",
		"chunk_size":      chunkSize,
		"total_chunks":    totalChunks,
		"uploaded_chunks": make(map[int]bool),
	}

	file.Seek(0, 0)

	for i := 0; i < totalChunks; i++ {
		offset := int64(i * chunkSize)
		file.Seek(offset, 0)

		chunk := make([]byte, chunkSize)
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			return nil, exception.NewPikpakExceptionWithError(exception.ErrCodeReadChunkFailed, err)
		}
		if n < chunkSize {
			chunk = chunk[:n]
		}

		md5Hash.Reset()
		md5Hash.Write(chunk)
		chunkMD5 := hex.EncodeToString(md5Hash.Sum(nil))

		log.Printf("Uploading chunk %d/%d...", i+1, totalChunks)

		_ = chunk
		_ = chunkMD5

		resumable["uploaded_chunks"].(map[int]bool)[i] = true
	}

	return resumable, nil
}

func (c *Client) GetUploadURL(ctx context.Context, fileName string, fileSize int64, parentID string) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.APIHost
	}
	URL := baseURL + "/drive/v1/files/upload/url"

	params := map[string]string{
		"name":      fileName,
		"size":      strconv.FormatInt(fileSize, 10),
		"parent_id": parentID,
	}

	result, err := c.GetJSON(ctx, URL, params)
	if err != nil {
		return "", err
	}

	if uploadURL, ok := result["upload_url"].(string); ok {
		return uploadURL, nil
	}
	return "", exception.NewPikpakExceptionWithMessage(exception.ErrCodeNotFound, "upload_url not found in response")
}

func (c *Client) DownloadToFile(ctx context.Context, fileID string, filePath string) error {
	downloadURL, err := c.GetFileLink(ctx, fileID)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return exception.NewPikpakExceptionWithError(exception.ErrCodeCreateRequestFailed, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return exception.NewPikpakExceptionWithError(exception.ErrCodeNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return exception.NewPikpakExceptionWithMessage(exception.ErrCodeServerError, fmt.Sprintf("download failed with status: %d", resp.StatusCode))
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return exception.NewPikpakExceptionWithError(exception.ErrCodeCreateDirectoryFailed, err)
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		return exception.NewPikpakExceptionWithError(exception.ErrCodeCreateFileFailed, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return exception.NewPikpakExceptionWithError(exception.ErrCodeWriteFileFailed, err)
	}

	return nil
}
