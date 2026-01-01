package file

import (
	"context"
	"fmt"

	"github.com/zhz8888/pikpakapi-go/internal/constants"
	"github.com/zhz8888/pikpakapi-go/internal/exception"
)

const (
	DriveAPIHost = "https://" + constants.APIHost
)

type File struct {
	httpClient   HTTPClient
	baseURL      string
	tokenRefresh func(ctx context.Context) error
}

type HTTPClient interface {
	GetJSON(ctx context.Context, url string, params map[string]string) (map[string]interface{}, error)
	PostJSON(ctx context.Context, url string, data interface{}) (map[string]interface{}, error)
	PatchJSON(ctx context.Context, url string, data interface{}) (map[string]interface{}, error)
}

type FileOption func(*File)

func NewFile(opts ...FileOption) *File {
	f := &File{
		httpClient: nil,
		baseURL:    "",
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

func WithFileBaseURL(baseURL string) FileOption {
	return func(f *File) {
		f.baseURL = baseURL
	}
}

func (f *File) SetHTTPClient(client HTTPClient) {
	f.httpClient = client
}

func (f *File) SetTokenRefresh(fn func(ctx context.Context) error) {
	f.tokenRefresh = fn
}

func (f *File) getBaseURL() string {
	if f.baseURL != "" {
		return f.baseURL
	}
	return DriveAPIHost
}

func (f *File) GetFileLink(ctx context.Context, fileID string) (string, error) {
	baseURL := f.getBaseURL()
	resp, err := f.httpClient.GetJSON(ctx, fmt.Sprintf("%s/drive/v1/files/%s", baseURL, fileID), map[string]string{
		"_magic":         "2021",
		"usage":          "CACHE",
		"thumbnail_size": "SIZE_LARGE",
	})
	if err != nil {
		return "", err
	}

	url := resp["web_content_link"].(string)

	if medias, ok := resp["medias"].([]interface{}); ok && len(medias) > 0 {
		if media, ok := medias[0].(map[string]interface{}); ok {
			if link, ok := media["link"].(map[string]interface{}); ok {
				if linkUrl, ok := link["url"].(string); ok && linkUrl != "" {
					url = linkUrl
				}
			}
		}
	}

	return url, nil
}

func (f *File) Move(ctx context.Context, fileID string, parentID string) error {
	if fileID == "" {
		return exception.ErrInvalidFileID
	}

	body := map[string]interface{}{
		"ids": []string{fileID},
		"to": map[string]string{
			"parent_id": parentID,
		},
	}

	_, err := f.httpClient.PostJSON(ctx, fmt.Sprintf("%s/drive/v1/files:batchMove", f.getBaseURL()), body)
	return err
}

func (f *File) Copy(ctx context.Context, fileID string, parentID string) error {
	body := map[string]interface{}{
		"ids": []string{fileID},
		"to": map[string]string{
			"parent_id": parentID,
		},
	}

	_, err := f.httpClient.PostJSON(ctx, fmt.Sprintf("%s/drive/v1/files:batchCopy", f.getBaseURL()), body)
	return err
}

func (f *File) Rename(ctx context.Context, fileID string, newName string) error {
	if fileID == "" {
		return exception.ErrInvalidFileID
	}
	if newName == "" {
		return exception.ErrInvalidFileName
	}

	body := map[string]string{
		"name": newName,
	}

	_, err := f.httpClient.PatchJSON(ctx, fmt.Sprintf("%s/drive/v1/files/%s", f.getBaseURL(), fileID), body)
	return err
}

func (f *File) CreateFolder(ctx context.Context, name string, parentID string) (map[string]interface{}, error) {
	if name == "" {
		return nil, exception.ErrInvalidFileName
	}

	data := map[string]interface{}{
		"kind":      "drive#folder",
		"name":      name,
		"parent_id": parentID,
	}

	return f.httpClient.PostJSON(ctx, fmt.Sprintf("%s/drive/v1/files", f.getBaseURL()), data)
}

func (f *File) DeleteToTrash(ctx context.Context, ids []string) (map[string]interface{}, error) {
	if len(ids) == 0 {
		return nil, exception.ErrEmptyFileIDs
	}

	data := map[string]interface{}{
		"ids": ids,
	}

	return f.httpClient.PostJSON(ctx, fmt.Sprintf("%s/drive/v1/files:batchTrash", f.getBaseURL()), data)
}

func (f *File) Untrash(ctx context.Context, ids []string) (map[string]interface{}, error) {
	data := map[string]interface{}{
		"ids": ids,
	}

	return f.httpClient.PostJSON(ctx, fmt.Sprintf("%s/drive/v1/files:batchUntrash", f.getBaseURL()), data)
}

func (f *File) DeleteForever(ctx context.Context, ids []string) (map[string]interface{}, error) {
	if len(ids) == 0 {
		return nil, exception.ErrEmptyFileIDs
	}

	data := map[string]interface{}{
		"ids": ids,
	}

	return f.httpClient.PostJSON(ctx, fmt.Sprintf("%s/drive/v1/files:batchDelete", f.getBaseURL()), data)
}

func (f *File) FileList(ctx context.Context, size int, parentID string, nextPageToken string, query string) (map[string]interface{}, error) {
	if size == 0 {
		size = 100
	}

	filters := `{"trashed":{"eq":false},"phase":{"eq":"PHASE_TYPE_COMPLETE"}}`

	params := map[string]string{
		"parent_id":      parentID,
		"thumbnail_size": "SIZE_MEDIUM",
		"limit":          fmt.Sprintf("%d", size),
		"with_audit":     "true",
		"filters":        filters,
	}

	if nextPageToken != "" {
		params["page_token"] = nextPageToken
	}

	if query != "" {
		params["query"] = query
	}

	return f.httpClient.GetJSON(ctx, fmt.Sprintf("%s/drive/v1/files", f.getBaseURL()), params)
}

func (f *File) GetAbout(ctx context.Context) (map[string]interface{}, error) {
	return f.httpClient.GetJSON(ctx, fmt.Sprintf("%s/drive/v1/about", f.getBaseURL()), nil)
}
