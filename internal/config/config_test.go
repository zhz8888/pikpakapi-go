package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configData := Config{
		Username:     "test@example.com",
		Password:     "testpassword",
		AccessToken:  "access_token_123",
		RefreshToken: "refresh_token_456",
		EncodedToken: "encoded_token_789",
		DeviceID:     "device_id_abc",
		CaptchaToken: "captcha_token_xyz",
		UserID:       "user_id_123",
	}

	data, _ := json.MarshalIndent(configData, "", "  ")
	os.WriteFile(configPath, data, 0644)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.Username != configData.Username {
		t.Errorf("Username = %q, want %q", cfg.Username, configData.Username)
	}
	if cfg.Password != configData.Password {
		t.Errorf("Password = %q, want %q", cfg.Password, configData.Password)
	}
	if cfg.AccessToken != configData.AccessToken {
		t.Errorf("AccessToken = %q, want %q", cfg.AccessToken, configData.AccessToken)
	}
	if cfg.RefreshToken != configData.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", cfg.RefreshToken, configData.RefreshToken)
	}
	if cfg.EncodedToken != configData.EncodedToken {
		t.Errorf("EncodedToken = %q, want %q", cfg.EncodedToken, configData.EncodedToken)
	}
	if cfg.DeviceID != configData.DeviceID {
		t.Errorf("DeviceID = %q, want %q", cfg.DeviceID, configData.DeviceID)
	}
	if cfg.CaptchaToken != configData.CaptchaToken {
		t.Errorf("CaptchaToken = %q, want %q", cfg.CaptchaToken, configData.CaptchaToken)
	}
	if cfg.UserID != configData.UserID {
		t.Errorf("UserID = %q, want %q", cfg.UserID, configData.UserID)
	}
}

func TestLoadConfig_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	os.Remove("config.json")
	os.Remove(".pikpakapi.json")
	os.Remove(filepath.Join(tmpDir, ".pikpakapi.json"))

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.Username != "" {
		t.Errorf("Username = %q, want empty string", cfg.Username)
	}
	if cfg.Password != "" {
		t.Errorf("Password = %q, want empty string", cfg.Password)
	}
	if cfg.AccessToken != "" {
		t.Errorf("AccessToken = %q, want empty string", cfg.AccessToken)
	}
	if cfg.RefreshToken != "" {
		t.Errorf("RefreshToken = %q, want empty string", cfg.RefreshToken)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	invalidJSON := []byte("{invalid json data")
	os.WriteFile(configPath, invalidJSON, 0644)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.Username != "" {
		t.Errorf("Username = %q, want empty string for invalid JSON", cfg.Username)
	}
}

func TestLoadConfig_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	partialData := map[string]interface{}{
		"username":     "partial@example.com",
		"access_token": "partial_token",
	}
	data, _ := json.MarshalIndent(partialData, "", "  ")
	os.WriteFile(configPath, data, 0644)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.Username != "partial@example.com" {
		t.Errorf("Username = %q, want %q", cfg.Username, "partial@example.com")
	}
	if cfg.AccessToken != "partial_token" {
		t.Errorf("AccessToken = %q, want %q", cfg.AccessToken, "partial_token")
	}
	if cfg.Password != "" {
		t.Errorf("Password = %q, want empty string", cfg.Password)
	}
}

func TestSaveConfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := &Config{
		Username:     "save_test@example.com",
		Password:     "save_password",
		AccessToken:  "save_access_token",
		RefreshToken: "save_refresh_token",
		EncodedToken: "save_encoded_token",
		DeviceID:     "save_device_id",
		CaptchaToken: "save_captcha_token",
		UserID:       "save_user_id",
	}

	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() returned unexpected error: %v", err)
	}

	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config file: %v", err)
	}

	var savedCfg Config
	if err := json.Unmarshal(data, &savedCfg); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if savedCfg.Username != cfg.Username {
		t.Errorf("Saved username = %q, want %q", savedCfg.Username, cfg.Username)
	}
	if savedCfg.Password != cfg.Password {
		t.Errorf("Saved password = %q, want %q", savedCfg.Password, cfg.Password)
	}
	if savedCfg.AccessToken != cfg.AccessToken {
		t.Errorf("Saved access_token = %q, want %q", savedCfg.AccessToken, cfg.AccessToken)
	}
}

