// Package utils 提供了PikPak API客户端所需的公共工具函数
//
// 该工具包包含了以下功能模块：
//   - 加密哈希：MD5、SHA1等哈希算法的封装
//   - 签名生成：设备签名、验证码签名的生成
//   - User-Agent构建：构建符合PikPak协议的User-Agent字符串
//   - 时间戳：获取毫秒级时间戳
//
// 这些工具函数主要服务于API请求的身份验证和签名验证流程
package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	// ClientID PikPak API的客户端标识符
	// 用于标识API调用的客户端类型
	// 该值是固定不变的，由PikPak官方分配
	ClientID = "YNxT9w7GMdWvEOKa"

	// ClientSecret PikPak API的客户端密钥
	// 用于API请求的签名验证
	// 该值是固定不变的，由PikPak官方分配
	// 重要：请勿将此值泄露给第三方
	ClientSecret = "dbw2OtmVEeuUvIptb1Coyg"

	// ClientVersion PikPak Android客户端的版本号
	// 用于模拟Android客户端发起API请求
	// 版本号需要与服务端兼容，过旧的版本可能导致API调用失败
	ClientVersion = "1.47.1"

	// PackageName PikPak Android应用的包名
	// 用于标识应用的唯一性
	// 也是构建User-Agent和签名的基础参数
	PackageName = "com.pikcloud.pikpak"

	// SDKVersion PikPak SDK的版本号
	// 用于标识所使用的SDK版本
	SDKVersion = "2.0.4.204000"

	// AppName 应用程序名称
	// 与PackageName相同，用于构建User-Agent字符串
	AppName = PackageName
)

// salts 用于验证码签名计算的盐值数组
//
// 这些盐值是PikPak API签名算法的一部分
// 在计算CaptchaSign时，会将这些盐值依次追加到待签名字符串后进行MD5哈希
// 盐值列表的顺序和内容都是固定的，不能随意修改
//
// 签名算法流程：
//   1. 拼接基础字符串：ClientID + ClientVersion + PackageName + deviceID + timestamp
//   2. 依次对每个盐值进行MD5哈希：hash = md5(previous_hash + salt)
//   3. 最终在结果前添加版本前缀"1."
//
// 注意事项：
//   - 盐值列表包含多个元素，每个都会参与哈希计算
//   - 空字符串""也是有效的盐值
//   - 该算法确保签名的唯一性和安全性
var salts = []string{
	"Gez0T9ijiI9WCeTsKSg3SMlx",
	"zQdbalsolyb1R/",
	"ftOjr52zt51JD68C3s",
	"yeOBMH0JkbQdEFNNwQ0RI9T3wU/v",
	"BRJrQZiTQ65WtMvwO",
	"je8fqxKPdQVJiy1DM6Bc9Nb1",
	"niV",
	"9hFCW2R1",
	"sHKHpe2i96",
	"p7c5E6AcXQ/IJUuAEC9W6",
	"",
	"aRv9hjc9P+Pbn+u3krN6",
	"BzStcgE8qVdqjEH16l4",
	"SqgeZvL5j9zoHP95xWHt",
	"zVof5yaJkPe3VFpadPof",
}

// GetTimestamp 获取当前时间的毫秒级时间戳
//
// 返回自Unix纪元（1970年1月1日）以来的毫秒数
// PikPak API的许多请求需要使用毫秒级时间戳进行签名验证
//
// 返回值：
//   - int64 当前时间的毫秒时间戳
//
// 使用场景：
//   - 验证码签名生成
//   - User-Agent构建
//   - API请求的时间戳参数
//
// 性能说明：
//   - 该函数调用time.Now().UnixMilli()，性能开销极低
//   - 可在高频调用场景下安全使用
func GetTimestamp() int64 {
	return time.Now().UnixMilli()
}

