# PikPak API 文档

PikPak Drive API 的 Go 语言客户端实现详细文档。

## 目录

- [客户端初始化](#客户端初始化)
- [认证管理](#认证管理)
- [用户信息](#用户信息)
- [文件管理](#文件管理)
- [离线下载](#离线下载)
- [分享功能](#分享功能)

## 客户端初始化

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

### 配置选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `WithUsername` | string | - | 用户名，支持邮箱、手机号或用户名 |
| `WithPassword` | string | - | 密码 |
| `WithDeviceID` | string | 自动生成 | 设备标识符 |
| `WithBaseURL` | string | api-drive.mypikpak.com | API 服务器地址 |
| `WithAccessToken` | string | - | 访问令牌 |
| `WithRefreshToken` | string | - | 刷新令牌 |
| `WithMaxRetries` | int | 3 | 最大重试次数 |
| `WithInitialBackoff` | time.Duration | 3s | 重试初始退避时间 |
| `WithTokenRefreshCallback` | func(*Client) | nil | 令牌刷新回调函数 |

## 认证管理

### 登录

```go
if err := cli.Login(ctx); err != nil {
	log.Fatalf("Login failed: %v", err)
}
```

登录流程包括：
1. 初始化验证码挑战（Captcha Init）
2. 获取验证码令牌（Captcha Token）
3. 提交登录凭证
4. 获取访问令牌和刷新令牌

### 刷新访问令牌

```go
if err := cli.RefreshAccessToken(ctx); err != nil {
	log.Fatalf("Refresh token failed: %v", err)
}
```

当 accessToken 过期时，客户端会自动调用此方法刷新令牌。

### 获取用户信息

```go
userInfo := cli.GetUserInfo()
// userInfo 包含:
//   - username: 用户名
//   - user_id: 用户ID
//   - access_token: 访问令牌
//   - refresh_token: 刷新令牌
//   - encoded_token: 编码后的令牌（用于持久化保存）
```

### 编码令牌（保存到配置文件）

```go
if err := cli.EncodeToken(); err != nil {
	log.Fatal(err)
}
token := cli.encodedToken
// 将 token 保存到配置文件或数据库
```

### 解码令牌（从配置文件恢复）

```go
cli.encodedToken = token
if err := cli.DecodeToken(); err != nil {
	log.Fatal(err)
}
// 现在可以使用 cli 调用 API 方法
```

## 用户信息

### 获取账户配额信息

```go
quota, err := cli.GetQuotaInfo(ctx)
// quota 包含:
//   - total_storage: 总存储空间
//   - used_storage: 已使用空间
//   - subscription_plan: 订阅计划
```

### 获取存储详细信息

```go
storage, err := cli.GetStorageInfo(ctx)
// storage 包含:
//   - TotalBytes: 总存储空间（字节）
//   - UsedBytes: 已使用空间（字节）
//   - TrashBytes: 回收站占用空间
//   - IsUnlimited: 是否无限容量
//   - Complimentary: 附加服务类型
//   - ExpiresAt: 过期时间
//   - UserType: 用户类型
```

## 文件管理

### 列出文件

```go
files, err := cli.FileList(ctx, 20, "", "")
// 参数: size, parentID(空为根目录), nextPageToken, query(搜索关键词)
```

### 获取文件下载链接

```go
downloadURL, err := cli.GetFileLink(ctx, "file_id")
// 返回文件的直接下载链接
```

### 创建文件夹

```go
result, err := cli.CreateFolder(ctx, "New Folder", "")
// 参数: name, parentID(空为根目录)
```

### 重命名文件

```go
renamed, err := cli.Rename(ctx, "file_id", "New Name")
```

### 移动文件

```go
err := cli.Move(ctx, "file_id", "target_folder_id")
// 将文件移动到指定文件夹
```

### 复制文件

```go
err := cli.Copy(ctx, "file_id", "target_folder_id")
// 复制文件到指定文件夹
```

### 收藏文件

```go
starred, err := cli.FileBatchStar(ctx, []string{"file_id1", "file_id2"})
```

### 取消收藏

```go
unstarred, err := cli.FileBatchUnstar(ctx, []string{"file_id1"})
```

### 获取收藏列表

```go
stars, err := cli.FileStarList(ctx)
```

### 移动到回收站

```go
trashed, err := cli.DeleteToTrash(ctx, []string{"file_id"})
```

### 从回收站恢复

```go
restored, err := cli.Untrash(ctx, []string{"file_id"})
```

### 永久删除

```go
deleted, err := cli.DeleteForever(ctx, []string{"file_id"})
```

### 上传文件（本地路径）

```go
uploaded, err := cli.Upload(ctx, "/path/to/file.txt", "", "file.txt")
// 参数: filePath, parentID, fileName
```

### 上传文件（流式）

```go
file, err := os.Open("/path/to/file.txt")
defer file.Close()
uploaded, err := cli.UploadReader(ctx, file, "file.txt", "")
```

### 获取文件变更事件

```go
events, err := cli.Events(ctx, 100, "")
// 参数: size, nextPageToken
```

### 截图

```go
screenshot, err := cli.CaptureScreenshot(ctx, "file_id")
// 获取文件的截图
```

## 离线下载

### 创建离线下载任务（磁力链接）

```go
result, err := cli.OfflineDownload(ctx, "magnet:?xt=urn:btih:...", "", "My Download")
// 参数: fileURL, parentID, name
```

### 创建离线下载任务（HTTP链接）

```go
result, err := cli.OfflineDownload(ctx, "https://example.com/file.zip", "", "File Download")
```

### 创建离线下载任务（BT种子文件）

```go
result, err := cli.OfflineDownload(ctx, "/path/to/torrent.torrent", "", "BT Download")
```

### 创建远程下载任务

```go
result, err := cli.RemoteDownload(ctx, "https://example.com/file.zip")
```

### 获取离线任务列表

```go
tasks, err := cli.OfflineList(ctx, 10, "", nil)
// phases 可选:
//   - "PHASE_TYPE_RUNNING": 运行中
//   - "PHASE_TYPE_ERROR": 失败
//   - "PHASE_TYPE_COMPLETE": 完成
//   - "PHASE_TYPE_PENDING": 等待中
```

### 获取任务状态

```go
status, err := cli.GetTaskStatus(ctx, taskID, fileID)
// 返回枚举值:
//   - enums.DownloadStatusPending
//   - enums.DownloadStatusInProgress
//   - enums.DownloadStatusCompleted
//   - enums.DownloadStatusError
//   - enums.DownloadStatusPaused
```

### 获取离线文件详情

```go
info, err := cli.OfflineFileInfo(ctx, fileID)
```

### 重试失败任务

```go
retried, err := cli.OfflineTaskRetry(ctx, taskID)
```

### 删除任务（不删除文件）

```go
err := cli.DeleteTasks(ctx, []string{taskID}, false)
```

### 删除任务（同时删除文件）

```go
err := cli.DeleteTasks(ctx, []string{taskID}, true)
```

### 删除离线任务

```go
err := cli.DeleteOfflineTasks(ctx, []string{taskID}, deleteFiles)
// deleteFiles: 是否同时删除已下载的文件
```

## 分享功能

### 创建分享链接

```go
share, err := cli.CreateShareLink(ctx, "file_id", false)
// 参数: fileID, needPassword(是否需要提取码)
```

### 批量分享文件

```go
shared, err := cli.FileBatchShare(ctx, []string{"file_id1", "file_id2"}, false)
```

### 获取分享信息

```go
info, err := cli.GetShareInfo(ctx, "https://www.mypikpak.com/s/xxx")
```

### 获取分享文件下载链接

```go
downloadURL, err := cli.GetShareDownloadURL(ctx, "https://www.mypikpak.com/s/xxx", "file_id")
```

### 恢复分享文件

```go
restored, err := cli.Restore(ctx, "share_id", "pass_code_token", []string{"file_id"})
```

## 错误处理

所有 API 方法返回的错误类型为 `*exception.PikpakException`，包含以下信息：

```go
err := cli.Login(ctx)
if err != nil {
	if pe, ok := err.(*exception.PikpakException); ok {
		log.Printf("Error Code: %d", pe.ErrorCode)
		log.Printf("Error Description: %s", pe.ErrorDescription)
	}
}
```

## 使用示例

完整的示例程序请参考 [cmd/example/main.go](cmd/example/main.go)。
