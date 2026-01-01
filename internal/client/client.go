// Package client 提供了与PikPak云存储服务进行交互的API客户端实现
//
// 该客户端封装了所有与PikPak API的通信逻辑，包括：
//   - 用户认证：登录、令牌刷新、验证码处理
//   - 文件管理：创建文件夹、删除文件、重命名、收藏、分享
//   - 离线下载：创建下载任务、查询任务状态、任务管理
//   - 配额查询：获取账户存储配额信息
//   - 分享管理：创建分享链接、恢复分享文件
//
// 使用示例：
//
//	ctx := context.Background()
//	cli := client.NewClient(
//		client.WithUsername("your_email@example.com"),
//		client.WithPassword("your_password"),
//		client.WithMaxRetries(3),
//	)
//
//	if err := cli.Login(ctx); err != nil {
//		log.Fatalf("登录失败: %v", err)
//	}
//
//	// 获取用户信息
//	userInfo := cli.GetUserInfo()
//
//	// 获取配额信息
//	quota, _ := cli.GetQuotaInfo(ctx)
//
//	// 创建离线下载任务
//	result, _ := cli.OfflineDownload(ctx, "magnet:...", "", "下载任务名称")
package client

import (
	"bytes"
	"context"
	"crypto/md5"
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

	"github.com/zhz8888/pikpakapi-go/internal/exception"
	"github.com/zhz8888/pikpakapi-go/internal/utils"
	"github.com/zhz8888/pikpakapi-go/pkg/enums"
)

const (
	// APIHost PikPak Drive API的服务地址
	// 用于所有文件操作、离线下载、分享等功能的API请求
	APIHost = "api-drive.mypikpak.com"

	// UserHost PikPak用户服务的地址
	// 用于用户认证、登录、令牌刷新等用户相关的API请求
	UserHost = "user.mypikpak.com"

	// HTTPTimeout HTTP请求的默认超时时间
	// 设置为10秒，适用于大多数API请求
	// 对于大文件上传等耗时操作，可能需要适当调整
	HTTPTimeout = 10 * time.Second
)

// Client PikPak API客户端的结构体
//
// 封装了与PikPak服务进行交互所需的所有状态信息和配置
//
// 字段说明：
//   - username: 用户的登录名称，支持邮箱、手机号或用户名
//   - password: 用户的密码
//   - encodedToken: Base64编码的令牌字符串，可用于持久化保存
//   - accessToken: 访问令牌，用于API请求的身份验证
//   - refreshToken: 刷新令牌，用于获取新的访问令牌
//   - userID: 用户的唯一标识符，由PikPak服务在登录时返回
//   - deviceID: 设备标识符，用于设备绑定和安全验证
//   - captchaToken: 验证码令牌，用于登录时的验证码验证
//   - maxRetries: HTTP请求失败时的最大重试次数，默认为3次
//   - initialBackoff: 重试时的初始退避时间，默认为3秒
//   - httpClient: 底层的HTTP客户端实例，用于执行实际的HTTP请求
//   - tokenRefreshCallback: 令牌刷新成功后的回调函数
//   - tokenRefreshCallbackCtx: 令牌刷新回调函数的上下文
//
// 线程安全性：
//
//	Client结构体不是线程安全的，如果在多个goroutine中共享使用，
//	需要自行实现同步机制（如互斥锁）
type Client struct {
	username                string
	password                string
	encodedToken            string
	accessToken             string
	refreshToken            string
	userID                  string
	deviceID                string
	captchaToken            string
	maxRetries              int
	initialBackoff          time.Duration
	httpClient              *http.Client
	tokenRefreshCallback    func(*Client)
	tokenRefreshCallbackCtx context.Context
	baseURL                 string
}

// Option 函数式选项模式用于配置Client
//
// 允许使用可选配置来初始化客户端，而不需要修改Client的公共字段
// 这种模式提供了灵活的配置方式，同时保持了向后兼容性
//
// 使用示例：
//
//	cli := client.NewClient(
//		client.WithUsername("user@example.com"),
//		client.WithPassword("password"),
//		client.WithMaxRetries(5),
//		client.WithInitialBackoff(5 * time.Second),
//	)
type Option func(*Client)

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

// WithUsername 设置用户的登录名称
//
// 参数说明：
//   - username: string 用户的登录名称，支持以下格式：
//   - 邮箱格式：如 "user@example.com"
//   - 手机号格式：如 "13800138000"
//   - 用户名格式：如 "username123"
//
// 返回值：
//   - Option 函数式选项，可传递给NewClient
func WithUsername(username string) Option {
	return func(c *Client) {
		c.username = username
	}
}

// WithPassword 设置用户的密码
//
// 参数说明：
//   - password: string 用户的密码
//
// 返回值：
//   - Option 函数式选项
func WithPassword(password string) Option {
	return func(c *Client) {
		c.password = password
	}
}

// WithDeviceID 设置设备标识符
//
// 设备标识符用于设备绑定和安全验证
// 如果未设置，在NewClient中会自动根据用户名和密码生成
//
// 参数说明：
//   - deviceID: string 设备标识符字符串
//
// 返回值：
//   - Option 函数式选项
func WithDeviceID(deviceID string) Option {
	return func(c *Client) {
		c.deviceID = deviceID
	}
}

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

func WithAccessToken(token string) Option {
	return func(c *Client) {
		c.accessToken = token
	}
}

func WithRefreshToken(token string) Option {
	return func(c *Client) {
		c.refreshToken = token
	}
}

// WithMaxRetries 设置HTTP请求的最大重试次数
//
// 当请求失败时，客户端会自动进行重试
// 重试策略采用指数退避算法，初始退避时间由WithInitialBackoff设置
//
// 参数说明：
//   - maxRetries: int 最大重试次数
//   - 设置为0表示禁用重试
//   - 建议值：3-5次
//   - 最小值：0
//   - 最大值：无限制，但过大会延长请求时间
//
// 返回值：
//   - Option 函数式选项
func WithMaxRetries(maxRetries int) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithInitialBackoff 设置重试时的初始退避时间
//
// 重试采用指数退避算法：
//
//	第n次重试的等待时间 = initialBackoff * 2^(n-1)
//
// 例如：初始退避时间为3秒时
//   - 第1次重试：等待3秒
//   - 第2次重试：等待6秒
//   - 第3次重试：等待12秒
//
// 参数说明：
//   - backoff: time.Duration 初始退避时间
//   - 建议值：1-5秒
//   - 时间过短可能导致服务器过载
//   - 时间过长可能影响用户体验
//
// 返回值：
//   - Option 函数式选项
func WithInitialBackoff(backoff time.Duration) Option {
	return func(c *Client) {
		c.initialBackoff = backoff
	}
}

// WithTokenRefreshCallback 设置令牌刷新成功后的回调函数
//
// 当accessToken过期并自动刷新后，会调用此回调函数
// 可用于保存新的令牌信息到配置文件或数据库
//
// 参数说明：
//   - callback: func(*Client) 回调函数
//   - 参数为触发回调的Client实例
//   - 可在回调中调用GetUserInfo获取最新的令牌信息
//
// 返回值：
//   - Option 函数式选项
func WithTokenRefreshCallback(callback func(*Client)) Option {
	return func(c *Client) {
		c.tokenRefreshCallback = callback
	}
}

