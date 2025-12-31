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
		client.WithUsername("your_username"),
		client.WithPassword("your_password"),
	)

	if err := cli.Login(ctx); err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	userInfo := cli.GetUserInfo()
	printJSON(userInfo)

	quota, _ := cli.GetQuotaInfo(ctx)
	printJSON(quota)

	files, _ := cli.FileList(ctx, 20, "", "")
	printJSON(files)
}
```

## API 文档

### 客户端初始化

```go
cli := client.NewClient(
	client.WithUsername("username"),
	client.WithPassword("password"),
	client.WithMaxRetries(3),
	client.WithInitialBackoff(2 * time.Second),
	client.WithTokenRefreshCallback(func(c *client.Client) {
		log.Println("Token refreshed!")
	}),
)
```

### 认证管理

```go
// 登录
if err := cli.Login(ctx); err != nil {
	log.Fatalf("Login failed: %v", err)
}

// 刷新访问令牌
if err := cli.RefreshAccessToken(ctx); err != nil {
	log.Fatalf("Refresh token failed: %v", err)
}

// 获取用户信息
userInfo := cli.GetUserInfo()
// userInfo包含: username, user_id, access_token, refresh_token, encoded_token

// 编码令牌（保存到配置文件）
if err := cli.EncodeToken(); err != nil {
	log.Fatal(err)
}
token := cli.encodedToken

// 解码令牌（从配置文件恢复）
cli.encodedToken = token
if err := cli.DecodeToken(); err != nil {
	log.Fatal(err)
}
```

### 用户信息

```go
// 获取账户配额信息
quota, err := cli.GetQuotaInfo(ctx)
// quota包含: total_storage, used_storage, subscription_plan
```

### 文件管理

```go
// 列出文件
files, err := cli.FileList(ctx, 20, "", "")
// 参数: size, parentID(空为根目录), nextPageToken

// 创建文件夹
result, err := cli.CreateFolder(ctx, "New Folder", "")

// 重命名文件
renamed, err := cli.FileRename(ctx, "file_id", "New Name")

// 收藏文件
starred, err := cli.FileBatchStar(ctx, []string{"file_id1", "file_id2"})

// 取消收藏
unstarred, err := cli.FileBatchUnstar(ctx, []string{"file_id1"})

// 获取收藏列表
stars, err := cli.FileStarList(ctx)

// 移动到回收站
trashed, err := cli.DeleteToTrash(ctx, []string{"file_id"})

// 从回收站恢复
restored, err := cli.Untrash(ctx, []string{"file_id"})

// 永久删除
deleted, err := cli.DeleteForever(ctx, []string{"file_id"})

// 上传文件（本地路径）
uploaded, err := cli.Upload(ctx, "/path/to/file.txt", "", "file.txt")

// 上传文件（流式）
file, err := os.Open("/path/to/file.txt")
defer file.Close()
uploaded, err := cli.UploadReader(ctx, file, "file.txt", "")

// 获取文件变更事件
events, err := cli.Events(ctx, 100, "")
```

### 离线下载

```go
// 创建离线下载任务（磁力链接）
result, err := cli.OfflineDownload(ctx, "magnet:?xt=urn:btih:...", "", "My Download")
// 参数: fileURL, parentID, name

// 创建离线下载任务（HTTP链接）
result, err := cli.OfflineDownload(ctx, "https://example.com/file.zip", "", "File Download")

// 创建离线下载任务（BT种子文件）
result, err := cli.OfflineDownload(ctx, "/path/to/torrent.torrent", "", "BT Download")

// 获取离线任务列表
tasks, err := cli.OfflineList(ctx, 10, "", nil)
// phases可选: "PHASE_TYPE_RUNNING", "PHASE_TYPE_ERROR", "PHASE_TYPE_COMPLETE", "PHASE_TYPE_PENDING"

// 获取任务状态
status, err := cli.GetTaskStatus(ctx, taskID, fileID)

// 重试失败任务
retried, err := cli.OfflineTaskRetry(ctx, taskID)

// 删除任务（可选是否删除文件）
err := cli.DeleteTasks(ctx, []string{taskID}, true)

// 获取离线文件详情
info, err := cli.OfflineFileInfo(ctx, fileID)
```

### 分享功能

```go
// 创建分享链接
share, err := cli.CreateShareLink(ctx, "file_id", false)
// needPassword: 是否需要提取码

// 批量分享文件
shared, err := cli.FileBatchShare(ctx, []string{"file_id1", "file_id2"}, false)

// 获取分享信息
info, err := cli.GetShareInfo(ctx, "https://www.mypikpak.com/s/xxx")

// 获取分享文件下载链接
downloadURL, err := cli.GetShareDownloadURL(ctx, "https://www.mypikpak.com/s/xxx", "file_id")

// 恢复分享文件
restored, err := cli.Restore(ctx, "share_id", "pass_code_token", []string{"file_id"})
```

## 构建

```bash
make          # 构建所有平台
make linux-amd64
make windows-amd64
```

## License

MIT

## Credit

[Quan666/PikPakAPI](https://github.com/Quan666/PikPakAPI)
