package download

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhz8888/pikpakapi-go/internal/constants"
	"github.com/zhz8888/pikpakapi-go/internal/exception"
	"github.com/zhz8888/pikpakapi-go/pkg/enums"
)

type Download struct {
	httpClient HTTPClient
	baseURL    string
}

type HTTPClient interface {
	PostJSON(ctx context.Context, url string, data interface{}) (map[string]interface{}, error)
	GetJSON(ctx context.Context, url string, params map[string]string) (map[string]interface{}, error)
	Delete(ctx context.Context, url string, params map[string]string) (map[string]interface{}, error)
}

type DownloadOption func(*Download)

func NewDownload(opts ...DownloadOption) *Download {
	d := &Download{
		httpClient: nil,
		baseURL:    "",
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func WithDownloadBaseURL(baseURL string) DownloadOption {
	return func(d *Download) {
		d.baseURL = baseURL
	}
}

func (d *Download) SetHTTPClient(client HTTPClient) {
	d.httpClient = client
}

func (d *Download) getBaseURL() string {
	if d.baseURL != "" {
		return d.baseURL
	}
	return "https://" + constants.APIHost
}

func (d *Download) OfflineDownload(ctx context.Context, fileURL string, parentID string, name string) (map[string]interface{}, error) {
	if fileURL == "" {
		return nil, exception.NewPikpakExceptionWithMessage(exception.ErrCodeInvalidURL, "file url is required")
	}

	URL := d.getBaseURL() + "/drive/v1/files"

	downloadData := map[string]interface{}{
		"kind":        "drive#file",
		"name":        name,
		"upload_type": "UPLOAD_TYPE_URL",
		"url":         map[string]string{"url": fileURL},
	}

	if parentID != "" {
		downloadData["parent_id"] = parentID
		downloadData["folder_type"] = ""
	} else {
		downloadData["folder_type"] = "DOWNLOAD"
	}

	return d.httpClient.PostJSON(ctx, URL, downloadData)
}

func (d *Download) CaptureScreenshot(ctx context.Context, fileID string) (map[string]interface{}, error) {
	if fileID == "" {
		return nil, exception.ErrInvalidFileID
	}

	URL := d.getBaseURL() + "/drive/v1/files:testScreenshot"

	data := map[string]interface{}{
		"file_id": fileID,
	}

	return d.httpClient.PostJSON(ctx, URL, data)
}

func (d *Download) RemoteDownload(ctx context.Context, fileURL string) (map[string]interface{}, error) {
	if fileURL == "" {
		return nil, exception.ErrInvalidURL
	}

	URL := d.getBaseURL() + "/drive/v1/files"

	data := map[string]interface{}{
		"kind":        "drive#task",
		"upload_type": "UPLOAD_TYPE_URL",
		"url":         map[string]string{"url": fileURL},
	}

	return d.httpClient.PostJSON(ctx, URL, data)
}

func (d *Download) OfflineList(ctx context.Context, size int, nextPageToken string, phases []string) (map[string]interface{}, error) {
	if size == 0 {
		size = 10000
	}

	if phases == nil {
		phases = []string{"PHASE_TYPE_RUNNING", "PHASE_TYPE_ERROR"}
	}

	URL := d.getBaseURL() + "/drive/v1/tasks"

	filters := fmt.Sprintf(`{"phase":{"in":"%s"}}`, strings.Join(phases, ","))

	params := map[string]string{
		"limit":   fmt.Sprintf("%d", size),
		"filters": filters,
	}

	if nextPageToken != "" {
		params["page_token"] = nextPageToken
	}

	return d.httpClient.GetJSON(ctx, URL, params)
}

func (d *Download) DeleteOfflineTasks(ctx context.Context, taskIDs []string, deleteFiles bool) error {
	URL := d.getBaseURL() + "/drive/v1/tasks"

	params := map[string]string{
		"task_ids":     strings.Join(taskIDs, ","),
		"delete_files": fmt.Sprintf("%t", deleteFiles),
	}

	_, err := d.httpClient.Delete(ctx, URL, params)
	return err
}

func (d *Download) OfflineTaskRetry(ctx context.Context, taskID string) (map[string]interface{}, error) {
	URL := d.getBaseURL() + "/drive/v1/task"

	data := map[string]interface{}{
		"type":        "offline",
		"create_type": "RETRY",
		"id":          taskID,
	}

	return d.httpClient.PostJSON(ctx, URL, data)
}

func (d *Download) DeleteTasks(ctx context.Context, taskIDs []string, deleteFiles bool) error {
	URL := d.getBaseURL() + "/drive/v1/tasks"

	params := map[string]string{
		"task_ids":     strings.Join(taskIDs, ","),
		"delete_files": fmt.Sprintf("%t", deleteFiles),
	}

	_, err := d.httpClient.Delete(ctx, URL, params)
	return err
}

func (d *Download) GetTaskStatus(ctx context.Context, taskID string, fileID string) (enums.DownloadStatus, error) {
	fileInfo, err := d.OfflineFileInfo(ctx, fileID)
	if err != nil {
		return enums.DownloadStatusNotFound, err
	}

	if phase, ok := fileInfo["phase"].(string); ok {
		return enums.ParseDownloadStatus(phase), nil
	}

	return enums.DownloadStatusNotFound, nil
}

func (d *Download) OfflineFileInfo(ctx context.Context, fileID string) (map[string]interface{}, error) {
	if fileID == "" {
		return nil, exception.ErrInvalidFileID
	}

	URL := d.getBaseURL() + "/drive/v1/files/" + fileID

	return d.httpClient.GetJSON(ctx, URL, nil)
}