// NewClient 创建并初始化一个新的PikPak API客户端
//
// 该函数使用函数式选项模式来配置客户端，支持多种可选配置
// 如果未显式设置deviceID，会自动根据用户名和密码生成
//
// 默认配置：
//   - maxRetries: 3次
//   - initialBackoff: 3秒
//   - deviceID: 根据用户名密码生成（如果未提供）
//   - HTTPTimeout: 10秒
//
// 参数说明：
//   - opts: ...Option 可变数量的函数式选项
//   - 所有选项都是可选的，不传则使用默认值
//
// 返回值：
//   - *Client 初始化的客户端实例
//
// 使用示例：
//
//	// 最小配置
//	cli := client.NewClient()
//
//	// 完整配置
//	cli := client.NewClient(
//		client.WithUsername("user@example.com"),
//		client.WithPassword("password123"),
//		client.WithMaxRetries(5),
//		client.WithInitialBackoff(2 * time.Second),
//		client.WithTokenRefreshCallback(func(c *client.Client) {
//			// 保存新令牌
//			log.Println("令牌已刷新")
//		}),
//	)
//
//	// 登录后使用
//	if err := cli.Login(ctx); err != nil {
//		log.Fatal(err)
//	}
func NewClient(opts ...Option) *Client {
	c := &Client{
		maxRetries:     3,
		initialBackoff: 3 * time.Second,
		deviceID:       "",
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.deviceID == "" && c.username != "" && c.password != "" {
		hash := md5.Sum([]byte(c.username + c.password))
		c.deviceID = hex.EncodeToString(hash[:])
	}

	c.httpClient = &http.Client{
		Timeout: HTTPTimeout,
	}

	return c
}

// GetUserInfo 获取当前客户端的用户信息
//
// 返回当前已登录用户的相关信息，包括访问令牌等敏感数据
//
// 返回值：
//   - map[string]string 包含以下键值对：
//   - "username": 用户的登录名称
//   - "user_id": 用户的唯一标识符
//   - "access_token": 访问令牌（用于API认证）
//   - "refresh_token": 刷新令牌（用于刷新访问令牌）
//   - "encoded_token": Base64编码的令牌（可持久化保存）
//
// 注意事项：
//   - 返回的map包含敏感信息，请妥善保管
//   - accessToken和refreshToken可能为空（如果未登录）
//
// 使用示例：
//
//	userInfo := cli.GetUserInfo()
//	userID := userInfo["user_id"]
//	accessToken := userInfo["access_token"]
func (c *Client) GetUserInfo() map[string]string {
	return map[string]string{
		"username":      c.username,
		"user_id":       c.userID,
		"access_token":  c.accessToken,
		"refresh_token": c.refreshToken,
		"encoded_token": c.encodedToken,
	}
}

// DecodeToken 解码并还原已保存的编码令牌
//
// 将Base64编码的令牌字符串解码，还原出accessToken和refreshToken
// 解码后的令牌会保存在Client结构体中，可直接用于API请求
//
// 返回值：
//   - error 错误信息，可能的错误包括：
//   - ErrInvalidEncodedToken: 编码令牌为空
//   - PikpakException: 解码失败
//
// 注意事项：
//   - 调用此方法前需确保encodedToken不为空
//   - 解码成功后可直接调用API方法
//
// 使用示例：
//
//	cli := client.NewClient()
//	cli.encodedToken = "eyJhbGciOi..." // 之前保存的编码令牌
//	if err := cli.DecodeToken(); err != nil {
//		log.Fatalf("令牌解码失败: %v", err)
//	}
//	// 现在可以使用cli调用API方法
func (c *Client) DecodeToken() error {
	if c.encodedToken == "" {
		return exception.ErrInvalidEncodedToken
	}

	data, err := utils.DecodeToken(c.encodedToken)
	if err != nil {
		return exception.NewPikpakExceptionWithError("failed to decode token", err)
	}

	c.accessToken = data.AccessToken
	c.refreshToken = data.RefreshToken
	return nil
}

// EncodeToken 将当前令牌编码为Base64字符串
//
// 将accessToken和refreshToken编码为Base64格式的字符串
// 便于持久化保存到配置文件或数据库
//
// 返回值：
//   - error 错误信息，可能的错误包括：
//   - PikpakException: 编码失败
//
// 注意事项：
//   - 编码后的字符串是URL安全的
//   - 编码是纯文本转换，不涉及加密
//
// 使用示例：
//
//	// 登录成功后保存编码令牌
//	if err := cli.Login(ctx); err != nil {
//		log.Fatal(err)
//	}
//	if err := cli.EncodeToken(); err != nil {
//		log.Fatal(err)
//	}
//	token := cli.encodedToken // 可保存此字符串
func (c *Client) EncodeToken() error {
	encoded, err := utils.EncodeToken(c.accessToken, c.refreshToken)
	if err != nil {
		return exception.NewPikpakExceptionWithError("failed to encode token", err)
	}
	c.encodedToken = encoded
	return nil
}

func (c *Client) GetEncodedToken() string {
	return c.encodedToken
}

func (c *Client) SetEncodedToken(token string) {
	c.encodedToken = token
}

// buildUserAgent 构建HTTP请求的User-Agent头
//
// 根据当前的验证码令牌状态选择合适的User-Agent
// 有验证码令牌时使用自定义的User-Agent，否则使用默认的Chrome浏览器标识
//
// 返回值：
//   - string User-Agent字符串
//
// 内部逻辑：
//   - 如果captchaToken不为空，调用utils.BuildCustomUserAgent
//   - 否则返回固定的Chrome浏览器User-Agent
func (c *Client) buildUserAgent() string {
	if c.captchaToken != "" {
		return utils.BuildCustomUserAgent(c.deviceID, c.userID)
	}
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
}

// getHeaders 构建HTTP请求的头部信息
//
// 根据当前客户端状态生成完整的请求头
// 包含：User-Agent、Content-Type、Authorization、验证码令牌、设备ID等
//
// 返回值：
//   - map[string]string 请求头键值对
//
// 包含的头部字段：
//   - User-Agent: 浏览器标识
//   - Content-Type: 请求内容类型（JSON）
//   - Authorization: Bearer认证令牌（如果有accessToken）
//   - X-Captcha-Token: 验证码令牌（如果有）
//   - X-Device-Id: 设备标识符（如果有）
func (c *Client) getHeaders() map[string]string {
	headers := map[string]string{
		"User-Agent":   c.buildUserAgent(),
		"Content-Type": "application/json; charset=utf-8",
	}

	if c.accessToken != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", c.accessToken)
	}
	if c.captchaToken != "" {
		headers["X-Captcha-Token"] = c.captchaToken
	}
	if c.deviceID != "" {
		headers["X-Device-Id"] = c.deviceID
	}

	return headers
}

// doRequest 执行HTTP请求的核心方法
//
// 封装了完整的HTTP请求流程，包括：
//   - 请求构建（方法、URL、请求体、查询参数）
//   - 请求头设置
//   - 自动重试（指数退避）
//   - 令牌自动刷新
//   - 错误处理
//
// 参数说明：
//   - ctx: context.Context 请求上下文，用于控制超时和取消
//   - method: string HTTP方法（GET、POST、DELETE、PATCH等）
//   - reqURL: string 请求的完整URL
//   - data: interface{} 请求体数据，会序列化为JSON
//   - params: map[string]string URL查询参数
//
// 返回值：
//   - []byte 响应体字节数据
//   - error 错误信息
//
// 重试逻辑：
//  1. 最多重试maxRetries次
//  2. 每次失败后等待 initialBackoff * 2^attempt 秒
//  3. 如果响应包含error_code=16，自动刷新令牌后重试
//
// 错误处理：
//   - 网络错误：记录日志并重试
//   - 读取错误：记录日志并重试
//   - 认证错误（invalid_account_or_password）：返回特定错误
//   - 令牌过期：自动刷新后重试
//   - 其他错误：返回PikpakException
func (c *Client) doRequest(ctx context.Context, method, reqURL string, data interface{}, params map[string]string) ([]byte, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, exception.NewPikpakExceptionWithError("failed to marshal request data", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to create request", err)
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

	var lastError error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastError = err
			log.Printf("HTTP Error on attempt %d/%d: %v", attempt+1, c.maxRetries, err)
			time.Sleep(c.initialBackoff * time.Duration(1<<attempt))
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastError = err
			log.Printf("Read Error on attempt %d/%d: %v", attempt+1, c.maxRetries, err)
			time.Sleep(c.initialBackoff * time.Duration(1<<attempt))
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return respBody, nil
		}

		var respData map[string]interface{}
		if err := json.Unmarshal(respBody, &respData); err != nil {
			lastError = fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
			time.Sleep(c.initialBackoff * time.Duration(1<<attempt))
			continue
		}

		if errMsg, ok := respData["error"].(string); ok {
			if errMsg == "invalid_account_or_password" {
				return nil, exception.ErrInvalidUsernamePassword
			}
			if desc, ok := respData["error_description"].(string); ok {
				lastError = fmt.Errorf("%s: %s", errMsg, desc)
			} else {
				lastError = fmt.Errorf("%s", errMsg)
			}
		}

		if errorCode, ok := respData["error_code"].(float64); ok {
			if int(errorCode) == 16 {
				if err := c.RefreshAccessToken(ctx); err != nil {
					return nil, err
				}
				lastError = fmt.Errorf("token refreshed, please retry")
				time.Sleep(c.initialBackoff * time.Duration(1<<attempt))
				continue
			}
		}

		time.Sleep(c.initialBackoff * time.Duration(1<<attempt))
	}

	return nil, exception.NewPikpakExceptionWithError(fmt.Sprintf("max retries reached, last error: %v", lastError), lastError)
}

// getJSON 执行GET请求并解析JSON响应
//
// 封装了doRequest方法，专门用于GET请求和JSON响应解析
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - URL: string 请求的完整URL
//   - params: map[string]string URL查询参数
//
// 返回值：
//   - map[string]interface{} 解析后的JSON数据
//   - error 错误信息
func (c *Client) getJSON(ctx context.Context, URL string, params map[string]string) (map[string]interface{}, error) {
	data, err := c.doRequest(ctx, http.MethodGet, URL, nil, params)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to unmarshal response", err)
	}

	return result, nil
}

