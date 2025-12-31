// Package config 提供了PikPak API客户端配置文件的加载和保存功能
//
// 该包负责管理客户端配置文件的读写操作：
//   - LoadConfig: 从多个可能的位置加载配置文件
//   - SaveConfig: 将配置保存到指定文件路径
//
// 配置文件格式：
//   - 使用JSON格式存储
//   - 文件扩展名：.json
//   - 支持多个搜索路径
//
// 配置文件搜索顺序：
//   1. 当前目录下的config.json
//   2. 当前目录下的.pikpakapi.json
//   3. 用户主目录下的.pikpakapi.json
//
// 使用示例：
//
//	// 加载配置
//	cfg, err := config.LoadConfig()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 创建新配置
//	newCfg := &config.Config{
//	    Username: "user@example.com",
//	    Password: "password123",
//	}
//
//	// 保存配置
//	if err := config.SaveConfig(newCfg, "config.json"); err != nil {
//	    log.Fatal(err)
//	}
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config PikPak API客户端的配置数据结构
//
// 该结构体存储了客户端的所有配置信息
// 支持JSON序列化和反序列化用于配置文件读写
//
// 字段说明：
//   - Username: string 用户登录名
//     - 支持邮箱、手机号或用户名格式
//     - 用于登录认证
//   - Password: string 用户密码
//     - 登录密码，明文存储（建议配合加密使用）
//     - 敏感信息，注意保护
//   - AccessToken: string 访问令牌
//     - API请求的身份验证凭证
//     - 有时效性，过期后需要刷新
//   - RefreshToken: string 刷新令牌
//     - 用于刷新访问令牌的长期凭证
//     - 有效期较长
//   - EncodedToken: string 编码后的令牌
//     - Base64编码的令牌字符串
//     - 便于持久化保存
//   - DeviceID: string 设备标识符
//     - 设备的唯一标识
//     - 用于设备绑定
//   - CaptchaToken: string 验证码令牌
//     - 登录时的验证码验证凭证
//     - 临时有效
//   - UserID: string 用户标识符
//     - 用户的唯一标识
//     - 由PikPak服务分配
//
// 配置文件格式：
//   {
//     "username": "user@example.com",
//     "password": "password123",
//     "access_token": "...",
//     "refresh_token": "...",
//     "encoded_token": "...",
//     "device_id": "...",
//     "captcha_token": "...",
//     "user_id": "..."
//   }
//
// 安全考虑：
//   - 密码和令牌是敏感信息
//   - 配置文件应设置适当的访问权限
//   - 生产环境建议加密存储敏感信息
type Config struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	EncodedToken  string `json:"encoded_token"`
	DeviceID      string `json:"device_id"`
	CaptchaToken  string `json:"captcha_token"`
	UserID        string `json:"user_id"`
}

// LoadConfig 从多个可能的位置加载配置文件
//
// 该函数按顺序搜索预定义的配置文件路径
// 找到第一个存在的有效配置文件即返回
// 如果所有配置文件都不存在，返回空的Config对象
//
// 配置文件搜索顺序：
//   1. 当前目录下的"config.json"
//   2. 当前目录下的".pikpakapi.json"
//   3. 用户主目录下的".pikpakapi.json"
//
// 返回值：
//   - *Config 加载的配置对象
//     - 如果找到配置文件，返回解析后的配置
//     - 如果未找到任何配置文件，返回空的Config{}
//   - error 错误信息
//     - 始终返回nil（当前实现不返回错误）
//     - 未来可能修改为返回读取错误
//
// 加载流程：
//   1. 遍历所有配置文件路径
//   2. 尝试读取文件内容
//   3. 解析JSON数据到Config结构体
//   4. 解析成功则返回，解析失败则继续尝试下一个路径
//   5. 所有路径都失败时返回空配置
//
// 使用场景：
//   - 程序启动时加载已保存的配置
//   - 恢复用户的登录状态
//   - 获取设备ID等预配置信息
//
// 注意事项：
//   - 函数不会返回错误，未找到配置时返回空对象
//   - 建议在保存配置时使用SaveConfig函数
//   - 配置文件权限应设置为0600以保护敏感信息
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

// SaveConfig 将配置保存到指定文件路径
//
// 该函数将Config结构体序列化为格式化的JSON字符串
// 并写入到指定的文件路径
//
// 参数说明：
//   - cfg: *Config 要保存的配置对象
//     - 包含所有要保存的配置信息
//     - 结构体会被序列化为JSON格式
//   - path: string 目标文件路径
//     - 可以是相对路径或绝对路径
//     - 文件目录必须存在，否则会保存失败
//
// 序列化格式：
//   - 使用json.MarshalIndent进行格式化
//   - 缩进为2个空格
//   - 键名使用原始的JSON标签
//
// 文件权限：
//   - 使用0644权限创建文件
//   - 文件所有者可读写，组用户和其他用户可读
//
// 返回值：
//   - error 保存过程中的错误
//     - 序列化失败：fmt.Errorf("failed to marshal config: %w", err)
//     - 写入失败：fmt.Errorf("failed to write config: %w", err)
//
// 使用场景：
//   - 登录成功后保存配置
//   - 令牌刷新后更新配置
//   - 保存用户偏好的配置项
//
// 注意事项：
//   - 密码和令牌是敏感信息
//   - 建议设置文件权限为0600
//   - 保存前可以加密敏感字段
//
// 使用示例：
//
//	cfg := &config.Config{
//	    Username:     "user@example.com",
//	    AccessToken:  "...",
//	    RefreshToken: "...",
//	    EncodedToken: "...",
//	}
//
//	if err := config.SaveConfig(cfg, "config.json"); err != nil {
//	    log.Fatalf("保存配置失败: %v", err)
//	}
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
