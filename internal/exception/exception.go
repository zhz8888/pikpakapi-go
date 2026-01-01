// Package exception 提供了PikPak API客户端的异常处理定义
//
// 该包定义了客户端特定的错误类型和错误处理函数：
//   - PikpakException: 封装了PikPak API返回的错误信息
//   - 预定义的错误变量: 用于常见的错误场景
//   - 错误判断函数: 用于识别特定类型的错误
//
// 错误处理策略：
//   - 使用自定义错误类型包装底层错误
//   - 保留原始错误信息用于调试
//   - 提供预定义错误避免硬编码错误字符串
//
// 使用示例：
//
//	if err := cli.Login(ctx); err != nil {
//	    if exception.IsPikpakException(err) {
//	        var pe *exception.PikpakException
//	        if errors.As(err, &pe) {
//	            log.Printf("PikPak错误: %s", pe.Message)
//	        }
//	    }
//	}
package exception

import (
	"errors"
	"fmt"
)

// PikpakException PikPak API错误类型
//
// 该结构体封装了PikPak API返回的错误信息
// 支持错误消息和底层错误的链式包装
//
// 字段说明：
//   - Message: string 错误消息描述
//     - 描述错误的类型和原因
//     - 人类可读的错误信息
//   - Err: error 底层错误
//     - 原始的错误对象
//     - 用于错误链追踪和调试
//
// 错误格式化：
//   - 如果Err不为nil，Error()返回"Message: Err"
//   - 如果Err为nil，Error()只返回Message
//
// 线程安全性：
//   - PikpakException是不可变类型，线程安全
//
// 使用场景：
//   - API请求失败的错误响应
//   - 网络错误或超时
//   - 认证失败
//   - 参数验证失败
type PikpakException struct {
	Message string
	Err     error
}

// Error 实现error接口，返回错误的字符串描述
//
// 返回格式：
//   - 如果底层错误不为空："Message: Err"
//   - 如果底层错误为空："Message"
//
// 该方法用于满足Go的error接口要求
// 使得PikpakException可以作为error类型使用
//
// 返回值：
//   - string 格式化的错误描述字符串
func (e *PikpakException) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap 实现errors.Wrapper接口，返回底层错误
//
// 用于支持Go 1.13+的错误链功能
// 允许使用errors.Is和errors.As进行错误匹配
//
// 返回值：
//   - error 底层错误对象，如果为nil则返回nil
//
// 使用示例：
//
//	err := NewPikpakExceptionWithError("登录失败", originalErr)
//	if errors.Is(err, context.DeadlineExceeded) {
//	    // 处理超时错误
//	}
func (e *PikpakException) Unwrap() error {
	return e.Err
}

// NewPikpakException 创建只包含消息的PikpakException
//
// 用于创建只有错误消息，没有底层错误的异常
// 适用于可以直接描述的错误场景
//
// 参数说明：
//   - message: string 错误消息
//     - 描述错误的类型和原因
//     - 应该简洁明了
//
// 返回值：
//   - *PikpakException 创建的异常对象
//
// 使用场景：
//   - 参数验证失败
//   - 状态不正确的错误
//   - 业务逻辑错误
func NewPikpakException(message string) *PikpakException {
	return &PikpakException{Message: message}
}

// NewPikpakExceptionWithError 创建包含消息和底层错误的PikpakException
//
// 用于创建带有原始错误的异常
// 可以保留错误的上下文信息，便于调试
//
// 参数说明：
//   - message: string 错误消息
//     - 描述错误的类型和原因
//   - err: error 底层错误
//     - 原始的错误对象
//     - 可以是任意实现了error接口的类型
//
// 返回值：
//   - *PikpakException 创建的异常对象
//
// 使用场景：
//   - API请求失败，保留网络错误信息
//   - JSON序列化/反序列化失败
//   - 文件操作失败
//
// 错误链示例：
//
//	err := NewPikpakExceptionWithError("登录失败", originalErr)
//	fmt.Println(err)        // 登录失败: 网络超时
//	fmt.Println(err.Err)    // 网络超时
func NewPikpakExceptionWithError(message string, err error) *PikpakException {
	return &PikpakException{Message: message, Err: err}
}

// IsPikpakException 判断错误是否为PikpakException类型
//
// 使用errors.As进行类型断言
// 用于在错误处理时识别PikPak特定的错误
//
// 参数说明：
//   - err: error 待检查的错误对象
//     - 可以是任意error类型
//     - nil输入返回false
//
// 返回值：
//   - bool 如果错误是PikpakException类型返回true，否则返回false
//
// 使用示例：
//
//	err := cli.Login(ctx)
//	if exception.IsPikpakException(err) {
//	    log.Printf("发生PikPak相关错误: %v", err)
//	    // 可以进一步使用errors.As获取详细信息
//	}
func IsPikpakException(err error) bool {
	var pe *PikpakException
	return errors.As(err, &pe)
}

// 预定义的PikPak API错误变量
//
// 这些错误变量用于常见的错误场景，避免硬编码错误字符串
// 使用预定义错误可以保持错误处理的一致性
//
// 错误说明：
//   - ErrInvalidUsernamePassword: 用户名或密码无效
//     - 通常在登录时，用户名或密码错误时返回
//     - HTTP状态码：401
//   - ErrInvalidEncodedToken: 编码令牌无效
//     - 当解码保存的令牌失败时返回
//     - 可能是令牌已过期或格式损坏
//   - ErrCaptchaTokenFailed: 获取验证码令牌失败
//     - 在登录时，如果验证码验证失败返回
//     - 可能需要重新获取验证码
//   - ErrUsernamePasswordRequired: 用户名和密码为必填项
//     - 登录前未设置用户名或密码时返回
//     - 客户端参数验证错误
//   - ErrMaxRetriesReached: 达到最大重试次数
//     - HTTP请求在多次重试后仍然失败
//     - 可能表示网络问题或服务器不可用
//   - ErrUnknownError: 未知错误
//     - 无法分类的错误类型
//     - 需要进一步调查错误原因
//   - ErrEmptyJSONData: JSON数据为空
//     - API返回空数据或JSON解析失败
//     - 可能表示服务器响应异常
//   - ErrInvalidFileID: 文件ID无效
//     - 当文件ID为空或格式不正确时返回
//     - 用于文件操作前的参数验证
//   - ErrInvalidFileName: 文件名为空
//     - 当新文件名为空时返回
//     - 用于重命名操作时的参数验证
//   - ErrEmptyFileIDs: 文件ID列表为空
//     - 当批量操作的ID列表为空时返回
//     - 用于批量删除等操作前的参数验证
var (
	ErrInvalidUsernamePassword = NewPikpakException("invalid username or password")
	ErrInvalidEncodedToken     = NewPikpakException("invalid encoded token")
	ErrCaptchaTokenFailed      = NewPikpakException("captcha_token get failed")
	ErrUsernamePasswordRequired = NewPikpakException("username and password are required")
	ErrMaxRetriesReached       = NewPikpakException("max retries reached")
	ErrUnknownError            = NewPikpakException("unknown error")
	ErrEmptyJSONData           = NewPikpakException("empty JSON data")
	ErrInvalidFileID           = NewPikpakException("invalid file id")
	ErrInvalidFileName         = NewPikpakException("invalid file name")
	ErrEmptyFileIDs            = NewPikpakException("file ids is empty")
	ErrInvalidURL              = NewPikpakException("invalid url")
)