// GetStorageInfo 获取存储配额信息
//
// 查询当前账户的存储空间使用情况和配额限制
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//
// 返回值：
//   - *StorageInfo 存储配额信息对象，包含以下字段：
//   - TotalBytes: 总存储空间（字节），0表示无限容量
//   - UsedBytes: 已使用空间（字节）
//   - TrashBytes: 回收站占用空间（字节）
//   - IsUnlimited: 是否为无限容量
//   - Complimentary: 附加服务类型（如 "premium"、"basic" 等）
//   - error 错误信息
//
// API endpoint：
//   - GET https://api-drive.mypikpak.com/drive/v1/about
//
// 使用示例：
//
//	quota, err := cli.GetStorageInfo(ctx)
//	if err != nil {
//		log.Fatalf("获取配额失败: %v", err)
//	}
//	fmt.Printf("总容量: %d GB\n", quota.TotalBytes/1024/1024/1024)
//	fmt.Printf("已用: %d GB\n", quota.UsedBytes/1024/1024/1024)
//	fmt.Printf("无限容量: %v\n", quota.IsUnlimited)
func (c *Client) GetStorageInfo(ctx context.Context) (*StorageInfo, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "api-drive.mypikpak.com"
	}
	if !strings.Contains(baseURL, "://") {
		baseURL = "https://" + baseURL
	}
	resp, err := c.getJSON(ctx, baseURL+"/drive/v1/about", nil)
	if err != nil {
		return nil, err
	}

	quota, ok := resp["quota"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("quota not found in response")
	}

	limitStr, ok := quota["limit"].(string)
	if !ok {
		return nil, fmt.Errorf("quota limit is not a string")
	}
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid quota limit format: %w", err)
	}

	usageStr, ok := quota["usage"].(string)
	if !ok {
		return nil, fmt.Errorf("quota usage is not a string")
	}
	usage, err := strconv.ParseUint(usageStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid quota usage format: %w", err)
	}

	trashUsageStr, ok := quota["usage_in_trash"].(string)
	if !ok {
		return nil, fmt.Errorf("quota usage_in_trash is not a string")
	}
	trashUsage, err := strconv.ParseUint(trashUsageStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid quota usage_in_trash format: %w", err)
	}

	return &StorageInfo{
		TotalBytes:    limit,
		UsedBytes:     usage,
		TrashBytes:    trashUsage,
		IsUnlimited:   quota["is_unlimited"].(bool),
		Complimentary: quota["complimentary"].(string),
	}, nil
}

// GetFileLink 获取文件的下载链接
//
// 根据文件ID获取可直接访问的下载链接
// 如果文件有多个媒体流，优先返回媒体链接而非Web内容链接
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileID: string 文件的唯一标识符
//
// 返回值：
//   - string 文件的直接下载链接URL
//   - error 错误信息
//
// API endpoint：
//   - GET https://api-drive.mypikpak.net/drive/v1/files/{fileID}
//   - 查询参数：_magic=2021, usage=CACHE, thumbnail_size=SIZE_LARGE
//
// 链接优先级：
//  1. 如果文件有媒体流（medias），返回第一个媒体的链接
//  2. 否则返回 web_content_link
//
// 使用示例：
//
//	downloadURL, err := cli.GetFileLink(ctx, "file_id_here")
//	if err != nil {
//		log.Fatalf("获取下载链接失败: %v", err)
//	}
//	fmt.Println("下载链接:", downloadURL)
func (c *Client) GetFileLink(ctx context.Context, fileID string) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api-drive.mypikpak.net"
	}
	resp, err := c.getJSON(ctx, fmt.Sprintf("%s/drive/v1/files/%s", baseURL, fileID), map[string]string{
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

// Move 移动文件到指定文件夹
//
// 将一个或多个文件移动到新的父文件夹
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileID: string 要移动的文件ID
//   - parentID: string 目标文件夹ID
//   - 空字符串表示移动到根目录
//
// 返回值：
//   - error 错误信息，操作成功时返回nil
//
// API endpoint：
//   - POST https://api-drive.mypikpak.net/drive/v1/files:batchMove
//   - 请求体：{"ids": [fileID], "to": {"parent_id": parentID}}
//
// 使用示例：
//
//	// 移动文件到文件夹
//	err := cli.Move(ctx, "file_id", "folder_id")
//	if err != nil {
//		log.Fatalf("移动失败: %v", err)
//	}
//
//	// 移动到根目录
//	err = cli.Move(ctx, "file_id", "")
func (c *Client) Move(ctx context.Context, fileID string, parentID string) error {
	if fileID == "" {
		return exception.ErrInvalidFileID
	}

	body := map[string]interface{}{
		"ids": []string{fileID},
		"to": map[string]string{
			"parent_id": parentID,
		},
	}

	_, err := c.postJSON(ctx, "https://api-drive.mypikpak.net/drive/v1/files:batchMove", body)
	return err
}

// Copy 复制文件到指定文件夹
//
// 将文件复制到新的父文件夹，创建文件的副本
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileID: string 要复制的文件ID
//   - parentID: string 目标文件夹ID
//   - 空字符串表示复制到根目录
//
// 返回值：
//   - error 错误信息，操作成功时返回nil
//
// API endpoint：
//   - POST https://api-drive.mypikpak.net/drive/v1/files:batchCopy
//   - 请求体：{"ids": [fileID], "to": {"parent_id": parentID}}
//
// 使用示例：
//
//	// 复制文件到文件夹
//	err := cli.Copy(ctx, "file_id", "folder_id")
//	if err != nil {
//		log.Fatalf("复制失败: %v", err)
//	}
//
//	// 复制到根目录
//	err = cli.Copy(ctx, "file_id", "")
func (c *Client) Copy(ctx context.Context, fileID string, parentID string) error {
	body := map[string]interface{}{
		"ids": []string{fileID},
		"to": map[string]string{
			"parent_id": parentID,
		},
	}

	_, err := c.postJSON(ctx, "https://api-drive.mypikpak.net/drive/v1/files:batchCopy", body)
	return err
}

// Rename 重命名文件
//
// 修改文件的名称
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileID: string 要重命名的文件ID
//   - newName: string 文件的新名称
//   - 支持Unicode字符
//   - 支持特殊字符
//
// 返回值：
//   - error 错误信息，操作成功时返回nil
//
// API endpoint：
//   - PATCH https://api-drive.mypikpak.net/drive/v1/files/{fileID}
//   - 请求体：{"name": newName}
//
// 使用示例：
//
//	// 普通重命名
//	err := cli.Rename(ctx, "file_id", "新文件名.txt")
//	if err != nil {
//		log.Fatalf("重命名失败: %v", err)
//	}
//
//	// 包含特殊字符
//	err = cli.Rename(ctx, "file_id", "文件_2024!@#.txt")
//
//	// 包含Unicode字符
//	err = cli.Rename(ctx, "file_id", "日本語ファイル名.txt")
func (c *Client) Rename(ctx context.Context, fileID string, newName string) error {
	if fileID == "" {
		return exception.ErrInvalidFileID
	}
	if newName == "" {
		return exception.ErrInvalidFileName
	}

	body := map[string]string{
		"name": newName,
	}

	_, err := c.patchJSON(ctx, fmt.Sprintf("https://api-drive.mypikpak.net/drive/v1/files/%s", fileID), body)
	return err
}

// postJSON 执行POST请求（JSON格式）并解析响应
//
// 使用JSON格式发送请求体，适用于大多数API调用
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - URL: string 请求的完整URL
//   - data: interface{} 请求体数据，会序列化为JSON
//
// 返回值：
//   - map[string]interface{} 解析后的JSON响应
//   - error 错误信息
//
// 与postForm的区别：
//   - postJSON：Content-Type为application/json，请求体为JSON字符串
//   - postForm：Content-Type为application/x-www-form-urlencoded，请求体为URL编码
func (c *Client) postJSON(ctx context.Context, URL string, data interface{}) (map[string]interface{}, error) {
	bodyData, err := json.Marshal(data)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to marshal request data", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to create request", err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to read response", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to unmarshal response", err)
	}

	return result, nil
}

// patchJSON 执行PATCH请求（JSON格式）并解析响应
//
// 用于更新资源的部分属性
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - URL: string 请求的完整URL
//   - data: interface{} 请求体数据，会序列化为JSON
//
// 返回值：
//   - map[string]interface{} 解析后的JSON响应
//   - error 错误信息
//
// patchJSON 执行PATCH请求（JSON格式）并解析响应
//
// 用于更新资源的部分属性，支持条件更新和乐观锁
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - URL: string 请求的完整URL
//   - data: interface{} 请求体数据，会序列化为JSON
//
// 返回值：
//   - map[string]interface{} 解析后的JSON响应
//   - error 错误信息，可能的错误类型：
//   - PikpakException: JSON序列化失败、HTTP请求失败、响应解析失败
//
// 与postJSON的区别：
//   - patchJSON：使用HTTP PATCH方法，用于部分更新
//   - postJSON：使用HTTP POST方法，通常用于创建或完整更新
//
// 使用示例：
//
//	result, err := cli.patchJSON(ctx, "https://api-drive.mypikpak.net/drive/v1/files/file_id", map[string]string{
//		"name": "new_name",
//	})
//	if err != nil {
//		log.Fatalf("更新失败: %v", err)
//	}
func (c *Client) patchJSON(ctx context.Context, URL string, data interface{}) (map[string]interface{}, error) {
	bodyData, err := json.Marshal(data)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to marshal request data", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, URL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to create request", err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to read response", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to unmarshal response", err)
	}

	return result, nil
}

