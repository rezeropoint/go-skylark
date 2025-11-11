package organization

import (
	"time"

	"github.com/rezeropoint/go-skylark/core"
)

// OrgIDMappingModel 是数据库查询专用结构体（基础设施层）
// 职责：处理数据库 ORM 映射，对应 skylark_org_mappings 表
type OrgIDMappingModel struct {
	ID          string    `db:"id"`            // 映射UUID
	TenantID    string    `db:"tenant_id"`     // 租户ID
	LocalOrgID  string    `db:"local_org_id"`  // 本地组织ID
	RemoteOrgID int       `db:"remote_org_id"` // Skylark组织ID
	CreatedAt   time.Time `db:"created_at"`    // 创建时间
	UpdatedAt   time.Time `db:"updated_at"`    // 更新时间
}

// ToDomain 将数据库模型转换为领域模型
func (m *OrgIDMappingModel) ToDomain() *core.OrgIDMapping {
	return &core.OrgIDMapping{
		ID:          m.ID,
		TenantID:    m.TenantID,
		LocalOrgID:  m.LocalOrgID,
		RemoteOrgID: m.RemoteOrgID,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromDomain 将领域模型转换为数据库模型
func FromDomain(mapping *core.OrgIDMapping) *OrgIDMappingModel {
	return &OrgIDMappingModel{
		ID:          mapping.ID,
		TenantID:    mapping.TenantID,
		LocalOrgID:  mapping.LocalOrgID,
		RemoteOrgID: mapping.RemoteOrgID,
		CreatedAt:   mapping.CreatedAt,
		UpdatedAt:   mapping.UpdatedAt,
	}
}
