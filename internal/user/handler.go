package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// userManager 用户管理器实现
type userManager struct {
	localDB           sqlx.SqlConn               // 本地数据库连接
	cache             core.CacheInterface        // 缓存接口
	getPlatformConfig core.GetPlatformConfigFunc // 获取平台配置函数
	httpClient        *skylarkHTTPClient         // HTTP 客户端
}

// newUserManager 创建用户管理器
func newUserManager(config Config, db sqlx.SqlConn, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc) (*userManager, error) {
	// 验证参数
	if db == nil {
		return nil, fmt.Errorf("db 不能为空")
	}
	if cache == nil {
		return nil, fmt.Errorf("cache 不能为空")
	}
	if getPlatformConfig == nil {
		return nil, fmt.Errorf("getPlatformConfig 函数不能为空")
	}

	manager := &userManager{
		localDB:           db,
		cache:             cache,
		getPlatformConfig: getPlatformConfig,
		httpClient:        newSkylarkHTTPClient(),
	}

	// 初始化数据库表
	ctx := context.Background()
	if err := manager.initTable(ctx); err != nil {
		return nil, fmt.Errorf("初始化用户映射表失败: %w", err)
	}

	return manager, nil
}

// CreateUser 创建Skylark用户
func (m *userManager) CreateUser(ctx context.Context, tenantID, localUserID, name string, identifier, phone, openid *string) (*core.User, error) {
	// 1. 获取平台配置
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取平台配置失败: %w", err)
	}

	// 2. 调用 Skylark API 创建用户
	req := &CreateUserRequest{
		Name:       name,
		Identifier: identifier,
		Phone:      phone,
		Openid:     openid,
	}

	userResp, err := m.httpClient.createUser(ctx, platformConfig.App, platformConfig.Token, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrUserCreateFailed, err)
	}

	// 3. 保存映射关系
	mapping := &core.UserIDMapping{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		LocalUserID:  localUserID,
		RemoteUserID: userResp.ID,
	}

	if err := m.saveMapping(ctx, mapping); err != nil {
		// TODO: 考虑是否需要回滚远程创建的用户（调用删除API，如果有的话）
		return nil, fmt.Errorf("保存映射失败: %w", err)
	}

	// 4. 更新缓存
	if err := m.cacheMapping(ctx, mapping); err != nil {
		logx.WithContext(ctx).Error("更新缓存失败（非致命错误）:", err)
	}

	// 5. 转换为领域模型
	user := &core.User{ID: userResp.ID}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "user_manager"),
		logx.Field("operation", "create_user"),
		logx.Field("tenant_id", tenantID),
		logx.Field("local_user_id", localUserID),
		logx.Field("remote_user_id", userResp.ID),
	).Info("创建用户成功")

	return user, nil
}

// GetUser 查询用户（通过本地用户ID）
func (m *userManager) GetUser(ctx context.Context, tenantID, localUserID string) (*core.User, error) {
	// 查询映射关系（缓存优先 → 数据库）
	remoteUserID, err := m.GetRemoteUserID(ctx, tenantID, localUserID)
	if err != nil {
		return nil, err
	}

	return &core.User{ID: remoteUserID}, nil
}

// GetRemoteUserID 查询远程用户ID（单个）
func (m *userManager) GetRemoteUserID(ctx context.Context, tenantID, localUserID string) (int, error) {
	// 1. 优先从缓存获取
	remoteUserID, err := m.cache.GetUserIDMapping(ctx, tenantID, localUserID)
	if err == nil {
		return remoteUserID, nil
	}

	// 2. 缓存未命中，查询数据库
	query := "SELECT remote_user_id FROM skylark_user_mappings WHERE tenant_id = $1 AND local_user_id = $2"
	err = m.localDB.QueryRowCtx(ctx, &remoteUserID, query, tenantID, localUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, core.ErrUserMappingNotFound
		}
		return 0, fmt.Errorf("查询映射失败: %w", err)
	}

	// 3. 回写缓存（TTL 30天）
	if err := m.cache.SetUserIDMapping(ctx, tenantID, localUserID, remoteUserID, 30*24*3600); err != nil {
		logx.WithContext(ctx).Error("回写缓存失败（非致命错误）:", err)
	}

	return remoteUserID, nil
}

