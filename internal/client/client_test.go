package client

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/zhz8888/pikpakapi-go/pkg/enums"
)

func TestNewClient(t *testing.T) {
	cli := NewClient(
		WithUsername("test_user"),
		WithPassword("test_pass"),
		WithMaxRetries(5),
	)

	if cli.username != "test_user" {
		t.Errorf("Expected username 'test_user', got '%s'", cli.username)
	}

	if cli.password != "test_pass" {
		t.Errorf("Expected password 'test_pass', got '%s'", cli.password)
	}

	if cli.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", cli.maxRetries)
	}
}

func TestNewClient_WithDeviceID(t *testing.T) {
	cli := NewClient(
		WithDeviceID("custom_device_id"),
	)

	if cli.GetDeviceID() != "custom_device_id" {
		t.Errorf("Expected deviceID 'custom_device_id', got '%s'", cli.GetDeviceID())
	}
}

func TestNewClient_DefaultDeviceID(t *testing.T) {
	cli := NewClient(
		WithUsername("test_user"),
		WithPassword("test_pass"),
	)

	if cli.GetDeviceID() == "" {
		t.Error("Expected deviceID to be generated")
	}
}

func TestGetUserInfo(t *testing.T) {
	cli := NewClient(
		WithUsername("test_user"),
		WithPassword("test_pass"),
	)

	cli.SetAccessToken("test_access")
	cli.SetRefreshToken("test_refresh")
	cli.SetUserID("test_user_id")
	cli.SetEncodedToken("test_encoded")

	info := cli.GetUserInfo()

	if info["username"] != "test_user" {
		t.Errorf("Expected username 'test_user', got '%s'", info["username"])
	}

	if info["access_token"] != "test_access" {
		t.Errorf("Expected access_token 'test_access', got '%s'", info["access_token"])
	}

	if info["refresh_token"] != "test_refresh" {
		t.Errorf("Expected refresh_token 'test_refresh', got '%s'", info["refresh_token"])
	}

	if info["user_id"] != "test_user_id" {
		t.Errorf("Expected user_id 'test_user_id', got '%s'", info["user_id"])
	}

	if info["encoded_token"] != "test_encoded" {
		t.Errorf("Expected encoded_token 'test_encoded', got '%s'", info["encoded_token"])
	}
}

func TestClient_Login_NoCredentials(t *testing.T) {
	cli := NewClient()

	err := cli.Login(context.Background())
	if err == nil {
		t.Error("Expected error when username and password are empty")
	}
}

func TestClient_RefreshAccessToken_NoRefreshToken(t *testing.T) {
	cli := NewClient()

	err := cli.RefreshAccessToken(context.Background())
	if err == nil {
		t.Error("Expected error when refresh_token is empty")
	}
}

func TestStorageInfo_Scenarios(t *testing.T) {
	tests := []struct {
		name        string
		info        StorageInfo
		expectValid bool
		expectUsage float64
	}{
		{
			name: "standard_storage",
			info: StorageInfo{
				TotalBytes:    100000000000,
				UsedBytes:     50000000000,
				TrashBytes:    1000000000,
				IsUnlimited:   false,
				Complimentary: "basic",
				ExpiresAt:     "2025-12-31T23:59:59Z",
				UserType:      1,
			},
			expectValid: true,
			expectUsage: 50.0,
		},
		{
			name: "unlimited_storage",
			info: StorageInfo{
				TotalBytes:    0,
				UsedBytes:     75000000000,
				TrashBytes:    500000000,
				IsUnlimited:   true,
				Complimentary: "premium",
			},
			expectValid: true,
			expectUsage: 0,
		},
		{
			name: "zero_usage",
			info: StorageInfo{
				TotalBytes:    10000000000,
				UsedBytes:     0,
				TrashBytes:    0,
				IsUnlimited:   false,
				Complimentary: "basic",
			},
			expectValid: true,
			expectUsage: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.info.TotalBytes != 0 && tt.expectValid {
				if tt.info.TotalBytes != 100000000000 && tt.info.TotalBytes != 10000000000 {
					t.Errorf("Unexpected TotalBytes %d", tt.info.TotalBytes)
				}
			}

			if tt.info.UsedBytes != 50000000000 && tt.info.UsedBytes != 0 && tt.info.UsedBytes != 75000000000 {
				t.Errorf("Unexpected UsedBytes %d", tt.info.UsedBytes)
			}

			if tt.info.IsUnlimited && tt.info.TotalBytes != 0 {
				t.Error("Expected TotalBytes 0 for unlimited storage")
			}

			if !tt.info.IsUnlimited && tt.info.TotalBytes == 0 {
				t.Error("Expected non-zero TotalBytes for non-unlimited storage")
			}

			if tt.expectUsage == 0 && tt.info.TotalBytes > 0 {
				usagePercent := float64(tt.info.UsedBytes) / float64(tt.info.TotalBytes) * 100
				if usagePercent != 0 {
					t.Errorf("Expected 0%% usage, got %.2f%%", usagePercent)
				}
			}
		})
	}
}

