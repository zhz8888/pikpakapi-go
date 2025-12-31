package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/zhz8888/pikpakapi-go/internal/client"
)

// printJSON 是一个辅助函数，用于将数据格式化为易读的JSON字符串输出
// 参数:
//   - data: interface{} - 需要格式化的任意数据类型
//
// 该函数使用json.MarshalIndent进行格式化，以缩进的方式打印JSON，
// 便于调试和查看API返回的详细数据结构
func printJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// 格式化失败时记录错误但不中断程序
		log.Printf("Failed to marshal data: %v", err)
		return
	}
	fmt.Println(string(jsonData))
}

// main 程序的主入口函数
// 该函数演示了PikPak API客户端的完整使用流程：
// 1. 登录认证
// 2. 获取用户信息
// 3. 查询配额信息
// 4. 列出离线下载任务
// 5. 列出云端文件
// 6. 创建离线下载任务
//
// 使用前请将username和password替换为真实的PikPak账号信息
func main() {
	// 创建上下文，用于控制请求的超时和取消
	ctx := context.Background()

	// 初始化PikPak客户端
	// WithUsername: 设置用户名，支持邮箱、手机号或用户名
	// WithPassword: 设置密码
	// WithMaxRetries: 设置最大重试次数，默认为3次
	// WithTokenRefreshCallback: 设置令牌刷新时的回调函数
	cli := client.NewClient(
		client.WithUsername("your_username"),
		client.WithPassword("your_password"),
		client.WithMaxRetries(3),
		client.WithTokenRefreshCallback(func(c *client.Client) {
			// 令牌刷新成功后的回调函数
			// 可在此处保存新的令牌信息到配置文件或数据库
			log.Println("Token refreshed successfully!")
		}),
	)

	fmt.Println("=== Logging in... ===")
	// 执行登录操作
	// 登录过程包含：
	// 1. 初始化验证码挑战（Captcha Init）
	// 2. 获取验证码令牌（Captcha Token）
	// 3. 提交登录凭证
	// 4. 获取访问令牌（Access Token）和刷新令牌（Refresh Token）
	if err := cli.Login(ctx); err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Println("Login successful!")

	// 获取并展示用户信息
	// 返回的信息包括：用户名、用户ID、访问令牌、刷新令牌、编码令牌
	fmt.Println("\n=== User Info ===")
	userInfo := cli.GetUserInfo()
	printJSON(userInfo)

	// 获取账户配额信息
	// 返回云盘容量使用情况，包括：
	// - total_storage: 总存储空间
	// - used_storage: 已使用空间
	// - subscription_plan: 订阅计划信息
	fmt.Println("\n=== Quota Info ===")
	quota, err := cli.GetQuotaInfo(ctx)
	if err != nil {
		log.Printf("Failed to get quota info: %v", err)
	} else {
		printJSON(quota)
	}

	// 获取离线下载任务列表
	// 参数说明：
	// - 10: 每次请求返回的最大任务数量
	// - "": 页码令牌，首次请求传空字符串
	// - nil: 任务状态筛选器，nil表示默认筛选运行中和失败的任务
	// 支持的筛选状态：PHASE_TYPE_RUNNING, PHASE_TYPE_ERROR, PHASE_TYPE_COMPLETE, PHASE_TYPE_PENDING
	fmt.Println("\n=== Offline List ===")
	tasks, err := cli.OfflineList(ctx, 10, "", nil)
	if err != nil {
		log.Printf("Failed to get offline list: %v", err)
	} else {
		printJSON(tasks)
	}

	// 获取云端文件列表
	// 参数说明：
	// - 20: 每次请求返回的最大文件数量
	// - "": 父文件夹ID，空字符串表示根目录
	// - "": 页码令牌，首次请求传空字符串
	fmt.Println("\n=== File List ===")
	files, err := cli.FileList(ctx, 20, "", "")
	if err != nil {
		log.Printf("Failed to get file list: %v", err)
	} else {
		printJSON(files)
	}

	// 创建离线下载任务
	// 支持多种下载方式：
	// - HTTP/HTTPS链接直接下载
	// - 磁力链接（Magnet URI）
	// - BT种子文件
	// 参数说明：
	// - "magnet:?xt=urn:btih:...": 磁力链接
	// - "": 父文件夹ID，空字符串默认保存到"我的数据包"
	// - "Test Download": 自定义文件名，不传则自动从链接中提取
	fmt.Println("\n=== Offline Download (Magnet) ===")
	result, err := cli.OfflineDownload(ctx, "magnet:?xt=urn:btih:42b46b971332e776e8b290ed34632d5c81a1c47c", "", "Test Download")
	if err != nil {
		log.Printf("Failed to start offline download: %v", err)
	} else {
		printJSON(result)
	}

	fmt.Println("\n=== All operations completed successfully! ===")
}