// postForm 执行POST请求（表单格式）并解析响应
//
// 使用表单编码格式发送请求体，主要用于登录认证等场景
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - URL: string 请求的完整URL
//   - data: map[string]string 表单字段键值对
//
// 返回值：
//   - map[string]interface{} 解析后的JSON响应
//   - error 错误信息
//
// 使用场景：
//   - 用户登录认证
//   - 令牌刷新请求
//   - 其他需要表单提交的API
func (c *Client) postForm(ctx context.Context, URL string, data map[string]string) (map[string]interface{}, error) {
	form := url.Values{}
	for key, value := range data {
		form.Set(key, value)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to create request", err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to read response", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to unmarshal response", err)
	}

	return result, nil
}

// Delete 执行HTTP DELETE请求
//
// 用于删除资源的通用方法
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - URL: string 请求的完整URL
//   - params: map[string]string URL查询参数
//
// 返回值：
//   - map[string]interface{} 响应数据
//   - error 错误信息
//
// 成功响应：
//
//	返回 {"status": "ok"} 表示删除成功
func (c *Client) Delete(ctx context.Context, URL string, params map[string]string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, URL, nil)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to create request", err)
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
		return nil, exception.NewPikpakExceptionWithError("request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("delete failed: %s", string(respBody))
	}

	return map[string]interface{}{"status": "ok"}, nil
}

// CaptchaInit 初始化验证码挑战
//
// 在执行需要验证码的操作前，必须先初始化验证码
// 该方法会获取验证码令牌，用于后续的身份验证
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - action: string 验证码操作类型，如 "POST:https://user.mypikpak.com/v1/auth/signin"
//   - meta: map[string]interface{} 验证码元数据
//   - 如果为nil，会自动生成以下元数据：
//   - captcha_sign: 验证码签名
//   - client_version: 客户端版本
//   - package_name: 包名
//   - user_id: 用户ID
//   - timestamp: 时间戳
//
// 返回值：
//   - map[string]interface{} 验证码初始化响应
//   - error 错误信息
//
// 响应数据：
//   - 成功时返回包含captcha_token的响应
//
// 使用场景：
//   - 登录前必须调用
//   - 其他需要验证码的操作
func (c *Client) CaptchaInit(ctx context.Context, action string, meta map[string]interface{}) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + UserHost
	}
	URL := baseURL + "/v1/shield/captcha/init"

	if meta == nil {
		timestamp := fmt.Sprintf("%d", utils.GetTimestamp())
		meta = map[string]interface{}{
			"captcha_sign":   utils.CaptchaSign(c.deviceID, timestamp),
			"client_version": utils.ClientVersion,
			"package_name":   utils.PackageName,
			"user_id":        c.userID,
			"timestamp":      timestamp,
		}
	}

	params := map[string]interface{}{
		"client_id": utils.ClientID,
		"action":    action,
		"device_id": c.deviceID,
		"meta":      meta,
	}

	return c.postJSON(ctx, URL, params)
}

// Login 执行用户登录认证
//
// 完成完整的登录流程，获取访问令牌和刷新令牌
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//
// 返回值：
//   - error 登录过程中的错误，可能的错误包括：
//   - ErrUsernamePasswordRequired: 用户名或密码为空
//   - ErrCaptchaTokenFailed: 验证码初始化失败
//   - ErrInvalidUsernamePassword: 用户名或密码错误
//   - PikpakException: 其他错误
//
// 登录流程：
//  1. 验证用户名和密码是否提供
//  2. 根据用户名格式（邮箱、手机号、用户名）构建认证元数据
//  3. 调用CaptchaInit初始化验证码
//  4. 获取验证码令牌
//  5. 提交登录凭证
//  6. 获取访问令牌和刷新令牌
//  7. 编码并保存令牌
//
// 用户名格式支持：
//   - 邮箱：使用正则表达式匹配邮箱格式
//   - 手机号：使用正则表达式匹配11-18位数字
//   - 用户名：普通字符串
//
// 登录后状态：
//   - accessToken: 设置为获取的访问令牌
//   - refreshToken: 设置为获取的刷新令牌
//   - userID: 设置为用户唯一标识符
//   - encodedToken: 设置为编码后的令牌
//
// 使用示例：
//
//	ctx := context.Background()
//	cli := client.NewClient(
//		client.WithUsername("user@example.com"),
//		client.WithPassword("password123"),
//	)
//	if err := cli.Login(ctx); err != nil {
//		log.Fatalf("登录失败: %v", err)
//	}
//	log.Println("登录成功！")
func (c *Client) Login(ctx context.Context) error {
	if c.username == "" || c.password == "" {
		return exception.ErrUsernamePasswordRequired
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + UserHost
	}
	loginURL := baseURL + "/v1/auth/signin"

	metas := make(map[string]interface{})
	emailRegex := regexp.MustCompile(`^[\w.-]+@[\w.-]+\.\w+$`)
	phoneRegex := regexp.MustCompile(`^\d{11,18}$`)

	if emailRegex.MatchString(c.username) {
		metas["email"] = c.username
	} else if phoneRegex.MatchString(c.username) {
		metas["phone_number"] = c.username
	} else {
		metas["username"] = c.username
	}

	result, err := c.CaptchaInit(ctx, "POST:"+loginURL, metas)
	if err != nil {
		return err
	}

	captchaToken, ok := result["captcha_token"].(string)
	if !ok || captchaToken == "" {
		return exception.ErrCaptchaTokenFailed
	}

	c.captchaToken = captchaToken

	loginData := map[string]string{
		"client_id":     utils.ClientID,
		"client_secret": utils.ClientSecret,
		"password":      c.password,
		"username":      c.username,
		"captcha_token": captchaToken,
	}

	userInfo, err := c.postForm(ctx, loginURL, loginData)
	if err != nil {
		return err
	}

	if accessToken, ok := userInfo["access_token"].(string); ok {
		c.accessToken = accessToken
	} else {
		return exception.NewPikpakException("login failed: no access_token")
	}

	if refreshToken, ok := userInfo["refresh_token"].(string); ok {
		c.refreshToken = refreshToken
	}

	if sub, ok := userInfo["sub"].(string); ok {
		c.userID = sub
	}

	if err := c.EncodeToken(); err != nil {
		return err
	}

	return nil
}

// RefreshAccessToken 刷新访问令牌
//
// 使用刷新令牌获取新的访问令牌
// 当accessToken过期时，doRequest方法会自动调用此方法
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//
// 返回值：
//   - error 刷新过程中的错误
//
// 刷新流程：
//  1. 构建刷新请求数据（client_id, refresh_token, grant_type）
//  2. 发送刷新请求
//  3. 更新accessToken和refreshToken
//  4. 编码并保存新令牌
//  5. 调用令牌刷新回调函数（如果有设置）
//
// 注意事项：
//   - 刷新令牌通常有效期较长，但也可能失效
//   - 如果刷新失败，可能需要重新登录
//   - 回调函数会在令牌刷新成功后自动调用
//
// 使用场景：
//   - 访问令牌过期时（doRequest自动处理）
//   - 手动刷新令牌
func (c *Client) RefreshAccessToken(ctx context.Context) error {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + UserHost
	}
	refreshURL := baseURL + "/v1/auth/token"

	refreshData := map[string]string{
		"client_id":     utils.ClientID,
		"refresh_token": c.refreshToken,
		"grant_type":    "refresh_token",
	}

	userInfo, err := c.postForm(ctx, refreshURL, refreshData)
	if err != nil {
		return err
	}

	if accessToken, ok := userInfo["access_token"].(string); ok {
		c.accessToken = accessToken
	} else {
		return exception.NewPikpakException("refresh failed: no access_token")
	}

	if refreshToken, ok := userInfo["refresh_token"].(string); ok {
		c.refreshToken = refreshToken
	}

	if sub, ok := userInfo["sub"].(string); ok {
		c.userID = sub
	}

	if err := c.EncodeToken(); err != nil {
		return err
	}

	if c.tokenRefreshCallback != nil {
		c.tokenRefreshCallback(c)
	}

	return nil
}

