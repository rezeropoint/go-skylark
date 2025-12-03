// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🔵 API 请求 - 附件相关
// 说明：定义 Skylark 附件数据结构和依赖注入函数类型
package core

import "context"

// API 路径常量
const (
	// APIAttachmentBase64Path 附件 Base64 API 路径前缀
	// 完整格式：/api/v4/attachments/:id/base64_file
	APIAttachmentBase64Path = "/api/v4/attachments/"
)

// SkylarkAttachmentGIDPrefix Skylark 附件 GID 前缀
// 用于识别字段值是否为图片/附件类型
const SkylarkAttachmentGIDPrefix = "gid://skylark/Attachment/"

// AttachmentValue Skylark 附件值结构（用于识别和解析）
// 说明：Skylark 图片字段在 value 数组中的元素格式
// 示例：{"gid": "gid://skylark/Attachment/6734", "id": 6734, "value": "image.jpeg"}
type AttachmentValue struct {
	GID   string `json:"gid"`   // 全局ID，格式：gid://skylark/Attachment/{id}
	ID    int64  `json:"id"`    // 附件ID（用于调用 API）
	Value string `json:"value"` // 文件名
}

// ConvertBusinessDataAttachmentsFunc 转换业务数据中的附件为 Base64 的函数类型
// 用途：依赖注入，供 query、flows 模块批量转换业务数据中的图片字段
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - data: 业务数据（会被原地修改）
//
// 说明：
//   - 遍历 data 中的所有字段
//   - 识别 Skylark 附件格式的字段值
//   - 调用 API 获取 Base64 并替换原值
//   - 单个附件获取失败时，该字段值设为 null
type ConvertBusinessDataAttachmentsFunc func(ctx context.Context, tenantID string, data map[string]interface{}) error
