package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type Config struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	EncodedToken string `json:"encoded_token"`
	DeviceID     string `json:"device_id"`
	CaptchaToken string `json:"captcha_token"`
	UserID       string `json:"user_id"`
}

var (
	ErrEmptyUsername   = errors.New("username cannot be empty")
	ErrEmptyPassword   = errors.New("password cannot be empty")
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPhone    = errors.New("invalid phone format")
	ErrInvalidUsername = errors.New("invalid username format")
)

type ConfigBuilder struct {
	config Config
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: Config{},
	}
}

func (b *ConfigBuilder) WithUsername(username string) *ConfigBuilder {
	b.config.Username = username
	return b
}

func (b *ConfigBuilder) WithPassword(password string) *ConfigBuilder {
	b.config.Password = password
	return b
}

func (b *ConfigBuilder) WithAccessToken(accessToken string) *ConfigBuilder {
	b.config.AccessToken = accessToken
	return b
}

func (b *ConfigBuilder) WithRefreshToken(refreshToken string) *ConfigBuilder {
	b.config.RefreshToken = refreshToken
	return b
}

func (b *ConfigBuilder) WithEncodedToken(encodedToken string) *ConfigBuilder {
	b.config.EncodedToken = encodedToken
	return b
}

func (b *ConfigBuilder) WithDeviceID(deviceID string) *ConfigBuilder {
	b.config.DeviceID = deviceID
	return b
}

func (b *ConfigBuilder) WithCaptchaToken(captchaToken string) *ConfigBuilder {
	b.config.CaptchaToken = captchaToken
	return b
}

func (b *ConfigBuilder) WithUserID(userID string) *ConfigBuilder {
	b.config.UserID = userID
	return b
}

func (b *ConfigBuilder) Build() (*Config, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	return &b.config, nil
}

func (b *ConfigBuilder) validate() error {
	if b.config.Username == "" {
		return ErrEmptyUsername
	}
	if b.config.Password == "" {
		return ErrEmptyPassword
	}
	return nil
}

func ValidateConfig(cfg *Config) error {
	if cfg.Username == "" {
		return ErrEmptyUsername
	}
	if cfg.Password == "" {
		return ErrEmptyPassword
	}
	return nil
}

func (b *ConfigBuilder) WithConfig(cfg *Config) *ConfigBuilder {
	if cfg != nil {
		b.config = *cfg
	}
	return b
}

type ValidationError struct {
	Field   string
	Message error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %s", e.Message.Error())
}

func (e *ValidationError) Unwrap() error {
	return e.Message
}

func (b *ConfigBuilder) ValidateUsername() error {
	username := b.config.Username
	if username == "" {
		return &ValidationError{Field: "Username", Message: ErrEmptyUsername}
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)

	if emailRegex.MatchString(username) {
		return nil
	}
	if phoneRegex.MatchString(username) {
		return nil
	}

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{2,50}$`)
	if usernameRegex.MatchString(username) {
		return nil
	}

	return &ValidationError{Field: "Username", Message: ErrInvalidUsername}
}

func (b *ConfigBuilder) ValidatePassword() error {
	if len(b.config.Password) < 6 {
		return &ValidationError{Field: "Password", Message: errors.New("password must be at least 6 characters")}
	}
	return nil
}

func (b *ConfigBuilder) ValidateDeviceID() error {
	if b.config.DeviceID != "" && len(b.config.DeviceID) < 5 {
		return &ValidationError{Field: "DeviceID", Message: errors.New("device ID must be at least 5 characters")}
	}
	return nil
}

func (b *ConfigBuilder) Validate() []error {
	var errs []error

	if err := b.ValidateUsername(); err != nil {
		errs = append(errs, err)
	}
	if err := b.ValidatePassword(); err != nil {
		errs = append(errs, err)
	}
	if err := b.ValidateDeviceID(); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func LoadConfig() (*Config, error) {
	configPaths := []string{
		"config.json",
		".pikpakapi.json",
		filepath.Join(os.Getenv("HOME"), ".pikpakapi.json"),
	}

	var cfg *Config
	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var c Config
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}

		cfg = &c
		break
	}

	if cfg == nil {
		cfg = &Config{}
	}

	return cfg, nil
}

func SaveConfig(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