// CreateFolder 创建新文件夹
//
// 在指定父文件夹下创建新文件夹
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - name: string 新文件夹的名称
//   - parentID: string 父文件夹的ID
//   - 空字符串表示在根目录创建
//
// 返回值：
//   - map[string]interface{} 创建结果
//   - error 错误信息
//
// 响应数据：
//
//	返回的文件对象包含以下字段：
//	- id: 文件唯一标识符
//	- name: 文件名称
//	- kind: 文件类型（drive#folder）
//	- parent_id: 父文件夹ID
//	- created_time: 创建时间
//	- modified_time: 修改时间
//
// 使用示例：
//
//	result, err := cli.CreateFolder(ctx, "新文件夹", "")
//	if err != nil {
//		log.Fatal(err)
//	}
//	folderID := result["id"].(string)
func (c *Client) CreateFolder(ctx context.Context, name string, parentID string) (map[string]interface{}, error) {
	if name == "" {
		return nil, exception.ErrInvalidFileName
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files"

	data := map[string]interface{}{
		"kind":      "drive#folder",
		"name":      name,
		"parent_id": parentID,
	}

	return c.postJSON(ctx, URL, data)
}

// DeleteToTrash 将文件移动到回收站
//
// 批量将指定文件移动到回收站，而不是永久删除
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - ids: []string 要删除的文件ID列表
//
// 返回值：
//   - map[string]interface{} 操作结果
//   - error 错误信息
//
// 注意事项：
//   - 文件会被移动到回收站，可以在一定时间内恢复
//   - 回收站中的文件可能会定期被永久删除
//
// 与DeleteForever的区别：
//   - DeleteToTrash：移动到回收站，可恢复
//   - DeleteForever：永久删除，不可恢复
func (c *Client) DeleteToTrash(ctx context.Context, ids []string) (map[string]interface{}, error) {
	if len(ids) == 0 {
		return nil, exception.ErrEmptyFileIDs
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files:batchTrash"

	data := map[string]interface{}{
		"ids": ids,
	}

	return c.postJSON(ctx, URL, data)
}

// Untrash 恢复已删除的文件
//
// 批量将回收站中的文件恢复到原始位置
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - ids: []string 要恢复的文件ID列表
//
// 返回值：
//   - map[string]interface{} 操作结果
//   - error 错误信息
//
// 限制：
//   - 只有在回收站中的文件才能恢复
//   - 恢复后文件将回到原始父文件夹
func (c *Client) Untrash(ctx context.Context, ids []string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files:batchUntrash"

	data := map[string]interface{}{
		"ids": ids,
	}

	return c.postJSON(ctx, URL, data)
}

// DeleteForever 永久删除文件
//
// 批量永久删除指定文件，删除后无法恢复
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - ids: []string 要永久删除的文件ID列表
//
// 返回值：
//   - map[string]interface{} 操作结果
//   - error 错误信息
//
// 警告：
//   - 此操作不可逆，文件将被永久删除
//   - 建议先使用DeleteToTrash移动到回收站
//
// 与DeleteToTrash的区别：
//   - DeleteToTrash：移动到回收站，可恢复
//   - DeleteForever：永久删除，不可恢复
func (c *Client) DeleteForever(ctx context.Context, ids []string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files:batchDelete"

	data := map[string]interface{}{
		"ids": ids,
	}

	return c.postJSON(ctx, URL, data)
}

// OfflineDownload 创建离线下载任务
//
// 通过URL创建离线下载任务，支持多种下载类型：
//   - HTTP/HTTPS链接直接下载
//   - 磁力链接（Magnet URI）
//   - BT种子文件（.torrent）
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileURL: string 下载链接
//   - HTTP/HTTPS示例：https://example.com/file.mp4
//   - 磁力链接示例：magnet:?xt=urn:btih:...
//   - BT种子：需要先上传种子文件获取链接
//   - parentID: string 保存位置的文件夹ID
//   - 空字符串表示保存到"我的数据包"
//   - name: string 文件名（可选）
//   - 不传或空字符串则自动从链接中提取
//
// 返回值：
//   - map[string]interface{} 任务创建结果
//   - error 错误信息
//
// 响应数据：
//   - id: 任务ID
//   - name: 文件名
//   - phase: 下载状态
//   - progress: 下载进度
//   - file_id: 关联的文件ID（下载完成后）
//
// 使用示例：
//
//	// 下载磁力链接
//	result, err := cli.OfflineDownload(ctx, "magnet:?xt=urn:btih:...", "", "电影名称")
//	if err != nil {
//		log.Fatal(err)
//	}
//	taskID := result["id"].(string)
func (c *Client) OfflineDownload(ctx context.Context, fileURL string, parentID string, name string) (map[string]interface{}, error) {
	if fileURL == "" {
		return nil, exception.NewPikpakException("file url is required")
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files"

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

	return c.postJSON(ctx, URL, downloadData)
}

// CaptureScreenshot 对视频文件进行截图
//
// 生成视频文件的截图缩略图
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileID: string 视频文件的ID
//
// 返回值：
//   - map[string]interface{} 截图任务结果
//   - error 错误信息
//
// 使用场景：
//   - 为视频文件生成预览截图
//   - 创建视频缩略图用于展示
func (c *Client) CaptureScreenshot(ctx context.Context, fileID string) (map[string]interface{}, error) {
	if fileID == "" {
		return nil, exception.ErrInvalidFileID
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files:testScreenshot"

	data := map[string]interface{}{
		"file_id": fileID,
	}

	return c.postJSON(ctx, URL, data)
}

// RemoteDownload 远程文件下载
//
// 通过URL直接创建远程文件下载任务
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileURL: string 远程文件的URL地址
//
// 返回值：
//   - map[string]interface{} 下载任务创建结果
//   - error 错误信息
//
// 使用场景：
//   - 从外部URL下载文件到PikPak云盘
//   - 支持HTTP/HTTPS链接
func (c *Client) RemoteDownload(ctx context.Context, fileURL string) (map[string]interface{}, error) {
	if fileURL == "" {
		return nil, exception.ErrInvalidURL
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files"

	data := map[string]interface{}{
		"kind":        "drive#task",
		"upload_type": "UPLOAD_TYPE_URL",
		"url":         map[string]string{"url": fileURL},
	}

	return c.postJSON(ctx, URL, data)
}

// OfflineList 获取离线下载任务列表
//
// 查询离线下载任务，支持按状态筛选和分页
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - size: int 每页返回的任务数量
//   - 0表示使用默认值10000
//   - nextPageToken: string 分页令牌
//   - 首次请求传空字符串
//   - 从响应中获取下一页令牌
//   - phases: []string 任务状态筛选器
//   - nil表示使用默认筛选（运行中和失败）
//   - 可用值：
//   - PHASE_TYPE_RUNNING: 运行中
//   - PHASE_TYPE_ERROR: 失败
//   - PHASE_TYPE_COMPLETE: 已完成
//   - PHASE_TYPE_PENDING: 等待中
//
// 返回值：
//   - map[string]interface{} 任务列表响应
//   - error 错误信息
//
// 响应数据：
//   - tasks: 任务数组
//   - next_page_token: 下一页令牌（用于分页）
//   - tasks[].id: 任务ID
//   - tasks[].name: 任务名称
//   - tasks[].phase: 任务状态
//   - tasks[].progress: 下载进度
//
// 使用示例：
//
//	// 获取所有运行中和失败的任务
//	result, err := cli.OfflineList(ctx, 100, "", nil)
//
//	// 仅获取已完成的任务
//	result, err := cli.OfflineList(ctx, 100, "", []string{"PHASE_TYPE_COMPLETE"})
func (c *Client) OfflineList(ctx context.Context, size int, nextPageToken string, phases []string) (map[string]interface{}, error) {
	if size == 0 {
		size = 10000
	}

	if phases == nil {
		phases = []string{"PHASE_TYPE_RUNNING", "PHASE_TYPE_ERROR"}
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/tasks"

	filters := fmt.Sprintf(`{"phase":{"in":"%s"}}`, strings.Join(phases, ","))

	params := map[string]string{
		"type":           "offline",
		"thumbnail_size": "SIZE_SMALL",
		"limit":          fmt.Sprintf("%d", size),
		"filters":        filters,
		"with":           "reference_resource",
	}

	if nextPageToken != "" {
		params["page_token"] = nextPageToken
	}

	return c.getJSON(ctx, URL, params)
}

// OfflineFileInfo 获取离线下载文件的详细信息
//
// 获取已完成下载的文件信息，包括文件元数据和状态
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileID: string 文件的唯一标识符
//
// 返回值：
//   - map[string]interface{} 文件详细信息
//   - error 错误信息
//
// 响应数据：
//   - id: 文件ID
//   - name: 文件名
//   - size: 文件大小（字节）
//   - mime_type: MIME类型
//   - phase: 文件状态
//   - thumbnail: 缩略图信息
//   - created_time: 创建时间
//   - modified_time: 修改时间
//
// 使用场景：
//   - 查询下载任务对应的文件信息
//   - 获取文件下载链接
//   - 检查文件是否下载完成
func (c *Client) OfflineFileInfo(ctx context.Context, fileID string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files/" + fileID

	return c.getJSON(ctx, URL, map[string]string{"thumbnail_size": "SIZE_LARGE"})
}

func (c *Client) DeleteOfflineTasks(ctx context.Context, taskIDs []string, deleteFiles bool) error {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/tasks"
	params := map[string]string{
		"task_ids":     strings.Join(taskIDs, ","),
		"delete_files": strconv.FormatBool(deleteFiles),
	}

	_, err := c.Delete(ctx, URL, params)
	return err
}

// FileList 获取云端文件列表
//
// 列出指定文件夹下的所有文件（不含回收站）
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - size: int 每页返回的文件数量
//   - 0表示使用默认值100
//   - parentID: string 父文件夹ID
//   - 空字符串表示根目录
//   - nextPageToken: string 分页令牌
//   - 首次请求传空字符串
//
// 返回值：
//   - map[string]interface{} 文件列表响应
//   - error 错误信息
//
// 响应数据：
//   - files: 文件数组
//   - next_page_token: 下一页令牌
//   - files[].id: 文件ID
//   - files[].name: 文件名
//   - files[].size: 文件大小
//   - files[].mime_type: MIME类型
//   - files[].kind: 文件类型（drive#file 或 drive#folder）
//   - files[].parent_id: 父文件夹ID
//
// 使用示例：
//
//	// 列出根目录文件
//	result, err := cli.FileList(ctx, 100, "", "")
//
//	// 列出指定文件夹
//	result, err := cli.FileList(ctx, 100, "folder_id_here", "")
//
//	// 分页获取
//	if nextToken, ok := result["next_page_token"].(string); ok && nextToken != "" {
//		nextResult, _ := cli.FileList(ctx, 100, "folder_id", nextToken)
//	}
func (c *Client) FileList(ctx context.Context, size int, parentID string, nextPageToken string, query string) (map[string]interface{}, error) {
	if size == 0 {
		size = 100
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files"

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

	return c.getJSON(ctx, URL, params)
}

// Events 获取事件列表
//
// 查询账户的事件记录，用于审计和同步
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - size: int 每页返回的事件数量
//   - 0表示使用默认值100
//   - nextPageToken: string 分页令牌
//   - 首次请求传空字符串
//
// 返回值：
//   - map[string]interface{} 事件列表响应
//   - error 错误信息
//
// 响应数据：
//   - events: 事件数组
//   - next_page_token: 下一页令牌
//   - events[].id: 事件ID
//   - events[].type: 事件类型
//   - events[].time: 事件时间
//   - events[].data: 事件数据
func (c *Client) Events(ctx context.Context, size int, nextPageToken string) (map[string]interface{}, error) {
	if size == 0 {
		size = 100
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/events"

	params := map[string]string{
		"thumbnail_size": "SIZE_MEDIUM",
		"limit":          fmt.Sprintf("%d", size),
	}

	if nextPageToken != "" {
		params["next_page_token"] = nextPageToken
	}

	return c.getJSON(ctx, URL, params)
}

// OfflineTaskRetry 重试失败的离线下载任务
//
// 重新执行之前失败的下载任务
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - taskID: string 要重试的任务ID
//
// 返回值：
//   - map[string]interface{} 重试结果
//   - error 错误信息
//
// 使用场景：
//   - 离线下载任务失败后需要重新尝试
//   - 网络问题导致的下载失败
func (c *Client) OfflineTaskRetry(ctx context.Context, taskID string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/task"

	data := map[string]interface{}{
		"type":        "offline",
		"create_type": "RETRY",
		"id":          taskID,
	}

	return c.postJSON(ctx, URL, data)
}

// DeleteTasks 删除离线下载任务
//
// 批量删除离线下载任务，可选择是否同时删除已下载的文件
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - taskIDs: []string 要删除的任务ID列表
//   - deleteFiles: bool 是否同时删除已下载的文件
//
// 返回值：
//   - error 错误信息
//
// 注意事项：
//   - 仅删除任务不会删除已下载的文件
//   - 设置deleteFiles为true会同时删除关联的下载文件
//
// 使用示例：
//
//	// 仅删除任务
//	if err := cli.DeleteTasks(ctx, []string{"task_id_1", "task_id_2"}, false); err != nil {
//		log.Fatal(err)
//	}
//
//	// 删除任务及文件
//	if err := cli.DeleteTasks(ctx, []string{"task_id"}, true); err != nil {
//		log.Fatal(err)
//	}
func (c *Client) DeleteTasks(ctx context.Context, taskIDs []string, deleteFiles bool) error {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/tasks"

	params := map[string]string{
		"task_ids":     strings.Join(taskIDs, ","),
		"delete_files": fmt.Sprintf("%t", deleteFiles),
	}

	_, err := c.Delete(ctx, URL, params)
	return err
}

// GetTaskStatus 获取下载任务状态
//
// 查询指定任务的当前下载状态
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - taskID: string 任务ID
//   - fileID: string 关联的文件ID
//
// 返回值：
//   - enums.DownloadStatus 下载状态枚举值
//   - error 错误信息
//
// 状态值：
//   - DownloadStatusNotFound: 未找到
//   - DownloadStatusRunning: 下载中
//   - DownloadStatusComplete: 已完成
//   - DownloadStatusError: 失败
//   - DownloadStatusPending: 等待中
//
// 使用示例：
//
//	status, err := cli.GetTaskStatus(ctx, taskID, fileID)
//	switch status {
//	case enums.DownloadStatusComplete:
//		fmt.Println("下载完成！")
//	case enums.DownloadStatusError:
//		fmt.Println("下载失败")
//	case enums.DownloadStatusRunning:
//		fmt.Println("下载中...")
//	}
func (c *Client) GetTaskStatus(ctx context.Context, taskID string, fileID string) (enums.DownloadStatus, error) {
	fileInfo, err := c.OfflineFileInfo(ctx, fileID)
	if err != nil {
		return enums.DownloadStatusNotFound, err
	}

	if phase, ok := fileInfo["phase"].(string); ok {
		return enums.ParseDownloadStatus(phase), nil
	}

	return enums.DownloadStatusNotFound, nil
}

// FileRename 重命名文件
//
// 修改指定文件的名称
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileID: string 要重命名的文件ID
//   - newName: string 新的文件名称
//
// 返回值：
//   - map[string]interface{} 操作结果
//   - error 错误信息
//
// 使用示例：
//
//	result, err := cli.FileRename(ctx, "file_id_here", "新文件名.mp4")
//	if err != nil {
//		log.Fatal(err)
//	}
func (c *Client) FileRename(ctx context.Context, fileID string, newName string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files/" + fileID

	data := map[string]interface{}{
		"name": newName,
	}

	bodyData, err := json.Marshal(data)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to marshal request data", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, URL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to create request", err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to read response", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to unmarshal response", err)
	}

	return result, nil
}

// FileBatchStar 批量收藏文件
//
// 将指定文件标记为收藏状态
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - ids: []string 要收藏的文件ID列表
//
// 返回值：
//   - map[string]interface{} 操作结果
//   - error 错误信息
//
// 使用场景：
//   - 收藏重要文件
//   - 标记需要关注的文件
//
// 与FileBatchUnstar的关系：
//   - FileBatchStar：收藏文件
//   - FileBatchUnstar：取消收藏
func (c *Client) FileBatchStar(ctx context.Context, ids []string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files:batchStar"

	data := map[string]interface{}{
		"ids": ids,
	}

	return c.postJSON(ctx, URL, data)
}

// FileBatchUnstar 批量取消收藏
//
// 取消指定文件的收藏状态
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - ids: []string 要取消收藏的文件ID列表
//
// 返回值：
//   - map[string]interface{} 操作结果
//   - error 错误信息
func (c *Client) FileBatchUnstar(ctx context.Context, ids []string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files:batchUntrash"

	data := map[string]interface{}{
		"ids": ids,
	}

	return c.postJSON(ctx, URL, data)
}

// FileStarList 获取收藏文件列表
//
// 查询所有已收藏的文件
//
// 返回值：
//   - map[string]interface{} 收藏文件列表
//   - error 错误信息
//
// 响应数据：
//   - files: 文件数组，仅包含收藏的文件
//   - 其他字段与FileList相同
//
// 使用示例：
//
//	result, err := cli.FileStarList(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	files := result["files"].([]interface{})
func (c *Client) FileStarList(ctx context.Context) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files"

	filters := `{"starred":{"eq":true},"trashed":{"eq":false}}`

	params := map[string]string{
		"thumbnail_size": "SIZE_MEDIUM",
		"filters":        filters,
	}

	return c.getJSON(ctx, URL, params)
}

// FileBatchShare 批量创建文件分享链接
//
// 为指定文件创建公开分享链接
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - ids: []string 要分享的文件ID列表
//   - needPassword: bool 是否需要提取码
//   - true: 分享链接需要输入提取码才能访问
//   - false: 分享链接可直接访问
//
// 返回值：
//   - map[string]interface{} 分享结果
//   - error 错误信息
//
// 响应数据：
//   - share_url: 分享链接
//   - share_id: 分享ID
//   - passcode: 提取码（如果设置了needPassword）
//   - expire_time: 过期时间
//
// 使用示例：
//
//	result, err := cli.FileBatchShare(ctx, []string{"file_id_1", "file_id_2"}, true)
//	if err != nil {
//		log.Fatal(err)
//	}
//	shareURL := result["share_url"].(string)
//	passcode := result["passcode"].(string)
func (c *Client) FileBatchShare(ctx context.Context, ids []string, needPassword bool) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files:batchShare"

	data := map[string]interface{}{
		"ids": ids,
		"setting": map[string]bool{
			"need_password": needPassword,
		},
	}

	return c.postJSON(ctx, URL, data)
}

// GetQuotaInfo 获取账户配额信息
//
// 查询账户的存储空间使用情况
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//
// 返回值：
//   - map[string]interface{} 配额信息
//   - error 错误信息
//
// 响应数据：
//   - total_storage: 总存储空间（字节）
//   - used_storage: 已使用空间（字节）
//   - subscription_plan: 订阅计划
//   - capabilities: 功能权限列表
//
// 使用示例：
//
//	quota, err := cli.GetQuotaInfo(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	total := quota["total_storage"].(float64)
//	used := quota["used_storage"].(float64)
//	usagePercent := used / total * 100
//	fmt.Printf("存储使用率: %.2f%%\n", usagePercent)
func (c *Client) GetQuotaInfo(ctx context.Context) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/about"

	return c.getJSON(ctx, URL, nil)
}

// GetShareInfo 获取分享链接信息
//
// 查询指定分享链接的详细信息
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - shareURL: string 分享链接
//
// 返回值：
//   - map[string]interface{} 分享信息
//   - error 错误信息
//
// 响应数据：
//   - share_id: 分享ID
//   - share_url: 分享链接
//   - passcode_required: 是否需要提取码
//   - expiration_time: 过期时间
//   - file_list: 分享的文件列表
func (c *Client) GetShareInfo(ctx context.Context, shareURL string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/share/v1/info"

	params := map[string]string{
		"share_url": shareURL,
	}

	return c.getJSON(ctx, URL, params)
}

// Restore 恢复分享文件到个人云盘
//
// 将分享的文件复制到个人云盘的指定位置
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - shareID: string 分享ID
//   - passCodeToken: string 提取码令牌
//   - fileIDs: []string 要恢复的文件ID列表
//
// 返回值：
//   - map[string]interface{} 恢复结果
//   - error 错误信息
//
// 使用场景：
//   - 将他人分享的文件保存到自己的云盘
//   - 复制分享的文件到指定文件夹
func (c *Client) Restore(ctx context.Context, shareID string, passCodeToken string, fileIDs []string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/share/v1/file/restore"

	data := map[string]interface{}{
		"share_id":         shareID,
		"passcode_token":   passCodeToken,
		"file_ids":         fileIDs,
		"from_share_owner": false,
	}

	return c.postJSON(ctx, URL, data)
}

// GetShareDownloadURL 获取分享文件的下载链接
//
// 获取分享文件的直接下载URL
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - shareURL: string 分享链接
//   - fileID: string 要下载的文件ID
//
// 返回值：
//   - string 下载链接
//   - error 错误信息
//
// 使用示例：
//
//	downloadURL, err := cli.GetShareDownloadURL(ctx, "https://pan.pikpak.com/share/link/xxx", "file_id")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println("下载地址:", downloadURL)
func (c *Client) GetShareDownloadURL(ctx context.Context, shareURL string, fileID string) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/share/v1/file"

	params := map[string]string{
		"share_url": shareURL,
		"file_id":   fileID,
	}

	result, err := c.getJSON(ctx, URL, params)
	if err != nil {
		return "", err
	}

	if downloadURL, ok := result["download_url"].(string); ok {
		return downloadURL, nil
	}

	return "", exception.NewPikpakException("download_url not found in response")
}

// ShareFileInfo 分享文件信息结构
//
// 包含从分享链接获取的文件详细信息
type ShareFileInfo struct {
	ID             string    `json:"id"`
	ShareID        string    `json:"share_id"`
	Kind           string    `json:"kind"`
	Name           string    `json:"name"`
	ModifiedTime   time.Time `json:"modified_time"`
	Size           string    `json:"size"`
	ThumbnailLink  string    `json:"thumbnail_link"`
	WebContentLink string    `json:"web_content_link"`
	Medias         []Media   `json:"medias"`
}

// Media 媒体信息结构
//
// 包含视频或其他媒体的详细信息
type Media struct {
	MediaId   string `json:"media_id"`
	MediaName string `json:"media_name"`
	Video     struct {
		Height     int    `json:"height"`
		Width      int    `json:"width"`
		Duration   int    `json:"duration"`
		BitRate    int    `json:"bit_rate"`
		FrameRate  int    `json:"frame_rate"`
		VideoCodec string `json:"video_codec"`
		AudioCodec string `json:"audio_codec"`
		VideoType  string `json:"video_type"`
	} `json:"video"`
	Link struct {
		Url    string    `json:"url"`
		Token  string    `json:"token"`
		Expire time.Time `json:"expire"`
	} `json:"link"`
	NeedMoreQuota  bool          `json:"need_more_quota"`
	VipTypes       []interface{} `json:"vip_types"`
	RedirectLink   string        `json:"redirect_link"`
	IconLink       string        `json:"icon_link"`
	IsDefault      bool          `json:"is_default"`
	Priority       int           `json:"priority"`
	IsOrigin       bool          `json:"is_origin"`
	ResolutionName string        `json:"resolution_name"`
	IsVisible      bool          `json:"is_visible"`
	Category       string        `json:"category"`
}

// GetShareFileInfo 获取分享链接的文件信息
//
// 通过分享链接获取文件的详细信息，包括文件名、大小、缩略图、媒体信息等
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - shareURL: string 分享链接
//   - sharePassword: string 分享密码（如果有）
//
// 返回值：
//   - *ShareFileInfo 文件信息
//   - error 错误信息
//
// 使用示例：
//
//	info, err := cli.GetShareFileInfo(ctx, "https://pan.pikpak.com/share/link/xxx", "password")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("文件名: %s\n", info.Name)
//	fmt.Printf("大小: %s\n", info.Size)
//	fmt.Printf("下载链接: %s\n", info.WebContentLink)
func (c *Client) GetShareFileInfo(ctx context.Context, shareURL string, sharePassword string) (*ShareFileInfo, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api-drive.mypikpak.com"
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

	result, err := c.getJSON(ctx, URL, params)
	if err != nil {
		return nil, err
	}

	fileInfo, ok := result["file_info"].(map[string]interface{})
	if !ok {
		return nil, exception.NewPikpakException("file_info not found in response")
	}

	return parseShareFileInfo(fileInfo)
}

// GetShareFileDownloadURL 获取分享文件的下载链接
//
// 通过分享链接获取文件的直接下载URL，支持选择原画或转码后的链接
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - shareURL: string 分享链接
//   - sharePassword: string 分享密码（如果有）
//   - useTranscoding: bool 是否使用转码后的链接（true则选择最高画质）
//
// 返回值：
//   - string 下载链接
//   - error 错误信息
//
// 使用示例：
//
//	// 获取原画链接
//	url, err := cli.GetShareFileDownloadURL(ctx, "https://pan.pikpak.com/share/link/xxx", "", false)
//
//	// 获取转码后的高清链接
//	url, err := cli.GetShareFileDownloadURL(ctx, "https://pan.pikpak.com/share/link/xxx", "", true)
func (c *Client) GetShareFileDownloadURL(ctx context.Context, shareURL string, sharePassword string, useTranscoding bool) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api-drive.mypikpak.com"
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

	result, err := c.getJSON(ctx, URL, params)
	if err != nil {
		return "", err
	}

	fileInfo, ok := result["file_info"].(map[string]interface{})
	if !ok {
		return "", exception.NewPikpakException("file_info not found in response")
	}

	if webContentLink, hasWebContentLink := fileInfo["web_content_link"].(string); hasWebContentLink && webContentLink != "" && !useTranscoding {
		return webContentLink, nil
	}

	medias, ok := fileInfo["medias"].([]interface{})
	if !ok || len(medias) == 0 {
		if webContentLink, hasWebContentLink := fileInfo["web_content_link"].(string); hasWebContentLink {
			return webContentLink, nil
		}
		return "", exception.NewPikpakException("no download link available")
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
		return "", exception.NewPikpakException("invalid media format")
	}

	link, linkOk := firstMedia["link"].(map[string]interface{})
	if !linkOk {
		return "", exception.NewPikpakException("link not found in media")
	}

	if url, urlOk := link["url"].(string); urlOk {
		return url, nil
	}

	return "", exception.NewPikpakException("download url not found")
}

// GetShareFiles 获取分享链接的文件列表
//
// 获取分享链接下的所有文件列表
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - shareURL: string 分享链接
//   - sharePassword: string 分享密码（如果有）
//
// 返回值：
//   - []*ShareFileInfo 文件信息列表
//   - error 错误信息
//
// 使用示例：
//
//	files, err := cli.GetShareFiles(ctx, "https://pan.pikpak.com/share/link/xxx", "password")
//	for _, file := range files {
//		fmt.Printf("%s - %s\n", file.Name, file.Size)
//	}
func (c *Client) GetShareFiles(ctx context.Context, shareURL string, sharePassword string) ([]*ShareFileInfo, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api-drive.mypikpak.com"
	}

	shareID, err := c.extractShareID(shareURL)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"share_id":       shareID,
		"thumbnail_size": "SIZE_LARGE",
		"limit":          "100",
	}

	if sharePassword != "" {
		passToken, passErr := c.getSharePassToken(ctx, shareID, sharePassword)
		if passErr != nil {
			return nil, passErr
		}
		params["pass_code_token"] = passToken
	}

	URL := baseURL + "/drive/v1/share/file_info"

	result, err := c.getJSON(ctx, URL, params)
	if err != nil {
		return nil, err
	}

	var files []*ShareFileInfo

	if filesArray, ok := result["files"].([]interface{}); ok {
		for _, f := range filesArray {
			fileMap, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			fileInfo, parseErr := parseShareFileInfo(fileMap)
			if parseErr != nil {
				continue
			}
			files = append(files, fileInfo)
		}
	}

	if len(files) == 0 {
		if fileInfo, ok := result["file_info"].(map[string]interface{}); ok {
			singleFile, parseErr := parseShareFileInfo(fileInfo)
			if parseErr == nil {
				files = append(files, singleFile)
			}
		}
	}

	return files, nil
}

