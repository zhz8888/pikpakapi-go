package token

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type Data struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func Encode(accessToken, refreshToken string) (string, error) {
	data := Data{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token data: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return encoded, nil
}

func Decode(encodedToken string) (*Data, error) {
	jsonData, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	var data Data
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	if data.AccessToken == "" || data.RefreshToken == "" {
		return nil, fmt.Errorf("invalid token: missing access_token or refresh_token")
	}

	return &data, nil
}
