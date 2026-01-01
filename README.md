# pikpakapi-go

PikPak Drive API 的 Go 语言客户端实现。

## 功能特性

- **认证管理** - 支持账号密码登录，自动令牌刷新
- **用户信息** - 获取用户资料和账户配额
- **文件管理** - 列出、搜索、管理云端文件
- **离线下载** - 支持 HTTP/HTTPS 链接、磁力链接、BT 种子
- **分享功能** - 创建和管理文件分享链接

## 安装

```bash
go get github.com/zhz8888/pikpakapi-go
```

## 快速开始

```go
package main

import (
	"context"
	"log"

	"github.com/zhz8888/pikpakapi-go/internal/client"
)

func main() {
	ctx := context.Background()

	cli := client.NewClient(
		client.WithUsername("your_email@example.com"),
		client.WithPassword("your_password"),
		client.WithMaxRetries(3),
		client.WithTokenRefreshCallback(func(c *client.Client) {
			log.Println("Token refreshed successfully!")
		}),
	)

	if err := cli.Login(ctx); err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	userInfo := cli.GetUserInfo()
	log.Printf("User: %s, UserID: %s", userInfo["username"], userInfo["user_id"])

	quota, err := cli.GetQuotaInfo(ctx)
	if err != nil {
		log.Fatalf("Get quota failed: %v", err)
	}
	log.Printf("Quota: %+v", quota)

	files, err := cli.FileList(ctx, 20, "", "")
	if err != nil {
		log.Fatalf("List files failed: %v", err)
	}
	log.Printf("Files: %+v", files)
}
```

## 项目结构

```
pikpakapi-go/
├── cmd/
│   └── example/          # 示例程序
│       └── main.go
├── internal/
│   ├── client/           # API 客户端核心实现
│   │   ├── client.go
│   │   └── client_test.go
│   ├── config/           # 配置管理
│   │   ├── config.go
│   │   └── config_test.go
│   ├── exception/        # 异常处理
│   │   ├── exception.go
│   │   └── exception_test.go
│   └── utils/            # 工具函数
│       ├── token.go
│       ├── token_test.go
│       ├── utils.go
│       └── utils_test.go
├── pkg/
│   └── enums/            # 枚举定义
│       ├── download_status.go
│       └── download_status_test.go
├── API.md                # 详细 API 文档
├── Makefile              # 构建脚本
├── go.mod
└── LICENSE
```

## 核心组件

### Client 客户端

`internal/client/client.go` 包含了与 PikPak API 交互的核心客户端实现：

- **认证** - 登录、令牌刷新、验证码处理
- **文件操作** - 创建文件夹、删除、重命名、收藏、分享
- **离线下载** - 创建下载任务、查询状态、任务管理
- **配额查询** - 获取账户存储配额信息
- **分享管理** - 创建分享链接、恢复分享文件

### 配置选项

```go
cli := client.NewClient(
	client.WithUsername("username"),
	client.WithPassword("password"),
	client.WithMaxRetries(3),                    // 最大重试次数（默认3次）
	client.WithInitialBackoff(2 * time.Second), // 重试初始退避时间（默认3秒）
	client.WithTokenRefreshCallback(func(c *client.Client) {
		log.Println("Token refreshed!")
	}),
)
```

## API 文档

详细 API 文档请参考 [API.md](API.md)。

## 构建

```bash
make          # 构建所有平台
make linux-amd64
make darwin-amd64
make windows-amd64
```

## 测试

```bash
go test ./...
```

## License

MIT

## Credit

[Quan666/PikPakAPI](https://github.com/Quan666/PikPakAPI)
