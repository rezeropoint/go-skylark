package platform

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// platformManager 平台配置管理器实现
type platformManager struct {
	dbConn   sqlx.SqlConn            // 本地数据库连接
	connPool map[string]sqlx.SqlConn // 远程数据库连接池（key: tenantID）
	mu       sync.RWMutex            // 连接池并发保护
}

// newPlatformManager 创建平台配置管理器
func newPlatformManager(db sqlx.SqlConn) (*platformManager, error) {
	manager := &platformManager{
		dbConn: db,
	}

	// 初始化连接池
	manager.initConnPool()

	return manager, nil
}

// Create 创建平台配置
func (m *platformManager) Create(ctx context.Context, cfg *core.PlatformConfig) (string, error) {
	// 1. 生成 UUID（禁止调用方指定）
	if cfg.ID != "" {
		return "", fmt.Errorf("%w: 创建时不允许指定ID，系统会自动生成", core.ErrInvalidPlatformConfig)
	}
	cfg.ID = uuid.New().String()

	// 2. 验证配置（基础字段验证）
	if cfg.TenantID == "" {
		return "", fmt.Errorf("%w: 租户ID不能为空", core.ErrInvalidPlatformConfig)
	}
	if cfg.Host == "" || cfg.Database == "" || cfg.Username == "" || cfg.Password == "" {
		return "", fmt.Errorf("%w: 连接信息不完整", core.ErrInvalidPlatformConfig)
	}
	if cfg.NamespaceID <= 0 {
		return "", fmt.Errorf("%w: NamespaceID 必须大于0", core.ErrInvalidPlatformConfig)
	}

	// 3. 验证连接是否可用（先测试连接再入库）
	if err := m.validateConnection(ctx, cfg); err != nil {
		return "", err
	}

	// 4. 检查租户是否已存在平台配置（一租户一平台）
	var existingID string
	checkQuery := "SELECT id FROM skylark_platform_configs WHERE tenant_id = $1"
	err := m.dbConn.QueryRowCtx(ctx, &existingID, checkQuery, cfg.TenantID)
	if err == nil {
		// 已存在配置
		return "", fmt.Errorf("%w: 该租户已存在平台配置 (ID: %s)", core.ErrInvalidPlatformConfig, existingID)
	} else if err != sql.ErrNoRows {
		// 查询错误
		return "", fmt.Errorf("检查租户平台配置失败: %w", err)
	}

	// 5. 插入数据库
	insertQuery := `
		INSERT INTO skylark_platform_configs
		(id, tenant_id, host, port, database, username, password, namespace_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = m.dbConn.ExecCtx(ctx, insertQuery,
		cfg.ID, cfg.TenantID, cfg.Host, cfg.Port, cfg.Database,
		cfg.Username, cfg.Password, cfg.NamespaceID, cfg.CreatedBy,
	)
	if err != nil {
		return "", fmt.Errorf("插入平台配置失败: %w", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "platform_manager"),
		logx.Field("operation", "create"),
		logx.Field("platform_id", cfg.ID),
		logx.Field("tenant_id", cfg.TenantID),
		logx.Field("namespace_id", cfg.NamespaceID),
	).Info("创建平台配置成功")

	return cfg.ID, nil
}

// Get 获取平台配置（根据租户ID查询）
func (m *platformManager) Get(ctx context.Context, tenantID string) (*core.PlatformConfig, error) {
	query := `
		SELECT id, tenant_id, host, port, database, username, password, namespace_id,
		       created_by, updated_by, created_at, updated_at
		FROM skylark_platform_configs
		WHERE tenant_id = $1
	`

	var model PlatformConfigModel
	err := m.dbConn.QueryRowCtx(ctx, &model, query, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrPlatformConfigNotFound
		}
		return nil, fmt.Errorf("查询平台配置失败: %w", err)
	}

	return model.ToDomain(), nil
}

// Update 更新平台配置
func (m *platformManager) Update(ctx context.Context, cfg *core.PlatformConfig) error {
	// 1. 验证配置
	if cfg.ID == "" || cfg.TenantID == "" {
		return fmt.Errorf("%w: ID和租户ID不能为空", core.ErrInvalidPlatformConfig)
	}
	if cfg.Host == "" || cfg.Database == "" || cfg.Username == "" || cfg.Password == "" {
		return fmt.Errorf("%w: 连接信息不完整", core.ErrInvalidPlatformConfig)
	}
	if cfg.NamespaceID <= 0 {
		return fmt.Errorf("%w: NamespaceID 必须大于0", core.ErrInvalidPlatformConfig)
	}

	// 2. 验证新连接是否可用
	if err := m.validateConnection(ctx, cfg); err != nil {
		return err
	}

	// 3. 查询旧配置验证权限
	var oldTenantID string
	checkQuery := "SELECT tenant_id FROM skylark_platform_configs WHERE id = $1"
	err := m.dbConn.QueryRowCtx(ctx, &oldTenantID, checkQuery, cfg.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return core.ErrPlatformConfigNotFound
		}
		return fmt.Errorf("查询平台配置失败: %w", err)
	}

	// 4. 验证租户ID一致性
	if oldTenantID != cfg.TenantID {
		return fmt.Errorf("%w: 不允许修改租户ID", core.ErrInvalidPlatformConfig)
	}

	// 5. 更新数据库
	updateQuery := `
		UPDATE skylark_platform_configs
		SET host = $1, port = $2, database = $3, username = $4, password = $5,
		    namespace_id = $6, updated_by = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND tenant_id = $9
	`
	result, err := m.dbConn.ExecCtx(ctx, updateQuery,
		cfg.Host, cfg.Port, cfg.Database, cfg.Username, cfg.Password,
		cfg.NamespaceID, cfg.UpdatedBy, cfg.ID, cfg.TenantID,
	)
	if err != nil {
		return fmt.Errorf("更新平台配置失败: %w", err)
	}

	// 6. 检查是否有记录被更新
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取更新结果失败: %w", err)
	}
	if affected == 0 {
		return core.ErrPlatformConfigNotFound
	}

	// 7. 清除该租户的远程连接缓存（连接信息已变更）
	// 注意：sqlx.SqlConn 没有 Close 方法，直接删除即可
	m.mu.Lock()
	delete(m.connPool, cfg.TenantID)
	m.mu.Unlock()

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "platform_manager"),
		logx.Field("operation", "update"),
		logx.Field("platform_id", cfg.ID),
		logx.Field("tenant_id", cfg.TenantID),
	).Info("更新平台配置成功")

	return nil
}

// Delete 删除平台配置
func (m *platformManager) Delete(ctx context.Context, tenantID string) error {
	// 1. 查询平台配置是否存在
	var platformID string
	checkQuery := "SELECT id FROM skylark_platform_configs WHERE tenant_id = $1"
	err := m.dbConn.QueryRowCtx(ctx, &platformID, checkQuery, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return core.ErrPlatformConfigNotFound
		}
		return fmt.Errorf("查询平台配置失败: %w", err)
	}

	// 2. 检查是否被事件配置使用
	usedByEvents, err := m.checkPlatformInUse(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("检查平台配置使用情况失败: %w", err)
	}
	if len(usedByEvents) > 0 {
		// 构建详细错误信息
		eventsInfo := ""
		for i, event := range usedByEvents {
			if i > 0 {
				eventsInfo += ", "
			}
			eventsInfo += event
		}
		return fmt.Errorf("无法删除平台配置: 以下事件正在使用该平台: %s", eventsInfo)
	}

	// 3. 硬删除
	deleteQuery := "DELETE FROM skylark_platform_configs WHERE tenant_id = $1"
	result, err := m.dbConn.ExecCtx(ctx, deleteQuery, tenantID)
	if err != nil {
		return fmt.Errorf("删除平台配置失败: %w", err)
	}

	// 4. 检查是否有记录被删除
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}
	if affected == 0 {
		return core.ErrPlatformConfigNotFound
	}

	// 5. 清除该租户的远程连接缓存
	// 注意：sqlx.SqlConn 没有 Close 方法，直接删除即可
	m.mu.Lock()
	delete(m.connPool, tenantID)
	m.mu.Unlock()

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "platform_manager"),
		logx.Field("operation", "delete"),
		logx.Field("platform_id", platformID),
		logx.Field("tenant_id", tenantID),
	).Info("删除平台配置成功")

	return nil
}

// Validate 验证平台连接
func (m *platformManager) Validate(ctx context.Context, cfg *core.PlatformConfig) error {
	return m.validateConnection(ctx, cfg)
}

// GetRemoteDB 获取远程 Skylark 数据库连接
func (m *platformManager) GetRemoteDB(ctx context.Context, tenantID string) (sqlx.SqlConn, error) {
	// 1. 从本地数据库读取平台配置
	cfg, err := m.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. 获取或创建远程连接（复用连接池）
	return m.getOrCreateRemoteConn(tenantID, cfg)
}

// Close 关闭管理器
func (m *platformManager) Close() error {
	return m.closeAllRemoteConns()
}
