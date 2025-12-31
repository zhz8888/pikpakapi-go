package client

import (
	"context"
	"testing"
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

	if cli.deviceID != "custom_device_id" {
		t.Errorf("Expected deviceID 'custom_device_id', got '%s'", cli.deviceID)
	}
}

func TestNewClient_DefaultDeviceID(t *testing.T) {
	cli := NewClient(
		WithUsername("test_user"),
		WithPassword("test_pass"),
	)

	if cli.deviceID == "" {
		t.Error("Expected deviceID to be generated")
	}
}

func TestGetUserInfo(t *testing.T) {
	cli := NewClient(
		WithUsername("test_user"),
		WithPassword("test_pass"),
	)

	cli.accessToken = "test_access"
	cli.refreshToken = "test_refresh"
	cli.userID = "test_user_id"
	cli.encodedToken = "test_encoded"

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
