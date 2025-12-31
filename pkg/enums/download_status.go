// Package enums 提供了PikPak API相关的枚举类型定义
//
// 该包包含API中使用到的各种枚举类型：
//   - DownloadStatus: 离线下载任务状态
//
// 使用枚举的优势：
//   - 提供类型安全的常量定义
//   - 避免使用字符串时的拼写错误
//   - 便于代码阅读和维护
//
// 命名约定：
//   - 枚举类型使用PascalCase命名
//   - 枚举值使用全大写加下划线命名
//   - 每个枚举值应有明确的语义
package enums

// DownloadStatus 离线下载任务状态的枚举类型
//
// 该类型表示PikPak离线下载任务的当前状态
// 使用字符串类型别名实现枚举，便于序列化和比较
//
// 状态流转：
//   开始任务 -> DownloadStatusNotDownloading（等待中）
//             -> DownloadStatusDownloading（下载中）
//             -> DownloadStatusDone（完成）
//             或
//             -> DownloadStatusError（错误）
//
// 使用场景：
//   - 查询任务列表时获取每个任务的状态
//   - 根据状态过滤任务
//   - 展示任务进度
//
// 状态说明：
//   - not_downloading: 任务已创建，等待开始下载
//   - downloading: 任务正在下载中
//   - done: 任务已完成，文件已可访问
//   - error: 任务出错，需要重新创建或处理
//   - not_found: 任务不存在或已被删除
//
// 类型特点：
//   - 基于string类型，可直接与字符串比较
//   - 支持JSON序列化和反序列化
//   - 提供了类型安全的状态常量
type DownloadStatus string

// 下载状态常量定义
//
// 这些常量提供了类型安全的状态值
// 使用Go的iota特性自动生成枚举值
//
// 状态常量说明：
//   - DownloadStatusNotDownloading: 任务处于等待状态，尚未开始下载
//     - 通常在任务刚创建时出现
//     - 可能因为队列满、资源准备中等原因
//   - DownloadStatusDownloading: 任务正在下载中
//     - 文件正在从源地址下载到PikPak云端
//     - 可以查询下载进度
//   - DownloadStatusDone: 任务已完成
//     - 文件已成功下载到云端
//     - 文件已可访问和分享
//   - DownloadStatusError: 任务出错
//     - 下载过程中发生错误
//     - 可能的原因：链接失效、存储空间不足、版权限制等
//   - DownloadStatusNotFound: 任务不存在
//     - 指定的task_id不存在
//     - 任务可能已被删除或过期
const (
	DownloadStatusNotDownloading DownloadStatus = "not_downloading"
	DownloadStatusDownloading    DownloadStatus = "downloading"
	DownloadStatusDone           DownloadStatus = "done"
	DownloadStatusError          DownloadStatus = "error"
	DownloadStatusNotFound       DownloadStatus = "not_found"
)

// String 将DownloadStatus转换为字符串
//
// 实现fmt.Stringer接口
// 便于在日志输出和字符串拼接时使用
//
// 返回值：
//   - string 状态的字符串表示
//     - 返回枚举值对应的字符串
//     - 例如：DownloadStatusDownloading 返回 "downloading"
//
// 使用示例：
//
//	status := enums.DownloadStatusDone
//	fmt.Printf("任务状态: %s", status.String())
//	// 输出: 任务状态: done
func (s DownloadStatus) String() string {
	return string(s)
}

// ParseDownloadStatus 将字符串解析为DownloadStatus
//
// 该函数将API返回的状态字符串转换为枚举类型
// 如果遇到未知状态，返回DownloadStatusNotFound
//
// 参数说明：
//   - status: string API返回的状态字符串
//     - 例如："not_downloading", "downloading", "done"等
//     - 不区分大小写，但API通常返回小写
//
// 支持解析的状态：
//   - "not_downloading" -> DownloadStatusNotDownloading
//   - "downloading" -> DownloadStatusDownloading
//   - "done" -> DownloadStatusDone
//   - "error" -> DownloadStatusError
//   - "not_found" -> DownloadStatusNotFound
//   - 其他 -> DownloadStatusNotFound
//
// 返回值：
//   - DownloadStatus 对应的枚举常量
//     - 匹配到已知状态时返回对应的枚举
//     - 未知状态返回DownloadStatusNotFound
//
// 使用场景：
//   - 解析API响应中的状态字段
//   - 处理用户输入的状态选择
//   - 状态过滤和查询
func ParseDownloadStatus(status string) DownloadStatus {
	switch status {
	case "not_downloading":
		return DownloadStatusNotDownloading
	case "downloading":
		return DownloadStatusDownloading
	case "done":
		return DownloadStatusDone
	case "error":
		return DownloadStatusError
	case "not_found":
		return DownloadStatusNotFound
	default:
		return DownloadStatusNotFound
	}
}