func TestGetStorageInfo_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		response      map[string]interface{}
		expectError   bool
		expectUnlimit bool
	}{
		{
			name: "success_standard",
			response: map[string]interface{}{
				"quota": map[string]interface{}{
					"limit":          "100000000000",
					"usage":          "50000000000",
					"usage_in_trash": "1000000000",
					"is_unlimited":   false,
					"complimentary":  "basic",
				},
			},
			expectError:   false,
			expectUnlimit: false,
		},
		{
			name: "success_unlimited",
			response: map[string]interface{}{
				"quota": map[string]interface{}{
					"limit":          "0",
					"usage":          "75000000000",
					"usage_in_trash": "500000000",
					"is_unlimited":   true,
					"complimentary":  "premium",
				},
			},
			expectError:   false,
			expectUnlimit: true,
		},
		{
			name: "missing_quota",
			response: map[string]interface{}{
				"some_other_field": "value",
			},
			expectError:   false,
			expectUnlimit: false,
		},
		{
			name: "invalid_quota_format",
			response: map[string]interface{}{
				"quota": map[string]interface{}{
					"limit":          "invalid_number",
					"usage":          "invalid_usage",
					"usage_in_trash": "1000000000",
					"is_unlimited":   false,
					"complimentary":  "basic",
				},
			},
			expectError:   false,
			expectUnlimit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET method, got %s", r.Method)
				}

				expectedPath := "/drive/v1/about"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

			resp, err := cli.GetStorageInfo(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for " + tt.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if tt.expectUnlimit {
				if !resp.IsUnlimited {
					t.Error("Expected IsUnlimited to be true")
				}
				if resp.TotalBytes != 0 {
					t.Errorf("Expected TotalBytes 0 for unlimited storage, got %d", resp.TotalBytes)
				}
			} else if tt.name == "missing_quota" || tt.name == "invalid_quota_format" {
				if resp.TotalBytes != 0 {
					t.Errorf("Expected TotalBytes 0 for %s, got %d", tt.name, resp.TotalBytes)
				}
				if resp.UsedBytes != 0 {
					t.Errorf("Expected UsedBytes 0 for %s, got %d", tt.name, resp.UsedBytes)
				}
			} else {
				if resp.TotalBytes != 100000000000 {
					t.Errorf("Expected TotalBytes 100000000000, got %d", resp.TotalBytes)
				}
				if resp.UsedBytes != 50000000000 {
					t.Errorf("Expected UsedBytes 50000000000, got %d", resp.UsedBytes)
				}
			}
		})
	}
}

func TestGetFileLink_Scenarios(t *testing.T) {
	tests := []struct {
		name        string
		response    map[string]interface{}
		expectedURL string
	}{
		{
			name: "web_content_link_only",
			response: map[string]interface{}{
				"web_content_link": "https://example.com/download/test",
				"medias":           []interface{}{},
			},
			expectedURL: "https://example.com/download/test",
		},
		{
			name: "with_media_link",
			response: map[string]interface{}{
				"web_content_link": "https://example.com/download/test",
				"medias": []interface{}{
					map[string]interface{}{
						"link": map[string]interface{}{
							"url": "https://example.com/media/test.mp4",
						},
					},
				},
			},
			expectedURL: "https://example.com/media/test.mp4",
		},
		{
			name: "media_link_empty",
			response: map[string]interface{}{
				"web_content_link": "https://example.com/download/test",
				"medias": []interface{}{
					map[string]interface{}{
						"link": map[string]interface{}{
							"url": "",
						},
					},
				},
			},
			expectedURL: "https://example.com/download/test",
		},
		{
			name: "no_medias_field",
			response: map[string]interface{}{
				"web_content_link": "https://example.com/download/test",
			},
			expectedURL: "https://example.com/download/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET method, got %s", r.Method)
				}

				expectedPath := "/drive/v1/files/test_file_id"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				if r.URL.Query().Get("_magic") != "2021" {
					t.Error("Expected _magic parameter to be 2021")
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

			url, err := cli.GetFileLink(context.Background(), "test_file_id")
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if url != tt.expectedURL {
				t.Errorf("Expected URL '%s', got '%s'", tt.expectedURL, url)
			}
		})
	}
}