func TestSaveConfig_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty_config.json")

	cfg := &Config{}

	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() with empty config returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config file: %v", err)
	}

	var savedCfg Config
	if err := json.Unmarshal(data, &savedCfg); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if savedCfg.Username != "" {
		t.Errorf("Saved username = %q, want empty string", savedCfg.Username)
	}
}

func TestSaveConfig_InvalidPath(t *testing.T) {
	cfg := &Config{
		Username:    "test@example.com",
		AccessToken: "test_token",
	}

	invalidPath := "/nonexistent/directory/config.json"

	err := SaveConfig(cfg, invalidPath)
	if err == nil {
		t.Error("SaveConfig() should return error for invalid path")
	}
}

func TestSaveConfig_TruncatesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	oldData := []byte(`{"username": "old@example.com", "access_token": "old_token"}`)
	os.WriteFile(configPath, oldData, 0644)

	newCfg := &Config{
		Username:     "new@example.com",
		AccessToken:  "new_token",
		RefreshToken: "new_refresh_token",
	}

	err := SaveConfig(newCfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var savedCfg Config
	if err := json.Unmarshal(data, &savedCfg); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if savedCfg.Username != "new@example.com" {
		t.Errorf("Username = %q, want %q", savedCfg.Username, "new@example.com")
	}
	if savedCfg.RefreshToken != "new_refresh_token" {
		t.Errorf("RefreshToken = %q, want %q", savedCfg.RefreshToken, "new_refresh_token")
	}
}

func TestLoadConfig_JSONFieldTags(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	jsonData := `{
		"username": "tag_test@example.com",
		"password": "tag_password",
		"access_token": "tag_access_token",
		"refresh_token": "tag_refresh_token",
		"encoded_token": "tag_encoded_token",
		"device_id": "tag_device_id",
		"captcha_token": "tag_captcha_token",
		"user_id": "tag_user_id"
	}`
	os.WriteFile(configPath, []byte(jsonData), 0644)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.Username != "tag_test@example.com" {
		t.Errorf("Username = %q, want %q", cfg.Username, "tag_test@example.com")
	}
	if cfg.AccessToken != "tag_access_token" {
		t.Errorf("AccessToken = %q, want %q", cfg.AccessToken, "tag_access_token")
	}
	if cfg.UserID != "tag_user_id" {
		t.Errorf("UserID = %q, want %q", cfg.UserID, "tag_user_id")
	}
}

func TestLoadConfig_UnicodeInConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	unicodeData := `{
		"username": "用户@example.com",
		"password": "密码123",
		"device_id": "设备标识"
	}`
	os.WriteFile(configPath, []byte(unicodeData), 0644)

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	if cfg.Username != "用户@example.com" {
		t.Errorf("Username = %q, want %q", cfg.Username, "用户@example.com")
	}
	if cfg.Password != "密码123" {
		t.Errorf("Password = %q, want %q", cfg.Password, "密码123")
	}
	if cfg.DeviceID != "设备标识" {
		t.Errorf("DeviceID = %q, want %q", cfg.DeviceID, "设备标识")
	}
}

func TestSaveConfig_PreservesJSONFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "formatted_config.json")

	cfg := &Config{
		Username: "format@test.com",
	}

	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	content := string(data)

	if len(content) == 0 {
		t.Error("Saved config file is empty")
	}

	if content[:2] != "{\n" {
		t.Errorf("Config should start with opening brace and newline, got: %q", content[:3])
	}
}