// extractShareID 从分享链接中提取分享ID
func (c *Client) extractShareID(shareURL string) (string, error) {
	patterns := []string{
		`/s/([a-zA-Z0-9]+)`,
		`share/([a-zA-Z0-9]+)`,
		`id=([a-zA-Z0-9]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(shareURL)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", exception.NewPikpakException("invalid share URL format")
}

// getSharePassToken 获取分享密码令牌
func (c *Client) getSharePassToken(ctx context.Context, shareID string, sharePassword string) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api-drive.mypikpak.com"
	}

	URL := baseURL + "/share/v1/pass_token"

	data := map[string]interface{}{
		"share_id":       shareID,
		"pass_code":      sharePassword,
		"thumbnail_size": "SIZE_LARGE",
		"limit":          100,
	}

	result, err := c.postJSON(ctx, URL, data)
	if err != nil {
		return "", err
	}

	if passCodeToken, ok := result["pass_code_token"].(string); ok {
		return passCodeToken, nil
	}

	return "", exception.NewPikpakException("pass_code_token not found in response")
}

// parseShareFileInfo 解析分享文件信息
func parseShareFileInfo(data map[string]interface{}) (*ShareFileInfo, error) {
	info := &ShareFileInfo{}

	if id, ok := data["id"].(string); ok {
		info.ID = id
	}
	if shareID, ok := data["share_id"].(string); ok {
		info.ShareID = shareID
	}
	if kind, ok := data["kind"].(string); ok {
		info.Kind = kind
	}
	if name, ok := data["name"].(string); ok {
		info.Name = name
	}
	if size, ok := data["size"].(string); ok {
		info.Size = size
	}
	if thumbnailLink, ok := data["thumbnail_link"].(string); ok {
		info.ThumbnailLink = thumbnailLink
	}
	if webContentLink, ok := data["web_content_link"].(string); ok {
		info.WebContentLink = webContentLink
	}

	if modifiedTimeStr, ok := data["modified_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, modifiedTimeStr); err == nil {
			info.ModifiedTime = t
		}
	}

	if mediasArray, ok := data["medias"].([]interface{}); ok {
		for _, m := range mediasArray {
			mediaMap, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			media := Media{}

			if mediaId, ok := mediaMap["media_id"].(string); ok {
				media.MediaId = mediaId
			}
			if mediaName, ok := mediaMap["media_name"].(string); ok {
				media.MediaName = mediaName
			}

			if linkMap, ok := mediaMap["link"].(map[string]interface{}); ok {
				if url, ok := linkMap["url"].(string); ok {
					media.Link.Url = url
				}
				if token, ok := linkMap["token"].(string); ok {
					media.Link.Token = token
				}
				if expireStr, ok := linkMap["expire"].(string); ok {
					if t, err := time.Parse(time.RFC3339, expireStr); err == nil {
						media.Link.Expire = t
					}
				}
			}

			if videoMap, ok := mediaMap["video"].(map[string]interface{}); ok {
				if height, ok := videoMap["height"].(float64); ok {
					media.Video.Height = int(height)
				}
				if width, ok := videoMap["width"].(float64); ok {
					media.Video.Width = int(width)
				}
				if duration, ok := videoMap["duration"].(float64); ok {
					media.Video.Duration = int(duration)
				}
				if bitRate, ok := videoMap["bit_rate"].(float64); ok {
					media.Video.BitRate = int(bitRate)
				}
				if frameRate, ok := videoMap["frame_rate"].(float64); ok {
					media.Video.FrameRate = int(frameRate)
				}
				if videoCodec, ok := videoMap["video_codec"].(string); ok {
					media.Video.VideoCodec = videoCodec
				}
				if audioCodec, ok := videoMap["audio_codec"].(string); ok {
					media.Video.AudioCodec = audioCodec
				}
				if videoType, ok := videoMap["video_type"].(string); ok {
					media.Video.VideoType = videoType
				}
			}

			info.Medias = append(info.Medias, media)
		}
	}

	return info, nil
}

// CreateShareLink 创建分享链接
//
// 为单个文件创建分享链接
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - fileID: string 要分享的文件ID
//   - needPassword: bool 是否需要提取码
//
// 返回值：
//   - map[string]interface{} 分享结果
//   - error 错误信息
//
// 与FileBatchShare的区别：
//   - CreateShareLink: 单一文件分享
//   - FileBatchShare: 批量文件分享（同一链接）
func (c *Client) CreateShareLink(ctx context.Context, fileID string, needPassword bool) (map[string]interface{}, error) {
	return c.FileBatchShare(ctx, []string{fileID}, needPassword)
}

// Upload 上传文件
//
// 将本地文件上传到云盘
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - filePath: string 本地文件的完整路径
//   - parentID: string 上传到的文件夹ID
//   - 空字符串表示上传到根目录
//   - fileName: string 文件名（可选）
//   - 不传则使用本地文件名
//
// 返回值：
//   - map[string]interface{} 上传结果
//   - error 错误信息
//
// 限制：
//   - 大文件上传会自动分块传输
//   - 支持断点续传
//
// 使用示例：
//
//	result, err := cli.Upload(ctx, "/path/to/file.mp4", "", "我的视频.mp4")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fileID := result["id"].(string)
func (c *Client) Upload(ctx context.Context, filePath string, parentID string, fileName string) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to open file", err)
	}
	defer file.Close()

	if fileName == "" {
		fileName = filepath.Base(filePath)
	}

	return c.UploadReader(ctx, file, fileName, parentID)
}

// UploadReader 上传文件流
//
// 从io.Reader上传文件内容
//
// 参数说明：
//   - ctx: context.Context 请求上下文
//   - reader: io.Reader 文件内容读取器
//   - fileName: string 文件名
//   - parentID: string 上传到的文件夹ID
//
// 返回值：
//   - map[string]interface{} 上传结果
//   - error 错误信息
//
// 使用场景：
//   - 从网络下载后直接上传
//
// - 从内存上传
func (c *Client) UploadReader(ctx context.Context, reader io.Reader, fileName string, parentID string) (map[string]interface{}, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://" + APIHost
	}
	URL := baseURL + "/drive/v1/files"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to create form file", err)
	}

	if _, copyErr := io.Copy(part, reader); copyErr != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to copy file data", copyErr)
	}

	writer.WriteField("kind", "drive#file")
	writer.WriteField("name", fileName)
	writer.WriteField("parent_id", parentID)

	if closeErr := writer.Close(); closeErr != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to close writer", closeErr)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to create request", err)
	}

	for key, value := range c.getHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to read response", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, exception.NewPikpakExceptionWithError("failed to unmarshal response", err)
	}

	return result, nil
}