// CaptchaSign 生成验证码签名
//
// 该函数用于生成PikPak API验证码请求所需的签名
// 签名算法确保请求的真实性和完整性，防止请求被篡改
//
// 参数说明：
//   - deviceID: string 设备标识符，用于区分不同的设备
//     - 可以是自动生成的设备ID
//     - 长度不限，但通常为32位MD5哈希值
//   - timestamp: string 时间戳字符串
//     - 通常使用毫秒级时间戳的字符串形式
//     - 确保签名的时效性
//
// 签名算法详解：
//   1. 基础字符串拼接：
//      sign = ClientID + ClientVersion + PackageName + deviceID + timestamp
//   2. 迭代哈希：
//      对于salts数组中的每个盐值：
//        sign = md5Hash(sign + salt)
//   3. 添加版本前缀：
//      result = "1." + sign
//
// 返回值：
//   - string 完整的验证码签名字符串
//     - 格式：1.{MD5哈希值}
//     - 例如：1.a1b2c3d4e5f6...
//
// 使用场景：
//   - CaptchaInit请求的签名验证
//   - 设备绑定验证
//
// 安全性说明：
//   - 签名算法是不可逆的
//   - 使用固定的盐值增加破解难度
//   - 时间戳确保签名的时效性
func CaptchaSign(deviceID string, timestamp string) string {
	sign := ClientID + ClientVersion + PackageName + deviceID + timestamp
	for _, salt := range salts {
		sign = md5Hash(sign + salt)
	}
	return fmt.Sprintf("1.%s", sign)
}

// GenerateDeviceSign 生成设备签名
//
// 该函数用于生成PikPak设备的唯一签名
// 签名通过组合设备ID、包名和应用密钥，经过双重哈希运算生成
// 用于User-Agent中，以验证设备的真实性
//
// 参数说明：
//   - deviceID: string 设备标识符
//     - 唯一标识一台设备
//     - 通常为32位MD5哈希值
//   - packageName: string 应用包名
//     - 对于PikPak应用，通常为"com.pikcloud.pikpak"
//     - 标识应用类型
//
// 签名算法详解：
//   1. 构建基础字符串：
//      signatureBase = deviceID + packageName + "1appkey"
//   2. 第一次哈希（SHA1）：
//      sha1Result = sha1(signatureBase)
//   3. 第二次哈希（MD5）：
//      md5Result = md5(sha1Result)
//   4. 组合结果：
//      result = "div101." + deviceID + md5Result
//
// 返回值：
//   - string 设备签名字符串
//     - 格式：div101.{deviceID}{MD5哈希值}
//     - 总长度约：8 + len(deviceID) + 32 = 约70字符
//
// 使用场景：
//   - 构建自定义User-Agent
//   - 设备验证请求
//
// 性能考虑：
//   - 涉及两次哈希运算，对于高频调用可考虑缓存签名
//   - SHA1和MD5都是计算效率较高的算法
func GenerateDeviceSign(deviceID string, packageName string) string {
	signatureBase := deviceID + packageName + "1appkey"

	sha1Hash := sha1.New()
	sha1Hash.Write([]byte(signatureBase))
	sha1Result := hex.EncodeToString(sha1Hash.Sum(nil))

	md5Hash := md5.New()
	md5Hash.Write([]byte(sha1Result))
	md5Result := hex.EncodeToString(md5Hash.Sum(nil))

	return fmt.Sprintf("div101.%s%s", deviceID, md5Result)
}

