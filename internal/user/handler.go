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
func (m *userManager) CreateUser(ctx context.Context, tenantID, localUserID, name string, identifier, phone, openid string) error {
	// 1. 获取平台配置
	platformConfig, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("获取平台配置失败: %w", err)
	}

	// 2. 转换可选参数：空字符串 → nil 指针
	var identifierPtr, phonePtr, openidPtr *string
	if identifier != "" {
		identifierPtr = &identifier
	}
	if phone != "" {
		phonePtr = &phone
	}
	if openid != "" {
		openidPtr = &openid
	}

	// 3. 调用 Skylark API 创建用户
	req := &CreateUserRequest{
		Name:       name,
		Identifier: identifierPtr,
		Phone:      phonePtr,
		Openid:     openidPtr,
	}

	userResp, err := m.httpClient.createUser(ctx, platformConfig.App, platformConfig.Token, req)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrUserCreateFailed, err)
	}

	// 4. 保存映射关系
	mapping := &core.UserIDMapping{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		LocalUserID:  localUserID,
		RemoteUserID: userResp.ID,
	}

	if err := m.saveMapping(ctx, mapping); err != nil {
		// TODO: 考虑是否需要回滚远程创建的用户（调用删除API，如果有的话）
		return fmt.Errorf("保存映射失败: %w", err)
	}

	// 5. 更新缓存
	if err := m.cacheMapping(ctx, mapping); err != nil {
		logx.WithContext(ctx).Error("更新缓存失败（非致命错误）:", err)
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "user_manager"),
		logx.Field("operation", "create_user"),
		logx.Field("tenant_id", tenantID),
		logx.Field("local_user_id", localUserID),
		logx.Field("remote_user_id", userResp.ID),
	).Info("创建用户成功")

	return nil
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

// GetLocalUserID 反向查询本地用户ID（单个）
func (m *userManager) GetLocalUserID(ctx context.Context, tenantID string, remoteUserID int) (string, error) {
	// 使用统一的缓存查询函数（单个ID）
	mapping, err := m.queryLocalUserIDsWithCache(ctx, tenantID, []int{remoteUserID})
	if err != nil {
		return "", err
	}

	// 检查是否有映射
	localUserID, ok := mapping[remoteUserID]
	if !ok {
		return "", core.ErrUserMappingNotFound
	}

	return localUserID, nil
}

// FillLocalUserIDMap 批量反向转换远程用户ID为本地用户ID（填充映射）
func (m *userManager) FillLocalUserIDMap(ctx context.Context, tenantID string, userIDMapping *map[int]string) error {
	if userIDMapping == nil {
		return fmt.Errorf("userIDMapping 不能为 nil")
	}

	if len(*userIDMapping) == 0 {
		return nil
	}

	// 从 map keys 提取需要查询的远程用户ID列表
	remoteUserIDs := make([]int, 0, len(*userIDMapping))
	for remoteUserID := range *userIDMapping {
		remoteUserIDs = append(remoteUserIDs, remoteUserID)
	}

	// 使用统一的缓存查询函数
	tempMap, err := m.queryLocalUserIDsWithCache(ctx, tenantID, remoteUserIDs)
	if err != nil {
		return err
	}

	// 检查是否所有ID都有映射
	for _, remoteUserID := range remoteUserIDs {
		if _, ok := tempMap[remoteUserID]; !ok {
			return fmt.Errorf("%w: remote_user_id=%d", core.ErrUserMappingNotFound, remoteUserID)
		}
	}

	// 填充传入的映射
	for remoteUserID, localUserID := range tempMap {
		(*userIDMapping)[remoteUserID] = localUserID
	}

	return nil
}

// GetUserSyncStatus 获取用户同步状态
func (m *userManager) GetUserSyncStatus(ctx context.Context, tenantID, localUserID string) (bool, error) {
	// 1. 优先从缓存检查
	_, err := m.cache.GetUserIDMapping(ctx, tenantID, localUserID)
	if err == nil {
		// 缓存命中，说明映射存在
		return true, nil
	}

	// 2. 缓存未命中，查询数据库
	var remoteUserID int
	query := "SELECT remote_user_id FROM skylark_user_mappings WHERE tenant_id = $1 AND local_user_id = $2"
	err = m.localDB.QueryRowCtx(ctx, &remoteUserID, query, tenantID, localUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			// 映射不存在，返回 false（不返回错误）
			return false, nil
		}
		// 数据库查询错误
		return false, fmt.Errorf("查询用户映射失败: %w", err)
	}

	// 3. 映射存在，回写缓存（TTL 30天）
	if err := m.cache.SetUserIDMapping(ctx, tenantID, localUserID, remoteUserID, 30*24*3600); err != nil {
		logx.WithContext(ctx).Error("回写缓存失败（非致命错误）:", err)
	}

	return true, nil
}
