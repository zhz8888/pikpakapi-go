// Package utils 提供了与令牌（Token）管理相关的工具函数
//
// 该子包专注于令牌的编码和解码功能：
//   - EncodeToken: 将accessToken和refreshToken编码为Base64字符串
//   - DecodeToken: 将Base64编码的令牌字符串解码还原
//
// 编码格式：
//   - 使用JSON序列化令牌数据
//   - 使用Base64标准编码进行字符串转换
//   - 编码结果是URL安全的，可直接存储在配置文件或数据库中
//
// 使用示例：
//
//	// 编码令牌
//	encoded, err := utils.EncodeToken(accessToken, refreshToken)
//	// 保存encoded到配置文件
//
//	// 解码令牌
//	data, err := utils.DecodeToken(encodedString)
//	accessToken := data.AccessToken
//	refreshToken := data.RefreshToken
package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// TokenData 存储PikPak API访问令牌和刷新令牌的数据结构
//
// 该结构体用于在编码/解码过程中临时存储令牌信息
// PikPak API使用OAuth 2.0风格的令牌认证机制
//
// 字段说明：
//   - AccessToken: string 访问令牌
//     - 用于API请求的身份验证
//     - 有时效性，过期后需要使用RefreshToken获取新的AccessToken
//     - 通常有效期为数小时到数天
//   - RefreshToken: string 刷新令牌
//     - 用于刷新访问令牌的长期有效凭证
//     - 有效期较长，通常为数周或数月
//     - 应该妥善保管，避免泄露
//
// JSON序列化：
//   - TokenData会序列化为JSON格式：{"access_token":"...","refresh_token":"..."}
//   - 然后进行Base64编码以便存储和传输
//
// 生命周期：
//   1. 用户登录成功后获得AccessToken和RefreshToken
//   2. 使用EncodeToken编码后保存到配置文件
//   下次启动时使用DecodeToken还原
//   3. AccessToken过期时，使用RefreshToken调用RefreshAccessToken获取新的AccessToken
type TokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// EncodeToken 将访问令牌和刷新令牌编码为Base64字符串
//
// 该函数将两个令牌编码为可持久化存储的字符串格式
// 编码后的字符串是URL安全的，可以直接保存到配置文件或数据库中
//
// 参数说明：
//   - accessToken: string 访问令牌
//     - 用于API请求的身份验证
//     - 通常由Login或RefreshAccessToken方法获取
//   - refreshToken: string 刷新令牌
//     - 用于刷新访问令牌的长期凭证
//     - 用于获取新的accessToken
//
// 编码流程：
//   1. 创建TokenData结构体，填入两个令牌
//   2. 使用json.Marshal将结构体序列化为JSON字节
//   3. 使用base64.StdEncoding将JSON字节编码为Base64字符串
//
// 返回值：
//   - string 编码后的Base64字符串
//     - 格式：eyJhY2Nlc3NfdG9rZW4iOiJ4eXoiLCJyZWZyZXNoX3Rva2VuIjoieXl5In0=
//     - 可以直接存储，不需要额外转义
//   - error 编码过程中的错误
//     - 通常为JSON序列化失败（罕见）
//
// 使用场景：
//   - 登录成功后保存令牌到配置文件
//   - 令牌刷新成功后更新保存的令牌
//
// 注意事项：
//   - 编码不等于加密，任何人都可以解码获取原始令牌
//   - 应妥善保管编码后的字符串，避免泄露
//   - 可配合加密函数增强安全性
func EncodeToken(accessToken, refreshToken string) (string, error) {
	data := TokenData{
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

// DecodeToken 将Base64编码的令牌字符串解码还原
//
// 该函数是EncodeToken的逆操作，将编码后的字符串还原为TokenData结构体
// 用于从持久化存储中恢复令牌信息
//
// 参数说明：
//   - encodedToken: string Base64编码的令牌字符串
//     - 通常来自配置文件或数据库
//     - 格式：eyJhY2Nlc3NfdG9rZW4iOiJ4eXoiLCJyZWZyZXNoX3Rva2VuIjoieXl5In0=
//
// 解码流程：
//   1. 使用base64.StdEncoding.DecodeString解码Base64字符串
//   2. 使用json.Unmarshal将JSON字节反序列化为TokenData结构体
//   3. 验证令牌的有效性（accessToken和refreshToken都不为空）
//
// 返回值：
//   - *TokenData 解码后的令牌数据结构体
//     - 包含AccessToken和RefreshToken字段
//   - error 解码过程中的错误
//     - ErrInvalidEncodedToken: Base64解码失败
//     - ErrInvalidEncodedToken: JSON反序列化失败
//     - ErrInvalidEncodedToken: 令牌字段为空
//
// 使用场景：
//   - 程序启动时从配置文件加载保存的令牌
//   - 还原用户之前登录的会话状态
//
// 错误处理：
//   - 建议在调用前验证encodedToken不为空
//   - 错误信息应记录日志以便调试
//   - 解码失败时应提示用户重新登录
func DecodeToken(encodedToken string) (*TokenData, error) {
	jsonData, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	var data TokenData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	if data.AccessToken == "" || data.RefreshToken == "" {
		return nil, fmt.Errorf("invalid token: missing access_token or refresh_token")
	}

	return &data, nil
}
