package user

import (
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// UserIDMappingModel 数据库查询专用结构体
//
// 说明：
//   - 用于数据库扫描操作（sqlx.QueryRow, sqlx.QueryRows）
//   - 包含 db 标签（允许框架类型）
//   - 通过 ToDomain() 方法转换为 core.UserIDMapping
//
// 数据库表：skylark_user_mappings
type UserIDMappingModel struct {
	ID           string    `db:"id"`             // UUID
	TenantID     string    `db:"tenant_id"`      // 租户ID
	LocalUserID  string    `db:"local_user_id"`  // 本地用户ID
	RemoteUserID int       `db:"remote_user_id"` // Skylark用户ID
	CreatedAt    time.Time `db:"created_at"`     // 创建时间
	UpdatedAt    time.Time `db:"updated_at"`     // 更新时间
}

// ToDomain 转换为领域模型
//
// 说明：
//   - 将数据库模型转换为 core.UserIDMapping（纯Go类型）
//   - 用于返回给调用方
//
// 返回：
//   - *core.UserIDMapping: 领域模型
func (m *UserIDMappingModel) ToDomain() *core.UserIDMapping {
	return &core.UserIDMapping{
		ID:           m.ID,
		TenantID:     m.TenantID,
		LocalUserID:  m.LocalUserID,
		RemoteUserID: m.RemoteUserID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// FromDomain 从领域模型转换
//
// 说明：
//   - 将 core.UserIDMapping 转换为数据库模型
//   - 用于插入数据库
//
// 参数：
//   - mapping: 领域模型
//
// 返回：
//   - *UserIDMappingModel: 数据库模型
func FromDomain(mapping *core.UserIDMapping) *UserIDMappingModel {
	return &UserIDMappingModel{
		ID:           mapping.ID,
		TenantID:     mapping.TenantID,
		LocalUserID:  mapping.LocalUserID,
		RemoteUserID: mapping.RemoteUserID,
		CreatedAt:    mapping.CreatedAt,
		UpdatedAt:    mapping.UpdatedAt,
	}
}
