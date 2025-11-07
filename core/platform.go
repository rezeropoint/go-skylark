// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 数据库查询
// 说明：本文件定义的类型用于查询 Skylark PostgreSQL 数据库（读操作：平台配置管理、远程数据库连接）
package core

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// GetRemoteDBFunc 获取远程 Skylark 数据库连接的函数类型
// 用途：供 query、event 等 Manager 获取远程数据库连接，实现 Manager 之间解耦
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID（用于查询平台配置）
//
// 返回：
//   - sqlx.SqlConn: 远程数据库连接（go-zero sqlx 连接）
//   - error: 错误信息（如平台配置不存在、连接失败等）
//
// 说明：
//   - 该函数会自动从 skylark_platform_configs 表读取配置
//   - 自动管理连接池（复用已建立的连接）
//   - 连接失败时返回 ErrDatabaseConnection 或 ErrPlatformConfigNotFound
type GetRemoteDBFunc func(ctx context.Context, tenantID string) (sqlx.SqlConn, error)

// PlatformConfig Skylark平台对接配置（纯领域模型）
type PlatformConfig struct {
	ID          string     // 配置UUID
	TenantID    string     // 租户ID（一租户一平台）
	Host        string     // 数据库地址（如：110.41.35.134）
	Port        int        // 端口（如：5432）
	Database    string     // 数据库名（如：sync）
	Username    string     // 用户名
	Password    string     // 密码（加密存储）
	NamespaceID int        // 命名空间ID（用于筛选flows）
	CreatedBy   *string    // 创建者用户ID
	UpdatedBy   *string    // 最后修改者用户ID
	CreatedAt   time.Time  // 创建时间
	UpdatedAt   time.Time  // 更新时间
}

// BuildDSN 构建数据库连接字符串
// 注意：密码中的特殊字符会自动进行 URL 编码
func (c *PlatformConfig) BuildDSN() string {
	// 对用户名和密码进行 URL 编码，防止特殊字符导致解析失败
	// 例如：密码 "A9&fG2!sH7#kL1@tM3^pN5*dQ8?zR0uV*" 会被正确编码
	user := url.QueryEscape(c.Username)
	pass := url.QueryEscape(c.Password)

	// postgres://user:password@host:port/database?sslmode=disable
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, pass, c.Host, c.Port, c.Database)
}
