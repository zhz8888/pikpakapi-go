package auth

import (
	"context"
	"fmt"
	"regexp"

	"github.com/zhz8888/pikpakapi-go/internal/constants"
	"github.com/zhz8888/pikpakapi-go/internal/exception"
	"github.com/zhz8888/pikpakapi-go/internal/signer"
	"github.com/zhz8888/pikpakapi-go/internal/token"
)

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Auth struct {
	username     string
	password     string
	encodedToken string
	accessToken  string
	refreshToken string
	userID       string
	deviceID     string
	captchaToken string
	httpClient   HTTPClient
	baseURL      string
}

type HTTPClient interface {
	PostJSON(ctx context.Context, url string, data interface{}) (map[string]interface{}, error)
	PostForm(ctx context.Context, url string, data map[string]string) (map[string]interface{}, error)
}

type AuthOption func(*Auth)

func WithUsername(username string) AuthOption {
	return func(a *Auth) {
		a.username = username
	}
}

func WithPassword(password string) AuthOption {
	return func(a *Auth) {
		a.password = password
	}
}

func WithDeviceID(deviceID string) AuthOption {
	return func(a *Auth) {
		a.deviceID = deviceID
	}
}

func WithAccessToken(token string) AuthOption {
	return func(a *Auth) {
		a.accessToken = token
	}
}

func WithRefreshToken(token string) AuthOption {
	return func(a *Auth) {
		a.refreshToken = token
	}
}

func WithEncodedToken(token string) AuthOption {
	return func(a *Auth) {
		a.encodedToken = token
	}
}

func WithBaseURL(baseURL string) AuthOption {
	return func(a *Auth) {
		a.baseURL = baseURL
	}
}

func NewAuth(opts ...AuthOption) *Auth {
	auth := &Auth{
		httpClient:   nil,
		baseURL:      "",
		encodedToken: "",
		accessToken:  "",
		refreshToken: "",
		userID:       "",
		deviceID:     "",
		captchaToken: "",
	}

	for _, opt := range opts {
		opt(auth)
	}

	return auth
}

func (a *Auth) SetHTTPClient(client HTTPClient) {
	a.httpClient = client
}

func (a *Auth) GetUserID() string {
	return a.userID
}

func (a *Auth) SetUserID(userID string) {
	a.userID = userID
}

func (a *Auth) GetCaptchaToken() string {
	return a.captchaToken
}

func (a *Auth) SetCaptchaToken(token string) {
	a.captchaToken = token
}

func (a *Auth) GetDeviceID() string {
	return a.deviceID
}

func (a *Auth) WithDeviceID(deviceID string) {
	a.deviceID = deviceID
}

func (a *Auth) GetAccessToken() string {
	return a.accessToken
}

func (a *Auth) SetAccessToken(token string) {
	a.accessToken = token
}

func (a *Auth) GetRefreshToken() string {
	return a.refreshToken
}

func (a *Auth) SetRefreshToken(token string) {
	a.refreshToken = token
}

func (a *Auth) GetEncodedToken() string {
	return a.encodedToken
}

func (a *Auth) SetEncodedToken(token string) {
	a.encodedToken = token
}

func (a *Auth) DecodeToken() error {
	if a.encodedToken == "" {
		return exception.ErrInvalidEncodedToken
	}

	data, err := token.Decode(a.encodedToken)
	if err != nil {
		return exception.NewPikpakExceptionWithError(exception.ErrCodeInvalidEncodedToken, err)
	}

	a.accessToken = data.AccessToken
	a.refreshToken = data.RefreshToken
	return nil
}

func (a *Auth) EncodeToken() error {
	encoded, err := token.Encode(a.accessToken, a.refreshToken)
	if err != nil {
		return exception.NewPikpakExceptionWithError(exception.ErrCodeInvalidEncodedToken, err)
	}
	a.encodedToken = encoded
	return nil
}

func (a *Auth) CaptchaInit(ctx context.Context, action string, meta map[string]interface{}) (map[string]interface{}, error) {
	baseURL := a.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.UserHost
	}
	URL := baseURL + "/v1/shield/captcha/init"

	if meta == nil {
		timestamp := fmt.Sprintf("%d", signer.GetTimestamp())
		meta = map[string]interface{}{
			"captcha_sign":   signer.CaptchaSign(a.deviceID, timestamp),
			"client_version": signer.ClientVersion,
			"package_name":   signer.PackageName,
			"user_id":        a.userID,
			"timestamp":      timestamp,
		}
	}

	params := map[string]interface{}{
		"client_id": constants.ClientID,
		"action":    action,
		"device_id": a.deviceID,
		"meta":      meta,
	}

	return a.httpClient.PostJSON(ctx, URL, params)
}

func (a *Auth) Login(ctx context.Context) error {
	if a.username == "" || a.password == "" {
		return exception.ErrUsernamePasswordRequired
	}

	baseURL := a.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.UserHost
	}
	loginURL := baseURL + "/v1/auth/signin"

	metas := make(map[string]interface{})
	emailRegex := regexp.MustCompile(`^[\w.-]+@[\w.-]+\.\w+$`)
	phoneRegex := regexp.MustCompile(`^\d{11,18}$`)

	if emailRegex.MatchString(a.username) {
		metas["email"] = a.username
	} else if phoneRegex.MatchString(a.username) {
		metas["phone_number"] = a.username
	} else {
		metas["username"] = a.username
	}

	result, err := a.CaptchaInit(ctx, "POST:"+loginURL, metas)
	if err != nil {
		return err
	}

	captchaToken, ok := result["captcha_token"].(string)
	if !ok || captchaToken == "" {
		return exception.ErrCaptchaTokenFailed
	}

	a.captchaToken = captchaToken

	loginData := map[string]string{
		"client_id":     constants.ClientID,
		"client_secret": constants.ClientSecret,
		"password":      a.password,
		"username":      a.username,
		"captcha_token": captchaToken,
	}

	userInfo, err := a.httpClient.PostForm(ctx, loginURL, loginData)
	if err != nil {
		return err
	}

	if accessToken, ok := userInfo["access_token"].(string); ok {
		a.accessToken = accessToken
	} else {
		return exception.NewPikpakExceptionWithMessage(exception.ErrCodeUnknownError, "login failed: no access_token")
	}

	if refreshToken, ok := userInfo["refresh_token"].(string); ok {
		a.refreshToken = refreshToken
	}

	if sub, ok := userInfo["sub"].(string); ok {
		a.userID = sub
	}

	if err := a.EncodeToken(); err != nil {
		return err
	}

	return nil
}

func (a *Auth) RefreshAccessToken(ctx context.Context) error {
	baseURL := a.baseURL
	if baseURL == "" {
		baseURL = "https://" + constants.UserHost
	}
	refreshURL := baseURL + "/v1/auth/token"

	refreshData := map[string]string{
		"client_id":     constants.ClientID,
		"refresh_token": a.refreshToken,
		"grant_type":    "refresh_token",
	}

	userInfo, err := a.httpClient.PostForm(ctx, refreshURL, refreshData)
	if err != nil {
		return err
	}

	if accessToken, ok := userInfo["access_token"].(string); ok {
		a.accessToken = accessToken
	} else {
		return exception.NewPikpakExceptionWithMessage(exception.ErrCodeUnknownError, "refresh failed: no access_token")
	}

	if refreshToken, ok := userInfo["refresh_token"].(string); ok {
		a.refreshToken = refreshToken
	}

	if sub, ok := userInfo["sub"].(string); ok {
		a.userID = sub
	}

	if err := a.EncodeToken(); err != nil {
		return err
	}

	return nil
}
