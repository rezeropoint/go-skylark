package platform

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/rezeropoint/go-skylark/v2/core"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// platformManager 平台配置管理器实现
type platformManager struct {
	config   Config
	dbConn   sqlx.SqlConn            // 本地数据库连接
	cache    core.CacheInterface     // 缓存接口（可选）
	connPool map[string]sqlx.SqlConn // 远程数据库连接池（key: tenantID）
	mu       sync.RWMutex            // 连接池并发保护
}

// newPlatformManager 创建平台配置管理器
func newPlatformManager(config Config, db sqlx.SqlConn, cache core.CacheInterface) (*platformManager, error) {
	manager := &platformManager{
		config: config,
		dbConn: db,
		cache:  cache,
	}

	// 初始化连接池
	manager.initConnPool()

	// 初始化数据库表（在包初始化时执行）
	ctx := context.Background()
	if err := manager.initTable(ctx); err != nil {
		return nil, fmt.Errorf("初始化平台配置表失败: %w", err)
	}

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
	if cfg.APIBaseURL == "" {
		return "", fmt.Errorf("%w: APIBaseURL 不能为空", core.ErrInvalidPlatformConfig)
	}
	if cfg.APIToken == "" {
		return "", fmt.Errorf("%w: APIToken 不能为空", core.ErrInvalidPlatformConfig)
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
	// 转换为数据模型
	model := FromDomain(cfg)

	insertQuery := `
		INSERT INTO skylark_platform_configs
		(id, tenant_id, host, port, database, username, password, namespace_id, api_base_url, api_token, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = m.dbConn.ExecCtx(ctx, insertQuery,
		model.ID, model.TenantID, model.Host, model.Port, model.Database,
		model.Username, model.Password, model.NamespaceID, model.APIBaseURL, model.APIToken, model.CreatedBy,
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
	// 1. 尝试从缓存获取
	if m.cache != nil {
		cached, err := m.cache.GetPlatformConfig(ctx, tenantID)
		if err == nil {
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "platform_manager"),
				logx.Field("operation", "get"),
				logx.Field("tenant_id", tenantID),
				logx.Field("cache_hit", true),
			).Debug("平台配置缓存命中")
			return cached, nil
		}
	}

	// 2. 缓存未命中，查询数据库
	query := `
		SELECT id, tenant_id, host, port, database, username, password, namespace_id,
		       api_base_url, api_token, created_by, updated_by, created_at, updated_at
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

	config := model.ToDomain()

	// 3. 写入缓存
	if m.cache != nil {
		ttl := int(m.config.PlatformConfigCacheTTL.Seconds())
		_ = m.cache.SetPlatformConfig(ctx, config, ttl)
	}

	return config, nil
}

// GetAPIConfig 获取Skylark API调用配置
// 实现了 core.GetPlatformConfigFunc 函数签名，用于依赖注入
func (m *platformManager) GetAPIConfig(ctx context.Context, tenantID string) (*core.SkylarkAPIConfig, error) {
	// 1. 查询平台配置
	cfg, err := m.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. 验证 APIBaseURL 和 APIToken 是否配置
	if cfg.APIBaseURL == "" {
		return nil, fmt.Errorf("%w: 租户 %s 的 APIBaseURL 未配置", core.ErrInvalidPlatformConfig, tenantID)
	}
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("%w: 租户 %s 的 APIToken 未配置", core.ErrInvalidPlatformConfig, tenantID)
	}

	// 3. 返回轻量级的 API 配置（不包含敏感数据库信息）
	return &core.SkylarkAPIConfig{
		App:   cfg.APIBaseURL,
		Token: cfg.APIToken,
	}, nil
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
	if cfg.APIBaseURL == "" {
		return fmt.Errorf("%w: APIBaseURL 不能为空", core.ErrInvalidPlatformConfig)
	}
	if cfg.APIToken == "" {
		return fmt.Errorf("%w: APIToken 不能为空", core.ErrInvalidPlatformConfig)
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
	// 转换为数据模型
	model := FromDomain(cfg)

	updateQuery := `
		UPDATE skylark_platform_configs
		SET host = $1, port = $2, database = $3, username = $4, password = $5,
		    namespace_id = $6, api_base_url = $7, api_token = $8, updated_by = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $10 AND tenant_id = $11
	`
	result, err := m.dbConn.ExecCtx(ctx, updateQuery,
		model.Host, model.Port, model.Database, model.Username, model.Password,
		model.NamespaceID, model.APIBaseURL, model.APIToken, model.UpdatedBy, model.ID, model.TenantID,
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

	// 8. 清除平台配置缓存
	if m.cache != nil {
		_ = m.cache.DeletePlatformConfig(ctx, cfg.TenantID)
	}

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

	// 6. 清除平台配置缓存
	if m.cache != nil {
		_ = m.cache.DeletePlatformConfig(ctx, tenantID)
	}

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

// initTable 初始化数据库表（私有方法，在包初始化时调用）
func (m *platformManager) initTable(ctx context.Context) error {
	// 先检查表是否已存在
	var count int
	err := m.dbConn.QueryRowCtx(ctx, &count, CheckTableExistsSQL)
	if err != nil {
		return fmt.Errorf("检查平台配置表存在性失败: %w", err)
	}

	// 如果表已存在，直接返回
	if count > 0 {
		return nil
	}

	// 表不存在，执行创建
	_, err = m.dbConn.ExecCtx(ctx, CreateTableSQL)
	if err != nil {
		return fmt.Errorf("创建平台配置表失败: %w", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "platform_manager"),
		logx.Field("operation", "init_table"),
		logx.Field("table", TableName),
	).Info("平台配置表初始化成功")

	return nil
}
