package share

import (
	"context"

	"github.com/zhz8888/pikpakapi-go/internal/constants"
)

type Share struct {
	httpClient HTTPClient
	baseURL    string
}

type HTTPClient interface {
	PostJSON(ctx context.Context, url string, data interface{}) (map[string]interface{}, error)
	GetJSON(ctx context.Context, url string, params map[string]string) (map[string]interface{}, error)
}

type ShareOption func(*Share)

func NewShare(opts ...ShareOption) *Share {
	s := &Share{
		httpClient: nil,
		baseURL:    "",
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func WithShareBaseURL(baseURL string) ShareOption {
	return func(s *Share) {
		s.baseURL = baseURL
	}
}

func (s *Share) SetHTTPClient(client HTTPClient) {
	s.httpClient = client
}

func (s *Share) getBaseURL() string {
	if s.baseURL != "" {
		return s.baseURL
	}
	return "https://" + constants.APIHost
}

func (s *Share) FileBatchShare(ctx context.Context, ids []string, needPassword bool) (map[string]interface{}, error) {
	URL := s.getBaseURL() + "/drive/v1/files:batchShare"

	data := map[string]interface{}{
		"ids": ids,
		"setting": map[string]bool{
			"need_password": needPassword,
		},
	}

	return s.httpClient.PostJSON(ctx, URL, data)
}

func (s *Share) GetShareInfo(ctx context.Context, shareURL string) (map[string]interface{}, error) {
	URL := s.getBaseURL() + "/share/v1/info"

	params := map[string]string{
		"share_url": shareURL,
	}

	return s.httpClient.GetJSON(ctx, URL, params)
}

func (s *Share) Restore(ctx context.Context, shareID string, passCodeToken string, fileIDs []string) (map[string]interface{}, error) {
	URL := s.getBaseURL() + "/share/v1/file/restore"

	data := map[string]interface{}{
		"share_id":         shareID,
		"passcode_token":   passCodeToken,
		"file_ids":         fileIDs,
		"from_share_owner": false,
	}

	return s.httpClient.PostJSON(ctx, URL, data)
}
