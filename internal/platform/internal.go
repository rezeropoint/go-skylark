package platform

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/rezeropoint/go-skylark/v2/core"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// validateConnection 验证远程 Skylark 数据库连接
// 测试方式：连接数据库并查询 flows 表验证 namespace_id
func (m *platformManager) validateConnection(ctx context.Context, cfg *core.PlatformConfig) error {
	// 1. 构建 DSN
	dsn := cfg.BuildDSN()

	// 2. 连接数据库
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrDatabaseConnection, err)
	}
	defer db.Close()

	// 3. 设置连接超时
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Second)

	// 4. Ping 测试连接
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("%w: ping 失败: %v", core.ErrDatabaseConnection, err)
	}

	// 5. 查询 flows 表验证 namespace_id
	query := "SELECT COUNT(*) FROM flows WHERE namespace_id = $1"
	var count int
	err = db.QueryRowContext(ctx, query, cfg.NamespaceID).Scan(&count)
	if err != nil {
		return fmt.Errorf("%w: 查询 flows 表失败: %v", core.ErrInvalidPlatformConfig, err)
	}

	return nil
}

// checkPlatformInUse 检查平台配置是否被事件配置使用
// 返回：使用该平台的事件配置列表，如果没有被使用则返回空列表
func (m *platformManager) checkPlatformInUse(ctx context.Context, tenantID string) ([]string, error) {
	// 查询是否有事件配置关联该租户的平台
	// 注意：由于 skylark_platform_configs 是一租户一平台，所以只需要检查 tenant_id
	query := `
		SELECT name
		FROM event_configs
		WHERE tenant_id = $1 AND deleted_at IS NULL
		LIMIT 5
	`

	var results []struct {
		Name string `db:"name"`
	}

	err := m.dbConn.QueryRowsCtx(ctx, &results, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询使用平台配置的事件失败: %w", err)
	}

	// 构建事件名称列表
	usedByEvents := make([]string, 0, len(results))
	for _, r := range results {
		usedByEvents = append(usedByEvents, r.Name)
	}

	return usedByEvents, nil
}

// getOrCreateRemoteConn 获取或创建远程数据库连接（使用 sqlx）
func (m *platformManager) getOrCreateRemoteConn(tenantID string, cfg *core.PlatformConfig) (sqlx.SqlConn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查连接池中是否已存在
	if conn, exists := m.connPool[tenantID]; exists {
		return conn, nil
	}

	// 创建新连接（使用 go-zero sqlx）
	dsn := cfg.BuildDSN()
	conn := sqlx.NewSqlConn("postgres", dsn)

	// 测试连接（sqlx.SqlConn 没有 Ping 方法，执行简单查询测试）
	var testResult int
	err := conn.QueryRowCtx(context.Background(), &testResult, "SELECT 1")
	if err != nil {
		return nil, fmt.Errorf("%w: 连接测试失败: %v", core.ErrDatabaseConnection, err)
	}

	// 加入连接池
	m.connPool[tenantID] = conn

	return conn, nil
}

// closeAllRemoteConns 关闭所有远程连接
// 注意：sqlx.SqlConn 没有 Close 方法，只需清空连接池即可
func (m *platformManager) closeAllRemoteConns() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空连接池（sqlx 底层连接池会自动管理）
	m.connPool = make(map[string]sqlx.SqlConn)

	return nil
}

// initConnPool 初始化连接池
func (m *platformManager) initConnPool() {
	m.connPool = make(map[string]sqlx.SqlConn)
	m.mu = sync.RWMutex{}
}
