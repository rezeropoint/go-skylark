package platform

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/rezeropoint/go-skylark/core"
)

// PlatformConfigModel 是数据库查询专用结构体（基础设施层）
// 职责：处理数据库 ORM 映射，允许使用框架类型
type PlatformConfigModel struct {
	ID          string         `db:"id"`           // 配置UUID
	TenantID    string         `db:"tenant_id"`    // 租户ID（一租户一平台）
	Host        string         `db:"host"`         // 数据库地址（如：110.41.35.134）
	Port        int            `db:"port"`         // 端口（如：5432）
	Database    string         `db:"database"`     // 数据库名（如：sync）
	Username    string         `db:"username"`     // 用户名
	Password    string         `db:"password"`     // 密码（加密存储）
	NamespaceID int            `db:"namespace_id"` // 命名空间ID（用于筛选flows）
	CreatedBy   sql.NullString `db:"created_by"`   // 创建者用户ID
	UpdatedBy   sql.NullString `db:"updated_by"`   // 最后修改者用户ID
	CreatedAt   time.Time      `db:"created_at"`   // 创建时间
	UpdatedAt   time.Time      `db:"updated_at"`   // 更新时间
}

// ToDomain 将数据库模型转换为领域模型
func (m *PlatformConfigModel) ToDomain() *core.PlatformConfig {
	return &core.PlatformConfig{
		ID:          m.ID,
		TenantID:    m.TenantID,
		Host:        m.Host,
		Port:        m.Port,
		Database:    m.Database,
		Username:    m.Username,
		Password:    m.Password,
		NamespaceID: m.NamespaceID,
		CreatedBy:   convertNullString(m.CreatedBy),
		UpdatedBy:   convertNullString(m.UpdatedBy),
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromDomain 将领域模型转换为数据库模型
func FromDomain(config *core.PlatformConfig) *PlatformConfigModel {
	return &PlatformConfigModel{
		ID:          config.ID,
		TenantID:    config.TenantID,
		Host:        config.Host,
		Port:        config.Port,
		Database:    config.Database,
		Username:    config.Username,
		Password:    config.Password,
		NamespaceID: config.NamespaceID,
		CreatedBy:   convertToNullString(config.CreatedBy),
		UpdatedBy:   convertToNullString(config.UpdatedBy),
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}
}

// BuildDSN 构建数据库连接字符串
func (m *PlatformConfigModel) BuildDSN() string {
	user := url.QueryEscape(m.Username)
	pass := url.QueryEscape(m.Password)
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, pass, m.Host, m.Port, m.Database)
}