// BuildCustomUserAgent 构建符合PikPak协议的User-Agent字符串
//
// 该函数构建一个完整的User-Agent字符串，用于模拟PikPak Android客户端
// User-Agent中包含了设备信息、SDK版本、签名等关键数据
// PikPak服务器会验证User-Agent的格式和签名，不正确的User-Agent可能导致请求被拒绝
//
// 参数说明：
//   - deviceID: string 设备标识符
//     - 唯一标识一台设备
//     - 用于设备和签名验证
//   - userID: string 用户标识符
//     - 登录用户的唯一标识
//     - 用于追踪用户行为和设备绑定
//
// 返回值：
//   - string 完整的User-Agent字符串
//     - 格式：ANDROID-{packageName}/{version} protocolVersion/200 ...
//     - 包含约30个键值对，以空格分隔
//
// User-Agent字段详解：
//   - ANDROID-{pkg}/{ver}: 应用名称和版本
//   - protocolVersion: 协议版本，固定为200
//   - accesstype: 访问类型，空
//   - clientid: 客户端ID
//   - clientversion: 客户端版本
//   - action_type: 动作类型，空
//   - networktype: 网络类型，固定为WIFI
//   - sessionid: 会话ID，空
//   - deviceid: 设备ID
//   - providername: 提供者名称，固定为NONE
//   - devicesign: 设备签名
//   - refresh_token: 刷新令牌，空
//   - sdkversion: SDK版本
//   - datetime: 时间戳
//   - usrno: 用户编号（userID）
//   - appname: 应用名称
//   - session_origin: 会话来源，空
//   - grant_type: 授权类型，空
//   - appid: 应用ID，空
//   - clientip: 客户端IP，空
//   - devicename: 设备名称
//   - osversion: 操作系统版本
//   - platformversion: 平台版本
//   - accessmode: 访问模式，空
//   - devicemodel: 设备型号
//
// 使用场景：
//   - 验证码初始化请求
//   - 需要特殊User-Agent的API请求
//
// 注意事项：
//   - User-Agent格式必须严格遵守
//   - 设备签名必须有效
//   - 设备型号和名称建议使用真实设备信息
func BuildCustomUserAgent(deviceID string, userID string) string {
	deviceSign := GenerateDeviceSign(deviceID, PackageName)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ANDROID-%s/%s ", AppName, ClientVersion))
	sb.WriteString("protocolVersion/200 ")
	sb.WriteString("accesstype/ ")
	sb.WriteString(fmt.Sprintf("clientid/%s ", ClientID))
	sb.WriteString(fmt.Sprintf("clientversion/%s ", ClientVersion))
	sb.WriteString("action_type/ ")
	sb.WriteString("networktype/WIFI ")
	sb.WriteString("sessionid/ ")
	sb.WriteString(fmt.Sprintf("deviceid/%s ", deviceID))
	sb.WriteString("providername/NONE ")
	sb.WriteString(fmt.Sprintf("devicesign/%s ", deviceSign))
	sb.WriteString("refresh_token/ ")
	sb.WriteString(fmt.Sprintf("sdkversion/%s ", SDKVersion))
	sb.WriteString(fmt.Sprintf("datetime/%d ", GetTimestamp()))
	sb.WriteString(fmt.Sprintf("usrno/%s ", userID))
	sb.WriteString(fmt.Sprintf("appname/%s ", AppName))
	sb.WriteString("session_origin/ ")
	sb.WriteString("grant_type/ ")
	sb.WriteString("appid/ ")
	sb.WriteString("clientip/ ")
	sb.WriteString("devicename/Xiaomi_M2004j7ac ")
	sb.WriteString("osversion/13 ")
	sb.WriteString("platformversion/10 ")
	sb.WriteString("accessmode/ ")
	sb.WriteString("devicemodel/M2004J7AC")

	return sb.String()
}

// md5Hash 计算输入字符串的MD5哈希值
//
// 这是一个内部辅助函数，用于计算字符串的MD5哈希
// 返回32位十六进制小写字符串（标准的MD5输出格式）
//
// 参数说明：
//   - input: string 待哈希的输入字符串
//
// 返回值：
//   - string MD5哈希值的十六进制表示
//     - 固定长度：32个字符
//     - 小写字母
//
// 内部实现：
//   1. 将输入字符串转换为字节切片
//   2. 使用crypto/md5计算哈希
//   3. 将哈希结果编码为十六进制字符串
//
// 性能说明：
//   - MD5算法计算速度很快
//   - 适用于批处理场景
//
// 使用场景：
//   - CaptchaSign函数中的迭代哈希
//   - GenerateDeviceSign函数中的MD5计算
func md5Hash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}