// GetRemoteUserIDs 批量查询远程用户ID
func (m *userManager) GetRemoteUserIDs(ctx context.Context, tenantID string, localUserIDs []string) ([]int, error) {
	if len(localUserIDs) == 0 {
		return []int{}, nil
	}

	// 结果映射：local_user_id -> remote_user_id
	resultMap := make(map[string]int)
	var missedLocalUserIDs []string

	// 1. 遍历查询缓存
	for _, localUserID := range localUserIDs {
		remoteUserID, err := m.cache.GetUserIDMapping(ctx, tenantID, localUserID)
		if err == nil {
			// 缓存命中
			resultMap[localUserID] = remoteUserID
		} else {
			// 缓存未命中，记录
			missedLocalUserIDs = append(missedLocalUserIDs, localUserID)
		}
	}

	// 2. 批量查询数据库（缓存未命中的）
	if len(missedLocalUserIDs) > 0 {
		// 构建批量查询（使用 ANY 运算符）
		query := `
			SELECT local_user_id, remote_user_id
			FROM skylark_user_mappings
			WHERE tenant_id = $1 AND local_user_id = ANY($2)
		`

		var mappings []struct {
			LocalUserID  string `db:"local_user_id"`
			RemoteUserID int    `db:"remote_user_id"`
		}

		err := m.localDB.QueryRowsCtx(ctx, &mappings, query, tenantID, missedLocalUserIDs)
		if err != nil {
			return nil, fmt.Errorf("批量查询映射失败: %w", err)
		}

		// 3. 填充结果并回写缓存
		for _, mapping := range mappings {
			resultMap[mapping.LocalUserID] = mapping.RemoteUserID

			// 回写缓存（TTL 30天）
			if err := m.cache.SetUserIDMapping(ctx, tenantID, mapping.LocalUserID, mapping.RemoteUserID, 30*24*3600); err != nil {
				logx.WithContext(ctx).Error("回写缓存失败（非致命错误）:", err)
			}
		}
	}

	// 4. 按输入顺序返回结果
	result := make([]int, 0, len(localUserIDs))
	for _, localUserID := range localUserIDs {
		if remoteUserID, ok := resultMap[localUserID]; ok {
			result = append(result, remoteUserID)
		} else {
			// 映射不存在，返回错误
			return nil, fmt.Errorf("%w: local_user_id=%s", core.ErrUserMappingNotFound, localUserID)
		}
	}

	return result, nil
}

// GetLocalUserID 反向查询本地用户ID
func (m *userManager) GetLocalUserID(ctx context.Context, tenantID string, remoteUserID int) (string, error) {
	// 1. 优先从反向缓存获取
	localUserID, err := m.cache.GetUserIDMappingReverse(ctx, tenantID, remoteUserID)
	if err == nil {
		return localUserID, nil
	}

	// 2. 缓存未命中，查询数据库
	query := "SELECT local_user_id FROM skylark_user_mappings WHERE tenant_id = $1 AND remote_user_id = $2"
	err = m.localDB.QueryRowCtx(ctx, &localUserID, query, tenantID, remoteUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", core.ErrUserMappingNotFound
		}
		return "", fmt.Errorf("查询映射失败: %w", err)
	}

	// 3. 回写反向缓存（TTL 30天）
	if err := m.cache.SetUserIDMappingReverse(ctx, tenantID, remoteUserID, localUserID, 30*24*3600); err != nil {
		logx.WithContext(ctx).Error("回写反向缓存失败（非致命错误）:", err)
	}

	return localUserID, nil
}

// saveMapping 保存映射关系到数据库
func (m *userManager) saveMapping(ctx context.Context, mapping *core.UserIDMapping) error {
	query := `
		INSERT INTO skylark_user_mappings (id, tenant_id, local_user_id, remote_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`

	_, err := m.localDB.ExecCtx(ctx, query,
		mapping.ID,
		mapping.TenantID,
		mapping.LocalUserID,
		mapping.RemoteUserID,
	)

	if err != nil {
		return fmt.Errorf("插入映射失败: %w", err)
	}

	return nil
}

// cacheMapping 更新缓存（正向 + 反向）
func (m *userManager) cacheMapping(ctx context.Context, mapping *core.UserIDMapping) error {
	// 1. 正向缓存（local_user_id -> remote_user_id）
	if err := m.cache.SetUserIDMapping(ctx, mapping.TenantID, mapping.LocalUserID, mapping.RemoteUserID, 30*24*3600); err != nil {
		return fmt.Errorf("缓存正向映射失败: %w", err)
	}

	// 2. 反向缓存（remote_user_id -> local_user_id）
	if err := m.cache.SetUserIDMappingReverse(ctx, mapping.TenantID, mapping.RemoteUserID, mapping.LocalUserID, 30*24*3600); err != nil {
		return fmt.Errorf("缓存反向映射失败: %w", err)
	}

	return nil
}

// initTable 初始化数据库表（如果不存在则创建）
func (m *userManager) initTable(ctx context.Context) error {
	// 检查表是否存在
	checkQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'skylark_user_mappings'
		)
	`

	var exists bool
	err := m.localDB.QueryRowCtx(ctx, &exists, checkQuery)
	if err != nil {
		return fmt.Errorf("检查表存在性失败: %w", err)
	}

	// 如果表已存在，直接返回
	if exists {
		return nil
	}

	// 创建表（使用 sql.go 中的常量）
	_, err = m.localDB.ExecCtx(ctx, CreateTableSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	logx.WithContext(ctx).Info("skylark_user_mappings 表创建成功")
	return nil
}
