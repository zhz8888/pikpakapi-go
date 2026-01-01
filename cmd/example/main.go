package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/zhz8888/pikpakapi-go/internal/client"
	"github.com/zhz8888/pikpakapi-go/pkg/enums"
)

func printJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return
	}
	fmt.Println(string(jsonData))
}

func main() {
	ctx := context.Background()

	cli := client.NewClient(
		client.WithUsername("your_username"),
		client.WithPassword("your_password"),
		client.WithMaxRetries(3),
		client.WithTokenRefreshCallback(func(c *client.Client) {
			log.Println("Token refreshed successfully!")
		}),
	)

	fmt.Println("=== 登录认证 ===")
	if err := cli.Login(ctx); err != nil {
		log.Fatalf("登录失败: %v", err)
	}
	fmt.Println("登录成功!")

	fmt.Println("\n=== 用户信息 ===")
	userInfo := cli.GetUserInfo()
	printJSON(userInfo)

	fmt.Println("\n=== 账户配额 ===")
	quota, err := cli.GetQuotaInfo(ctx)
	if err != nil {
		log.Printf("获取配额失败: %v", err)
	} else {
		printJSON(quota)
	}

	fmt.Println("\n=== 存储详情 ===")
	storage, err := cli.GetStorageInfo(ctx)
	if err != nil {
		log.Printf("获取存储信息失败: %v", err)
	} else {
		fmt.Printf("总空间: %.2f GB\n", float64(storage.TotalBytes)/(1024*1024*1024))
		fmt.Printf("已用空间: %.2f GB\n", float64(storage.UsedBytes)/(1024*1024*1024))
		fmt.Printf("无限容量: %v\n", storage.IsUnlimited)
	}

	fmt.Println("\n=== 文件列表 ===")
	files, err := cli.FileList(ctx, 20, "", "", "")
	if err != nil {
		log.Printf("获取文件列表失败: %v", err)
	} else {
		printJSON(files)
	}

	fmt.Println("\n=== 创建文件夹 ===")
	folder, err := cli.CreateFolder(ctx, "测试文件夹", "")
	if err != nil {
		log.Printf("创建文件夹失败: %v", err)
	} else {
		printJSON(folder)
	}

	fmt.Println("\n=== 离线任务列表 ===")
	tasks, err := cli.OfflineList(ctx, 10, "", nil)
	if err != nil {
		log.Printf("获取离线任务失败: %v", err)
	} else {
		printJSON(tasks)
	}

	fmt.Println("\n=== 创建离线下载任务 (磁力链接) ===")
	downloadResult, err := cli.OfflineDownload(ctx, "magnet:?xt=urn:btih:42b46b971332e776e8b290ed34632d5c81a1c47c", "", "测试下载")
	if err != nil {
		log.Printf("创建离线下载失败: %v", err)
	} else {
		printJSON(downloadResult)
	}

	fmt.Println("\n=== 创建离线下载任务 (HTTP链接) ===")
	httpResult, err := cli.OfflineDownload(ctx, "https://example.com/file.zip", "", "HTTP下载测试")
	if err != nil {
		log.Printf("创建HTTP下载失败: %v", err)
	} else {
		printJSON(httpResult)
	}

	fmt.Println("\n=== 远程下载任务 ===")
	remoteResult, err := cli.RemoteDownload(ctx, "https://example.com/file.torrent")
	if err != nil {
		log.Printf("创建远程下载失败: %v", err)
	} else {
		printJSON(remoteResult)
	}

	fmt.Println("\n=== 获取任务状态 ===")
	status, err := cli.GetTaskStatus(ctx, "task_id", "file_id")
	if err != nil {
		log.Printf("获取任务状态失败: %v", err)
	} else {
		fmt.Printf("任务状态: %s\n", status.String())
		switch status {
		case enums.DownloadStatusNotDownloading:
			fmt.Println("任务正在等待处理")
		case enums.DownloadStatusDownloading:
			fmt.Println("任务正在下载中")
		case enums.DownloadStatusDone:
			fmt.Println("任务已完成")
		case enums.DownloadStatusError:
			fmt.Println("任务下载失败")
		}
	}

	fmt.Println("\n=== 重试失败任务 ===")
	retryResult, err := cli.OfflineTaskRetry(ctx, "task_id")
	if err != nil {
		log.Printf("重试任务失败: %v", err)
	} else {
		printJSON(retryResult)
	}

	fmt.Println("\n=== 文件操作演示 ===")

	fileID := "your_file_id"
	newFileID := "your_new_file_id"

	fmt.Println("\n--- 重命名文件 ---")
	renameResult, err := cli.FileRename(ctx, fileID, "新文件名")
	if err != nil {
		log.Printf("重命名失败: %v", err)
	} else {
		printJSON(renameResult)
	}

	fmt.Println("\n--- 移动文件 ---")
	if err = cli.Move(ctx, fileID, "parent_folder_id"); err != nil {
		log.Printf("移动文件失败: %v", err)
	} else {
		fmt.Println("文件移动成功")
	}

	fmt.Println("\n--- 复制文件 ---")
	if err = cli.Copy(ctx, fileID, "target_folder_id"); err != nil {
		log.Printf("复制文件失败: %v", err)
	} else {
		fmt.Println("文件复制成功")
	}

	fmt.Println("\n--- 收藏文件 ---")
	starResult, err := cli.FileBatchStar(ctx, []string{fileID})
	if err != nil {
		log.Printf("收藏文件失败: %v", err)
	} else {
		printJSON(starResult)
	}

	fmt.Println("\n--- 收藏列表 ---")
	starList, err := cli.FileStarList(ctx)
	if err != nil {
		log.Printf("获取收藏列表失败: %v", err)
	} else {
		printJSON(starList)
	}

	fmt.Println("\n--- 取消收藏 ---")
	unstarResult, err := cli.FileBatchUnstar(ctx, []string{fileID})
	if err != nil {
		log.Printf("取消收藏失败: %v", err)
	} else {
		printJSON(unstarResult)
	}

	fmt.Println("\n--- 获取文件下载链接 ---")
	downloadURL, err := cli.GetFileLink(ctx, fileID)
	if err != nil {
		log.Printf("获取下载链接失败: %v", err)
	} else {
		fmt.Printf("下载链接: %s\n", downloadURL)
	}

	fmt.Println("\n--- 上传文件 (本地路径) ---")
	uploadResult, err := cli.Upload(ctx, "/path/to/local/file.txt", "", "上传的文件.txt")
	if err != nil {
		log.Printf("上传文件失败: %v", err)
	} else {
		printJSON(uploadResult)
	}

	fmt.Println("\n--- 上传文件 (流式) ---")
	file, err := os.Open("/path/to/local/file.txt")
	if err != nil {
		log.Printf("打开文件失败: %v", err)
	} else {
		uploadReaderResult, uploadErr := cli.UploadReader(ctx, file, "流式上传.txt", "")
		if uploadErr != nil {
			log.Printf("流式上传失败: %v", uploadErr)
		} else {
			printJSON(uploadReaderResult)
		}
		file.Close()
	}

	fmt.Println("\n--- 文件变更事件 ---")
	events, err := cli.Events(ctx, 50, "")
	if err != nil {
		log.Printf("获取事件失败: %v", err)
	} else {
		printJSON(events)
	}

	fmt.Println("\n--- 移动到回收站 ---")
	trashResult, err := cli.DeleteToTrash(ctx, []string{fileID})
	if err != nil {
		log.Printf("移动到回收站失败: %v", err)
	} else {
		printJSON(trashResult)
	}

	fmt.Println("\n--- 从回收站恢复 ---")
	untrashResult, err := cli.Untrash(ctx, []string{newFileID})
	if err != nil {
		log.Printf("恢复文件失败: %v", err)
	} else {
		printJSON(untrashResult)
	}

	fmt.Println("\n--- 永久删除 ---")
	deleteResult, err := cli.DeleteForever(ctx, []string{fileID})
	if err != nil {
		log.Printf("永久删除失败: %v", err)
	} else {
		printJSON(deleteResult)
	}

	fmt.Println("\n=== 分享功能演示 ===")

	fmt.Println("\n--- 创建分享链接 ---")
	shareResult, err := cli.CreateShareLink(ctx, fileID, false)
	if err != nil {
		log.Printf("创建分享链接失败: %v", err)
	} else {
		printJSON(shareResult)
	}

	fmt.Println("\n--- 批量分享文件 ---")
	batchShareResult, err := cli.FileBatchShare(ctx, []string{fileID, newFileID}, false)
	if err != nil {
		log.Printf("批量分享失败: %v", err)
	} else {
		printJSON(batchShareResult)
	}

	fmt.Println("\n--- 获取分享信息 ---")
	shareInfo, err := cli.GetShareInfo(ctx, "https://www.mypikpak.com/s/xxxxxx")
	if err != nil {
		log.Printf("获取分享信息失败: %v", err)
	} else {
		printJSON(shareInfo)
	}

	fmt.Println("\n--- 获取分享文件下载链接 ---")
	shareDownloadURL, err := cli.GetShareDownloadURL(ctx, "https://www.mypikpak.com/s/xxxxxx", fileID)
	if err != nil {
		log.Printf("获取分享下载链接失败: %v", err)
	} else {
		fmt.Printf("分享下载链接: %s\n", shareDownloadURL)
	}

	fmt.Println("\n--- 恢复分享文件 ---")
	restoreResult, err := cli.Restore(ctx, "share_id", "pass_code_token", []string{fileID})
	if err != nil {
		log.Printf("恢复分享文件失败: %v", err)
	} else {
		printJSON(restoreResult)
	}

	fmt.Println("\n=== 分享链接功能演示 ===")

	var shareFileInfo *client.ShareFileInfo
	var shareFileInfoWithPwd *client.ShareFileInfo
	var shareFiles []*client.ShareFileInfo
	var shareTranscodedURL string

	fmt.Println("\n--- 获取分享链接文件信息 (无密码) ---")
	shareFileInfo, err = cli.GetShareFileInfo(ctx, "https://pan.pikpak.com/share/link/xxxxxx", "")
	if err != nil {
		log.Printf("获取分享文件信息失败: %v", err)
	} else {
		fmt.Printf("文件名: %s\n", shareFileInfo.Name)
		fmt.Printf("大小: %s\n", shareFileInfo.Size)
		fmt.Printf("文件类型: %s\n", shareFileInfo.Kind)
		if shareFileInfo.WebContentLink != "" {
			fmt.Printf("Web下载链接: %s\n", shareFileInfo.WebContentLink)
		}
		if len(shareFileInfo.Medias) > 0 {
			fmt.Printf("媒体数量: %d\n", len(shareFileInfo.Medias))
			for i, media := range shareFileInfo.Medias {
				fmt.Printf("  媒体%d: %s\n", i+1, media.MediaName)
				fmt.Printf("    分辨率: %s\n", media.ResolutionName)
				fmt.Printf("    视频: %dx%d\n", media.Video.Width, media.Video.Height)
			}
		}
	}

	fmt.Println("\n--- 获取分享链接文件信息 (有密码) ---")
	shareFileInfoWithPwd, err = cli.GetShareFileInfo(ctx, "https://pan.pikpak.com/share/link/xxxxxx", "password123")
	if err != nil {
		log.Printf("获取分享文件信息失败: %v", err)
	} else {
		fmt.Printf("文件名: %s\n", shareFileInfoWithPwd.Name)
		fmt.Printf("大小: %s\n", shareFileInfoWithPwd.Size)
	}

	fmt.Println("\n--- 获取分享文件下载链接 (原画) ---")
	shareDownloadURL, err = cli.GetShareFileDownloadURL(ctx, "https://pan.pikpak.com/share/link/xxxxxx", "", false)
	if err != nil {
		log.Printf("获取分享下载链接失败: %v", err)
	} else {
		fmt.Printf("原画下载链接: %s\n", shareDownloadURL)
	}

	fmt.Println("\n--- 获取分享文件下载链接 (转码高清) ---")
	shareTranscodedURL, err = cli.GetShareFileDownloadURL(ctx, "https://pan.pikpak.com/share/link/xxxxxx", "", true)
	if err != nil {
		log.Printf("获取转码下载链接失败: %v", err)
	} else {
		fmt.Printf("转码高清下载链接: %s\n", shareTranscodedURL)
	}

	fmt.Println("\n--- 获取分享链接文件列表 ---")
	shareFiles, err = cli.GetShareFiles(ctx, "https://pan.pikpak.com/share/link/xxxxxx", "")
	if err != nil {
		log.Printf("获取分享文件列表失败: %v", err)
	} else {
		fmt.Printf("分享中包含 %d 个文件/文件夹:\n", len(shareFiles))
		for i, file := range shareFiles {
			isFolder := file.Kind == "drive#folder"
			fmt.Printf("  %d. %s %s\n", i+1, file.Name, map[bool]string{true: "(文件夹)", false: ""}[isFolder])
		}
	}

	fmt.Println("\n=== 令牌管理演示 ===")

	fmt.Println("\n--- 编码令牌 (保存到配置文件) ---")
	if err := cli.EncodeToken(); err != nil {
		log.Printf("编码令牌失败: %v", err)
	} else {
		fmt.Printf("编码后的令牌: %s\n", cli.GetEncodedToken())
	}

	fmt.Println("\n--- 解码令牌 (从配置文件恢复) ---")
	cli.SetEncodedToken("your_saved_token")
	if err := cli.DecodeToken(); err != nil {
		log.Printf("解码令牌失败: %v", err)
	} else {
		fmt.Println("令牌解码成功")
	}

	fmt.Println("\n--- 刷新访问令牌 ---")
	if err := cli.RefreshAccessToken(ctx); err != nil {
		log.Printf("刷新令牌失败: %v", err)
	} else {
		fmt.Println("令牌刷新成功")
	}

	fmt.Println("\n=== 所有操作完成 ===")
}
