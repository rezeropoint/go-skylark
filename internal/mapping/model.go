package mapping

import (
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// OrgMappingModel 是数据库查询专用结构体（基础设施层）
// 职责：处理数据库 ORM 映射
type OrgMappingModel struct {
	ID             string    `db:"id"`               // 映射UUID
	RemoteOrgValue string    `db:"remote_org_value"` // 远程表组织字段的值
	LocalOrgID     string    `db:"local_org_id"`     // 本地组织ID
	LocalOrgName   string    `db:"local_org_name"`   // 本地组织名称（查询时JOIN获取）
	TenantID       string    `db:"tenant_id"`        // 租户ID
	CreatedAt      time.Time `db:"created_at"`       // 创建时间
	UpdatedAt      time.Time `db:"updated_at"`       // 更新时间
}

// ToDomain 将数据库模型转换为领域模型
func (m *OrgMappingModel) ToDomain() *core.OrgMapping {
	return &core.OrgMapping{
		ID:             m.ID,
		RemoteOrgValue: m.RemoteOrgValue,
		LocalOrgID:     m.LocalOrgID,
		LocalOrgName:   m.LocalOrgName,
		TenantID:       m.TenantID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// FromDomain 将领域模型转换为数据库模型
func FromDomain(mapping *core.OrgMapping) *OrgMappingModel {
	return &OrgMappingModel{
		ID:             mapping.ID,
		RemoteOrgValue: mapping.RemoteOrgValue,
		LocalOrgID:     mapping.LocalOrgID,
		LocalOrgName:   mapping.LocalOrgName,
		TenantID:       mapping.TenantID,
		CreatedAt:      mapping.CreatedAt,
		UpdatedAt:      mapping.UpdatedAt,
	}
}