func TestMove_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchMove"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok || len(ids) != 1 || ids[0] != "test_file_id" {
			t.Error("Expected ids to contain 'test_file_id'")
		}

		to, ok := body["to"].(map[string]interface{})
		if !ok || to["parent_id"] != "new_parent_id" {
			t.Error("Expected to.parent_id to be 'new_parent_id'")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	err := cli.Move(context.Background(), "test_file_id", "new_parent_id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestMove_EmptyFileID(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	err := cli.Move(context.Background(), "", "new_parent_id")
	if err == nil {
		t.Error("Expected error when file ID is empty")
	}
}

func TestCopy_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchCopy"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok || len(ids) != 1 || ids[0] != "test_file_id" {
			t.Error("Expected ids to contain 'test_file_id'")
		}

		to, ok := body["to"].(map[string]interface{})
		if !ok || to["parent_id"] != "new_parent_id" {
			t.Error("Expected to.parent_id to be 'new_parent_id'")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	err := cli.Copy(context.Background(), "test_file_id", "new_parent_id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestCopy_SameParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchCopy"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok || len(ids) != 1 || ids[0] != "test_file_id" {
			t.Error("Expected ids to contain 'test_file_id'")
		}

		to, ok := body["to"].(map[string]interface{})
		if !ok || to["parent_id"] != "same_parent_id" {
			t.Error("Expected to.parent_id to be 'same_parent_id'")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	err := cli.Copy(context.Background(), "test_file_id", "same_parent_id")
	if err != nil {
		t.Fatalf("Expected no error when copying to same parent, got %v", err)
	}
}

func TestRename_Scenarios(t *testing.T) {
	tests := []struct {
		name        string
		newName     string
		expectError bool
	}{
		{
			name:        "success",
			newName:     "new_file_name",
			expectError: false,
		},
		{
			name:        "empty_name",
			newName:     "",
			expectError: true,
		},
		{
			name:        "special_characters",
			newName:     "新文件名_123!@#",
			expectError: false,
		},
		{
			name:        "unicode",
			newName:     "日本語ファイル名.txt",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH method, got %s", r.Method)
				}

				expectedPath := "/drive/v1/files/test_file_id"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)

				if body["name"] != tt.newName {
					t.Errorf("Expected name '%s', got '%s'", tt.newName, body["name"])
				}

				response := map[string]interface{}{
					"code": "OK",
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

			err := cli.Rename(context.Background(), "test_file_id", tt.newName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for " + tt.name)
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestPatchJSON_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files/test_file_id"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json; charset=utf-8" {
			t.Errorf("Expected Content-Type 'application/json; charset=utf-8', got '%s'", contentType)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "updated_name" {
			t.Errorf("Expected name 'updated_name', got '%s'", body["name"])
		}

		response := map[string]interface{}{
			"id":   "test_file_id",
			"name": "updated_name",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.PatchJSON(context.Background(), server.URL+"/drive/v1/files/test_file_id", map[string]interface{}{"name": "updated_name"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["name"] != "updated_name" {
		t.Errorf("Expected name 'updated_name', got '%v'", result["name"])
	}
}

func TestDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files/test_file_id"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	_, err := cli.Delete(context.Background(), server.URL+"/drive/v1/files/test_file_id", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDelete_EmptyFileID(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	_, err := cli.Delete(context.Background(), "", nil)
	if err == nil {
		t.Error("Expected error when file ID is empty")
	}
}

func TestCreateFolder_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "new_folder" {
			t.Errorf("Expected name 'new_folder', got '%s'", body["name"])
		}

		if body["kind"] != "drive#folder" {
			t.Errorf("Expected kind 'drive#folder', got '%s'", body["kind"])
		}

		if body["parent_id"] != "parent_folder_id" {
			t.Errorf("Expected parent_id 'parent_folder_id', got '%s'", body["parent_id"])
		}

		response := map[string]interface{}{
			"id":   "new_folder_id",
			"name": "new_folder",
			"kind": "drive#folder",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.CreateFolder(context.Background(), "new_folder", "parent_folder_id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["name"] != "new_folder" {
		t.Errorf("Expected name 'new_folder', got '%v'", result["name"])
	}
}

func TestCreateFolder_EmptyName(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	_, err := cli.CreateFolder(context.Background(), "", "parent_id")
	if err == nil {
		t.Error("Expected error when folder name is empty")
	}
}

func TestCreateFile_Success(t *testing.T) {
	uploadServerURL := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/drive/v1/files/upload/url") {
			if r.URL.Query().Get("name") != "test_file.txt" {
				t.Errorf("Expected name 'test_file.txt', got '%s'", r.URL.Query().Get("name"))
			}

			response := map[string]interface{}{
				"upload_url": uploadServerURL,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/upload" {
			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				t.Errorf("Expected multipart/form-data content type, got '%s'", contentType)
			}

			mr := multipart.NewReader(r.Body, contentType[len("multipart/form-data; boundary="):])
			form, err := mr.ReadForm(10 * 1024 * 1024)
			if err != nil {
				t.Fatalf("Failed to read multipart form: %v", err)
			}

			if form.Value["name"] == nil || form.Value["name"][0] != "test_file.txt" {
				t.Errorf("Expected name 'test_file.txt', got '%v'", form.Value["name"])
			}

			if form.Value["kind"] == nil || form.Value["kind"][0] != "drive#file" {
				t.Errorf("Expected kind 'drive#file', got '%v'", form.Value["kind"])
			}

			if form.File["file"] == nil {
				t.Error("Expected file field in form")
			}

			response := map[string]interface{}{
				"id":          "upload_token_id",
				"upload_type": "UPLOAD_TYPE_RESUMABLE",
				"resumable": map[string]interface{}{
					"endpoint": "https://upload.example.com/upload",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	uploadServerURL = server.URL + "/upload"

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	tmpFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("test content"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	defer tmpFile.Close()

	result, err := cli.UploadReader(context.Background(), tmpFile, "test_file.txt", int64(len("test content")), "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["id"] != "upload_token_id" {
		t.Errorf("Expected id 'upload_token_id', got '%v'", result["id"])
	}
}

func TestCreateFile_WithParent(t *testing.T) {
	uploadServerURL := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/drive/v1/files/upload/url") {
			if r.URL.Query().Get("name") != "test_file.txt" {
				t.Errorf("Expected name 'test_file.txt', got '%s'", r.URL.Query().Get("name"))
			}

			if r.URL.Query().Get("parent_id") != "parent_id_value" {
				t.Errorf("Expected parent_id 'parent_id_value', got '%s'", r.URL.Query().Get("parent_id"))
			}

			response := map[string]interface{}{
				"upload_url": uploadServerURL,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/upload" {
			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				t.Errorf("Expected multipart/form-data content type, got '%s'", contentType)
			}

			mr := multipart.NewReader(r.Body, contentType[len("multipart/form-data; boundary="):])
			form, err := mr.ReadForm(10 * 1024 * 1024)
			if err != nil {
				t.Fatalf("Failed to read multipart form: %v", err)
			}

			if form.Value["parent_id"] == nil || form.Value["parent_id"][0] != "parent_id_value" {
				t.Errorf("Expected parent_id 'parent_id_value', got '%v'", form.Value["parent_id"])
			}

			response := map[string]interface{}{
				"id": "upload_token_id",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	uploadServerURL = server.URL + "/upload"

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	tmpFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("test content"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	defer tmpFile.Close()

	result, err := cli.UploadReader(context.Background(), tmpFile, "test_file.txt", int64(len("test content")), "parent_id_value")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		if r.URL.Query().Get("parent_id") != "test_folder_id" {
			t.Error("Expected parent_id parameter to be 'test_folder_id'")
		}

		if r.URL.Query().Get("limit") != "50" {
			t.Error("Expected limit parameter to be '50'")
		}

		response := map[string]interface{}{
			"files": []interface{}{
				map[string]interface{}{
					"id":   "file_id_1",
					"name": "file1.txt",
					"kind": "drive#file",
				},
				map[string]interface{}{
					"id":   "folder_id_1",
					"name": "folder1",
					"kind": "drive#folder",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.FileList(context.Background(), 50, "test_folder_id", "", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	files, ok := result["files"].([]interface{})
	if !ok {
		t.Fatal("Expected files to be an array")
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestList_DefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "100" {
			t.Error("Expected default limit to be 100")
		}

		response := map[string]interface{}{
			"files": []interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	_, err := cli.FileList(context.Background(), 0, "folder_id", "", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		if r.URL.Query().Get("query") != "test query" {
			t.Error("Expected query parameter to be 'test query'")
		}

		response := map[string]interface{}{
			"files": []interface{}{
				map[string]interface{}{
					"id":   "search_result_id",
					"name": "test_result.txt",
					"kind": "drive#file",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.FileList(context.Background(), 50, "", "", "test query")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	files, ok := result["files"].([]interface{})
	if !ok {
		t.Fatal("Expected files to be an array")
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"files": []interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	_, err := cli.FileList(context.Background(), 50, "", "", "")
	if err != nil {
		t.Fatalf("Expected no error for empty query, got %v", err)
	}
}

func TestGetFileInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files/test_file_id"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"id":        "test_file_id",
			"name":      "test_file.txt",
			"kind":      "drive#file",
			"mime_type": "text/plain",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.GetJSON(context.Background(), server.URL+"/drive/v1/files/test_file_id", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["name"] != "test_file.txt" {
		t.Errorf("Expected name 'test_file.txt', got '%v'", result["name"])
	}
}

func TestGetFileInfo_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	_, err := cli.GetJSON(context.Background(), server.URL+"/drive/v1/files/nonexistent_id", nil)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestBatchDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchTrash"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok {
			t.Error("Expected ids field in body")
		}

		if len(ids) != 2 {
			t.Errorf("Expected 2 ids, got %d", len(ids))
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	_, err := cli.DeleteToTrash(context.Background(), []string{"file_id_1", "file_id_2"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestBatchDelete_EmptyIDs(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	_, err := cli.DeleteToTrash(context.Background(), []string{})
	if err == nil {
		t.Error("Expected error for empty file IDs")
	}
}

func TestSort_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:sort"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["id"] != "test_folder_id" {
			t.Errorf("Expected id 'test_folder_id', got '%s'", body["id"])
		}

		if body["sort_name"] != "FILES_TYPE_THUMBNAIL" {
			t.Errorf("Expected sort_name 'FILES_TYPE_THUMBNAIL', got '%s'", body["sort_name"])
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	_, err := cli.PatchJSON(context.Background(), server.URL+"/drive/v1/files:sort", map[string]interface{}{
		"id":        "test_folder_id",
		"sort_name": "FILES_TYPE_THUMBNAIL",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSort_InvalidFolderID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	_, err := cli.PatchJSON(context.Background(), server.URL+"/drive/v1/files:sort", map[string]interface{}{
		"id":        "",
		"sort_name": "FILES_TYPE_THUMBNAIL",
	})
	if err == nil {
		t.Error("Expected error for empty folder ID")
	}
}

func TestStartDownload_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["kind"] != "drive#file" {
			t.Errorf("Expected kind 'drive#file', got '%s'", body["kind"])
		}

		if body["name"] != "Test Task" {
			t.Errorf("Expected name 'Test Task', got '%s'", body["name"])
		}

		if body["upload_type"] != "UPLOAD_TYPE_URL" {
			t.Errorf("Expected upload_type 'UPLOAD_TYPE_URL', got '%s'", body["upload_type"])
		}

		urlMap, ok := body["url"].(map[string]interface{})
		if !ok {
			t.Error("Expected url field to be a map")
		} else if urlMap["url"] != "magnet:test_link" {
			t.Errorf("Expected url 'magnet:test_link', got '%v'", urlMap["url"])
		}

		response := map[string]interface{}{
			"id":      "new_task_id",
			"file_id": "test_file_id",
			"status":  "downloading",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.OfflineDownload(context.Background(), "magnet:test_link", "", "Test Task")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["id"] != "new_task_id" {
		t.Errorf("Expected task_id 'new_task_id', got '%v'", result["id"])
	}
}

func TestStartDownload_EmptyFileID(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	_, err := cli.OfflineDownload(context.Background(), "", "", "")
	if err == nil {
		t.Error("Expected error for empty file ID")
	}
}

func TestGetDownloadTask_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files/test_task_id"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"id":      "test_task_id",
			"file_id": "test_file_id",
			"status":  "done",
			"phase":   "PHASE_TYPE_COMPLETE",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.OfflineFileInfo(context.Background(), "test_task_id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["status"] != "done" {
		t.Errorf("Expected status 'done', got '%v'", result["status"])
	}
}

func TestGetDownloadTask_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	_, err := cli.OfflineFileInfo(context.Background(), "nonexistent_task")
	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}

func TestDeleteDownloadTask_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/tasks"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		if r.URL.Query().Get("task_ids") != "test_task_id" {
			t.Errorf("Expected task_ids 'test_task_id', got '%s'", r.URL.Query().Get("task_ids"))
		}

		if r.URL.Query().Get("delete_files") != "false" {
			t.Errorf("Expected delete_files 'false', got '%s'", r.URL.Query().Get("delete_files"))
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	err := cli.DeleteTasks(context.Background(), []string{"test_task_id"}, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDeleteDownloadTask_EmptyTaskID(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	err := cli.DeleteTasks(context.Background(), []string{}, false)
	if err == nil {
		t.Error("Expected error for empty task ID")
	}
}

func TestWithAccessToken(t *testing.T) {
	cli := NewClient(WithAccessToken("test_access_token"))

	if cli.GetAccessToken() != "test_access_token" {
		t.Errorf("Expected accessToken 'test_access_token', got '%s'", cli.GetAccessToken())
	}
}

func TestWithBaseURL(t *testing.T) {
	cli := NewClient(WithBaseURL("https://custom.api.example.com"))

	if cli.baseURL != "https://custom.api.example.com" {
		t.Errorf("Expected baseURL 'https://custom.api.example.com', got '%s'", cli.baseURL)
	}
}

func TestWithMaxRetries(t *testing.T) {
	cli := NewClient(WithMaxRetries(10))

	if cli.maxRetries != 10 {
		t.Errorf("Expected maxRetries 10, got %d", cli.maxRetries)
	}
}

func TestGetSortOptions_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/sort_options"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"sort_options": []interface{}{
				map[string]interface{}{
					"sort_name":    "FILES_TYPE_THUMBNAIL",
					"display_name": "Thumbnail",
				},
				map[string]interface{}{
					"sort_name":    "FILES_TYPE_NAME",
					"display_name": "Name",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.GetJSON(context.Background(), server.URL+"/drive/v1/sort_options", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	sortOptions, ok := result["sort_options"].([]interface{})
	if !ok {
		t.Fatal("Expected sort_options to be an array")
	}

	if len(sortOptions) != 2 {
		t.Errorf("Expected 2 sort options, got %d", len(sortOptions))
	}
}

func TestGetSortOptions_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"sort_options": []interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.GetJSON(context.Background(), server.URL+"/drive/v1/sort_options", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	sortOptions, ok := result["sort_options"].([]interface{})
	if !ok || len(sortOptions) != 0 {
		t.Error("Expected empty sort options")
	}
}

func TestCaptureScreenshot_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:testScreenshot"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["file_id"] != "test_file_id" {
			t.Errorf("Expected file_id 'test_file_id', got '%s'", body["file_id"])
		}

		response := map[string]interface{}{
			"task_id": "screenshot_task_id",
			"file_id": "test_file_id",
			"status":  "done",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.CaptureScreenshot(context.Background(), "test_file_id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["task_id"] != "screenshot_task_id" {
		t.Errorf("Expected task_id 'screenshot_task_id', got '%v'", result["task_id"])
	}
}

func TestCaptureScreenshot_EmptyFileID(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	_, err := cli.CaptureScreenshot(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty file ID")
	}
}

func TestRemoteDownload_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["kind"] != "drive#task" {
			t.Errorf("Expected kind 'drive#task', got '%s'", body["kind"])
		}

		urlMap, ok := body["url"].(map[string]interface{})
		if !ok {
			t.Error("Expected url to be a map")
		} else if urlMap["url"] != "https://example.com/file.mp4" {
			t.Errorf("Expected url 'https://example.com/file.mp4', got '%v'", urlMap["url"])
		}

		response := map[string]interface{}{
			"task_id": "remote_task_id",
			"file_id": "new_file_id",
			"status":  "downloading",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.RemoteDownload(context.Background(), "https://example.com/file.mp4")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["task_id"] != "remote_task_id" {
		t.Errorf("Expected task_id 'remote_task_id', got '%v'", result["task_id"])
	}
}

func TestRemoteDownload_EmptyURL(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	_, err := cli.RemoteDownload(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty URL")
	}
}

func TestWithAllOptions(t *testing.T) {
	cli := NewClient(
		WithUsername("user@example.com"),
		WithPassword("secure_password"),
		WithDeviceID("unique_device_id"),
		WithAccessToken("access_token_value"),
		WithRefreshToken("refresh_token_value"),
		WithBaseURL("https://api.custom.com"),
		WithMaxRetries(15),
	)

	if cli.username != "user@example.com" {
		t.Errorf("Expected username 'user@example.com', got '%s'", cli.username)
	}
	if cli.password != "secure_password" {
		t.Errorf("Expected password 'secure_password', got '%s'", cli.password)
	}
	if cli.GetDeviceID() != "unique_device_id" {
		t.Errorf("Expected deviceID 'unique_device_id', got '%s'", cli.GetDeviceID())
	}
	if cli.GetAccessToken() != "access_token_value" {
		t.Errorf("Expected accessToken 'access_token_value', got '%s'", cli.GetAccessToken())
	}
	if cli.GetRefreshToken() != "refresh_token_value" {
		t.Errorf("Expected refreshToken 'refresh_token_value', got '%s'", cli.GetRefreshToken())
	}
	if cli.baseURL != "https://api.custom.com" {
		t.Errorf("Expected baseURL 'https://api.custom.com', got '%s'", cli.baseURL)
	}
	if cli.maxRetries != 15 {
		t.Errorf("Expected maxRetries 15, got %d", cli.maxRetries)
	}
}

func TestClient_OptionOrder(t *testing.T) {
	cli := NewClient(
		WithMaxRetries(1),
		WithUsername("first"),
		WithPassword("second"),
		WithMaxRetries(5),
	)

	if cli.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", cli.maxRetries)
	}
	if cli.username != "first" {
		t.Errorf("Expected username 'first', got '%s'", cli.username)
	}
	if cli.password != "second" {
		t.Errorf("Expected password 'second', got '%s'", cli.password)
	}
}

func TestClient_DefaultValues(t *testing.T) {
	cli := NewClient()

	if cli.maxRetries != 3 {
		t.Errorf("Expected default maxRetries 3, got %d", cli.maxRetries)
	}
}

func TestURLConstruction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/drive/v1/about"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"quota": map[string]interface{}{
				"limit":            "10000000000",
				"used":             "1000000000",
				"usage":            "1000000000",
				"trash":            "100000000",
				"usage_in_trash":   "100000000",
				"over_quota":       false,
				"is_unlimited":     false,
				"complimentary":    "",
				"system_indicator": "default",
				"expand":           []interface{}{},
			},
		})
	}))
	defer server.Close()

	tests := []struct {
		name string
		host string
	}{
		{"http_server", server.URL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewClient(WithBaseURL(tt.host))

			_, err := cli.GetStorageInfo(context.Background())
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		})
	}
}

func TestTokenRefresh_EmptyRefreshToken(t *testing.T) {
	cli := NewClient(
		WithAccessToken("access_token"),
	)

	err := cli.RefreshAccessToken(context.Background())
	if err == nil {
		t.Error("Expected error when refresh_token is empty")
	}
}

func TestGetUserInfo_EmptyTokens(t *testing.T) {
	cli := NewClient()

	info := cli.GetUserInfo()

	if info["username"] != "" {
		t.Errorf("Expected empty username, got '%s'", info["username"])
	}
}

func TestStorageInfo_ExpiresAt(t *testing.T) {
	info := StorageInfo{
		TotalBytes: 100000000000,
		UsedBytes:  50000000000,
		ExpiresAt:  "2025-12-31T23:59:59Z",
	}

	if info.ExpiresAt == "" {
		t.Error("Expected ExpiresAt to be set")
	}

	if info.ExpiresAt != "2025-12-31T23:59:59Z" {
		t.Errorf("Expected ExpiresAt '2025-12-31T23:59:59Z', got '%s'", info.ExpiresAt)
	}
}

func TestFileKind_Check(t *testing.T) {
	tests := []struct {
		name     string
		fileKind enums.FileKind
		expected string
	}{
		{"file_kind", enums.FileKindFile, "drive#file"},
		{"folder_kind", enums.FileKindFolder, "drive#folder"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.fileKind) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.fileKind))
			}
		})
	}
}

func TestDownloadStatus_Check(t *testing.T) {
	tests := []struct {
		name     string
		status   enums.DownloadStatus
		expected string
	}{
		{"not_downloading", enums.DownloadStatusNotDownloading, "not_downloading"},
		{"downloading", enums.DownloadStatusDownloading, "downloading"},
		{"done", enums.DownloadStatusDone, "done"},
		{"error", enums.DownloadStatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.status))
			}
		})
	}
}

func TestEncodeToken_Success(t *testing.T) {
	cli := NewClient()
	cli.SetAccessToken("test_access_token")
	cli.SetRefreshToken("test_refresh_token")

	err := cli.EncodeToken()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cli.GetEncodedToken() == "" {
		t.Error("Expected encodedToken to be set")
	}

	expected := "eyJhY2Nlc3NfdG9rZW4iOiJ0ZXN0X2FjY2Vzc190b2tlbiIsInJlZnJlc2hfdG9rZW4iOiJ0ZXN0X3JlZnJlc2hfdG9rZW4ifQ=="
	if cli.GetEncodedToken() != expected {
		t.Errorf("Expected encodedToken '%s', got '%s'", expected, cli.GetEncodedToken())
	}
}

func TestEncodeToken_EmptyTokens(t *testing.T) {
	cli := NewClient()

	err := cli.EncodeToken()
	if err != nil {
		t.Errorf("Expected no error when tokens are empty, got %v", err)
	}

	expected := "eyJhY2Nlc3NfdG9rZW4iOiIiLCJyZWZyZXNoX3Rva2VuIjoiIn0="
	if cli.GetEncodedToken() != expected {
		t.Errorf("Expected encodedToken '%s', got '%s'", expected, cli.GetEncodedToken())
	}
}

func TestDecodeToken_Success(t *testing.T) {
	cli := NewClient()
	cli.SetEncodedToken("eyJhY2Nlc3NfdG9rZW4iOiJ0ZXN0X2FjY2Vzc190b2tlbiIsInJlZnJlc2hfdG9rZW4iOiJ0ZXN0X3JlZnJlc2hfdG9rZW4ifQ==")

	err := cli.DecodeToken()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cli.GetAccessToken() != "test_access_token" {
		t.Errorf("Expected accessToken 'test_access_token', got '%s'", cli.GetAccessToken())
	}

	if cli.GetRefreshToken() != "test_refresh_token" {
		t.Errorf("Expected refreshToken 'test_refresh_token', got '%s'", cli.GetRefreshToken())
	}
}

func TestDecodeToken_EmptyToken(t *testing.T) {
	cli := NewClient()

	err := cli.DecodeToken()
	if err == nil {
		t.Error("Expected error when encodedToken is empty")
	}
}

func TestDecodeToken_InvalidToken(t *testing.T) {
	cli := NewClient()
	cli.SetEncodedToken("invalid_base64_token!!!")

	err := cli.DecodeToken()
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestGetEncodedToken(t *testing.T) {
	cli := NewClient()
	cli.SetEncodedToken("test_encoded_token")

	token := cli.GetEncodedToken()
	if token != "test_encoded_token" {
		t.Errorf("Expected 'test_encoded_token', got '%s'", token)
	}
}

func TestSetEncodedToken(t *testing.T) {
	cli := NewClient()

	cli.SetEncodedToken("new_encoded_token")
	if cli.GetEncodedToken() != "new_encoded_token" {
		t.Errorf("Expected 'new_encoded_token', got '%s'", cli.GetEncodedToken())
	}
}

func TestDeleteToTrash_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchTrash"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok || len(ids) != 2 {
			t.Error("Expected ids to contain 2 items")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.DeleteToTrash(context.Background(), []string{"file1", "file2"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestDeleteToTrash_EmptyIDs(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	_, err := cli.DeleteToTrash(context.Background(), []string{})
	if err == nil {
		t.Error("Expected error when ids are empty")
	}
}

func TestUntrash_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchUntrash"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok || len(ids) != 1 {
			t.Error("Expected ids to contain 1 item")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.Untrash(context.Background(), []string{"file_id"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestDeleteForever_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchDelete"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok || len(ids) != 1 {
			t.Error("Expected ids to contain 1 item")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.DeleteForever(context.Background(), []string{"file_id"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestOfflineDownload_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "test_download" {
			t.Error("Expected name to be 'test_download'")
		}

		urlObj, ok := body["url"].(map[string]interface{})
		if !ok {
			t.Error("Expected url to be an object")
		} else if urlObj["url"] != "magnet:xxx" {
			t.Error("Expected url.url to be 'magnet:xxx'")
		}

		response := map[string]interface{}{
			"id":     "task_123",
			"name":   "test_download",
			"status": "not_downloading",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.OfflineDownload(context.Background(), "magnet:xxx", "", "test_download")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result["id"] != "task_123" {
		t.Errorf("Expected id 'task_123', got '%v'", result["id"])
	}
}

func TestOfflineDownload_EmptyURL(t *testing.T) {
	cli := NewClient(WithAccessToken("test_token"))

	_, err := cli.OfflineDownload(context.Background(), "", "", "test")
	if err == nil {
		t.Error("Expected error when url is empty")
	}
}

func TestOfflineList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/tasks"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"tasks": []interface{}{
				map[string]interface{}{
					"task_id": "task_1",
					"status":  "done",
				},
			},
			"next_page_token": "next_token",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.OfflineList(context.Background(), 20, "", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestDeleteOfflineTasks_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/tasks"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		if r.URL.Query().Get("task_ids") != "task1,task2" {
			t.Error("Expected task_ids to be 'task1,task2'")
		}

		if r.URL.Query().Get("delete_files") != "false" {
			t.Error("Expected delete_files to be 'false'")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	err := cli.DeleteOfflineTasks(context.Background(), []string{"task1", "task2"}, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestGetTaskStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files/file_456"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"task_id": "task_123",
			"phase":   "PHASE_TYPE_NOT_FOUND",
			"file_id": "file_456",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	status, err := cli.GetTaskStatus(context.Background(), "task_123", "file_456")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status != enums.DownloadStatusNotFound {
		t.Errorf("Expected status 'not_found', got '%s'", status)
	}
}

func TestFileBatchStar_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchStar"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok || len(ids) != 1 {
			t.Error("Expected ids to contain 1 item")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	err := cli.FileBatchStar(context.Background(), []string{"file_id"}, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestFileBatchUnstar_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files:batchStar"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		ids, ok := body["ids"].([]interface{})
		if !ok || len(ids) != 1 {
			t.Error("Expected ids to contain 1 item")
		}

		star, ok := body["star"].(bool)
		if !ok || star != false {
			t.Error("Expected star to be false")
		}

		response := map[string]interface{}{
			"code": "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	err := cli.FileBatchUnstar(context.Background(), []string{"file_id"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestFileStarList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		if r.URL.Query().Get("thumbnail_size") != "SIZE_LARGE" {
			t.Error("Expected thumbnail_size to be 'SIZE_LARGE'")
		}

		response := map[string]interface{}{
			"files": []interface{}{
				map[string]interface{}{
					"id":   "file_1",
					"name": "starred_file",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.FileStarList(context.Background(), 20, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestFileRename_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/files/file_id"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "new_name" {
			t.Errorf("Expected name 'new_name', got '%v'", body["name"])
		}

		response := map[string]interface{}{
			"id":   "file_id",
			"name": "new_name",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	err := cli.FileRename(context.Background(), "file_id", "new_name")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestGetShareInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/share/v1/info"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		if r.URL.Query().Get("share_url") != "https://my.pikpak.com/share/share_123" {
			t.Error("Expected share_url to be 'https://my.pikpak.com/share/share_123'")
		}

		response := map[string]interface{}{
			"share_id":    "share_123",
			"share_token": "token_abc",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.GetShareInfo(context.Background(), "https://my.pikpak.com/share/share_123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestCreateShareLink_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/share"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["file_id"] != "file_123" {
			t.Errorf("Expected file_id 'file_123', got '%v'", body["file_id"])
		}

		if body["share_type"] != float64(2) {
			t.Errorf("Expected share_type 2, got '%v'", body["share_type"])
		}

		response := map[string]interface{}{
			"share_info": map[string]interface{}{
				"id":    "share_123",
				"url":   "https://my.pikpak.com/s/share_123",
				"token": "share_token_xyz",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.CreateShareLink(context.Background(), "file_123", 86400, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestGetShareDownloadURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/share/file_info"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		if r.URL.Query().Get("share_id") != "share_123" {
			t.Error("Expected share_id to be 'share_123'")
		}

		response := map[string]interface{}{
			"file_info": map[string]interface{}{
				"download_url":     "https://download.example.com/file",
				"web_content_link": "https://download.example.com/file",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	url, err := cli.GetShareDownloadURL(context.Background(), "https://my.pikpak.com/share/link/share_123", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if url != "https://download.example.com/file" {
		t.Errorf("Expected download URL, got '%s'", url)
	}
}

func TestRestore_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/share/v1/file/restore"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["share_id"] != "share_123" {
			t.Error("Expected share_id to be 'share_123'")
		}

		response := map[string]interface{}{
			"file_ids": []string{"restored_file_1"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.Restore(context.Background(), "share_123", "pass_token", []string{"file_1"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestGetQuotaInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/about"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"total_capacity": "107374182400",
			"used_capacity":  "53687091200",
			"trash_capacity": "1073741824",
			"is_vip":         true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.GetQuotaInfo(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

func TestEvents_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/drive/v1/events"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{
					"event_id": "event_1",
					"type":     "file_created",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cli := NewClient(WithBaseURL(server.URL), WithAccessToken("test_token"))

	result, err := cli.Events(context.Background(), 50, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}
