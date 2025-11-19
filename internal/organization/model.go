package organization

import (
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"
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

// OrganizationMemberModel 组织成员 API 响应模型
// 职责：处理 Skylark API 响应解析
type OrganizationMemberModel struct {
	ID         int     `json:"id"`                   // Skylark 用户ID
	Name       string  `json:"name"`                 // 用户姓名
	Nickname   *string `json:"nickname"`             // 昵称（可空）
	Sex        *string `json:"sex"`                  // 性别（可空）
	Phone      *string `json:"phone"`                // 电话号码（可空）
	Identifier string  `json:"identifier"`           // 识别码
	OpenID     *string `json:"openid"`               // 微信 OpenID（可空）
	Headimgurl *string `json:"headimgurl"`           // 头像URL（可空）
	CreatedAt  *string `json:"created_at,omitempty"` // 创建时间（可空）
	UpdatedAt  *string `json:"updated_at,omitempty"` // 更新时间（可空）
}

// ToDomain 将 API 响应模型转换为领域模型
func (m *OrganizationMemberModel) ToDomain() *core.OrganizationMember {
	member := &core.OrganizationMember{
		ID:         m.ID,
		Name:       m.Name,
		Nickname:   m.Nickname,
		Sex:        m.Sex,
		Phone:      m.Phone,
		Identifier: m.Identifier,
		OpenID:     m.OpenID,
		Headimgurl: m.Headimgurl,
	}

	// 解析时间字符串（可空）
	if m.CreatedAt != nil {
		if t, err := time.Parse(time.RFC3339, *m.CreatedAt); err == nil {
			member.CreatedAt = &t
		}
	}
	if m.UpdatedAt != nil {
		if t, err := time.Parse(time.RFC3339, *m.UpdatedAt); err == nil {
			member.UpdatedAt = &t
		}
	}

	return member
}

// AddMembersRequest 批量增加组织成员请求
type AddMembersRequest struct {
	MemberIDs []int `json:"member_ids"` // 成员ID列表
}

// RemoveMembersRequest 批量移除组织成员请求
type RemoveMembersRequest struct {
	MemberIDs []int `json:"member_ids"` // 成员ID列表
}

// ========== UpdateOrganization 内部使用的数据结构 ==========

// UpdateOrganizationBasicInfoRequest 更新组织基本信息请求（内部使用）
type UpdateOrganizationBasicInfoRequest struct {
	Name *string `json:"name,omitempty"` // 组织名称（可选）
}

// OrganizationAccessModel 组织管理员权限 API 响应模型（内部使用）
type OrganizationAccessModel struct {
	ID      int    `json:"id"`      // 权限ID
	Name    string `json:"name"`    // 权限名称
	Actions []int  `json:"actions"` // 操作权限列表
	Founded bool   `json:"founded"` // 是否为创建者
}

// OrganizationAdministratorModel 组织管理员 API 响应模型（内部使用）
type OrganizationAdministratorModel struct {
	ID             int                      `json:"id"`           // 管理员关联ID
	User           *OrganizationMemberModel `json:"user"`         // 用户信息（可空）
	Access         *OrganizationAccessModel `json:"access"`       // 权限信息（可空）
	OrganizationID int                      `json:"-"`            // 组织ID（从请求上下文获取，不在响应中）
	Organization   *map[string]interface{}  `json:"organization"` // 组织信息（仅用于 API 响应解析，不转换为领域模型）
}

// CreateAdministratorRequest 创建组织管理员请求（内部使用）
type CreateAdministratorRequest struct {
	UserID   int `json:"user_id"`   // 用户ID
	AccessID int `json:"access_id"` // 权限ID
}
