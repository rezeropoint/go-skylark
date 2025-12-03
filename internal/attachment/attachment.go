// Package attachment 提供 Skylark 附件管理功能
//
// 职责：管理 Skylark 附件的 Base64 获取和业务数据转换
// 用途：在 query、flows 模块返回数据前，将附件字段转换为 Base64
package attachment

import (
	"context"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// Manager 附件管理器接口
// 职责：管理 Skylark 附件的 Base64 获取和转换
type Manager interface {
	// GetBase64 获取单个附件的 Base64 内容
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID（用于获取平台配置）
	//   - attachmentID: 附件ID
	// 返回：
	//   - Base64 编码的文件内容
	//   - 错误信息
	GetBase64(ctx context.Context, tenantID string, attachmentID int64) (string, error)

	// ConvertBusinessData 转换业务数据中的附件为 Base64
	// 该方法实现 core.ConvertBusinessDataAttachmentsFunc 签名
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - data: 业务数据（会被原地修改）
	// 说明：
	//   - 遍历 data 中的所有字段
	//   - 识别 Skylark 附件格式的字段值（包含 gid://skylark/Attachment/ 前缀）
	//   - 调用 API 获取 Base64 并替换原值
	//   - 单个附件获取失败时，该字段值设为 nil
	ConvertBusinessData(ctx context.Context, tenantID string, data map[string]interface{}) error
}

// NewManager 创建附件管理器
// 参数：
//   - getPlatformConfig: 获取平台配置的函数（由 Platform Manager 提供）
func NewManager(getPlatformConfig core.GetPlatformConfigFunc) (Manager, error) {
	if getPlatformConfig == nil {
		return nil, fmt.Errorf("getPlatformConfig 不能为 nil")
	}
	return &attachmentManager{
		getPlatformConfig: getPlatformConfig,
	}, nil
}
