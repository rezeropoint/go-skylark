package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/zeromicro/go-zero/core/logx"
)

// 以下是为了避免在未使用时产生编译错误的占位符
var _ = uuid.New

// queryLocalUserIDsWithCache 查询本地用户ID（带缓存优化，支持单个和批量）
//
// 职责：
//  1. 优先从反向缓存获取（skylark:user_mapping_rev:{tenantID}:{remoteUserID}）
//  2. 缓存未命中时批量查询数据库
//  3. 异步回写反向缓存（TTL 30天）
//
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - remoteUserIDs: 远程用户ID列表
//
// 返回：
//   - map[int]string: 远程ID → 本地ID 映射
//   - error: 查询失败时返回错误
//
// 说明：
//   - 该函数不检查是否所有ID都有映射（由调用方负责）
//   - 缓存回写采用异步方式（不阻塞主流程）
func (m *userManager) queryLocalUserIDsWithCache(ctx context.Context, tenantID string, remoteUserIDs []int) (map[int]string, error) {
	if len(remoteUserIDs) == 0 {
		return map[int]string{}, nil
	}

	// 临时映射：remote_user_id -> local_user_id
	tempMap := make(map[int]string)
	var missedRemoteUserIDs []int

	// 1. 遍历查询反向缓存
	for _, remoteUserID := range remoteUserIDs {
		localUserID, err := m.cache.GetUserIDMappingReverse(ctx, tenantID, remoteUserID)
		if err == nil {
			// 缓存命中
			tempMap[remoteUserID] = localUserID
		} else {
			// 缓存未命中，记录
			missedRemoteUserIDs = append(missedRemoteUserIDs, remoteUserID)
		}
	}

	// 2. 批量查询数据库（缓存未命中的）
	if len(missedRemoteUserIDs) > 0 {
		dbMappings, err := m.batchGetLocalUserIDsFromDB(ctx, tenantID, missedRemoteUserIDs)
		if err != nil {
			return nil, err
		}

		// 3. 填充结果并异步回写缓存
		for remoteUserID, localUserID := range dbMappings {
			tempMap[remoteUserID] = localUserID

			// 异步回写反向缓存（TTL 30天）
			go func(rid int, lid string) {
				if err := m.cache.SetUserIDMappingReverse(context.Background(), tenantID, rid, lid, 30*24*3600); err != nil {
					logx.Error("回写反向缓存失败（非致命错误）:", err)
				}
			}(remoteUserID, localUserID)
		}
	}

	return tempMap, nil
}

// batchGetLocalUserIDsFromDB 批量查询数据库获取本地用户ID（未命中缓存的）
func (m *userManager) batchGetLocalUserIDsFromDB(ctx context.Context, tenantID string, remoteUserIDs []int) (map[int]string, error) {
	if len(remoteUserIDs) == 0 {
		return map[int]string{}, nil
	}

	// 使用 pq.Array 构建批量查询（使用 ANY 运算符）
	query := `
		SELECT remote_user_id, local_user_id
		FROM skylark_user_mappings
		WHERE tenant_id = $1 AND remote_user_id = ANY($2)
	`

	var mappings []struct {
		RemoteUserID int    `db:"remote_user_id"`
		LocalUserID  string `db:"local_user_id"`
	}

	err := m.localDB.QueryRowsCtx(ctx, &mappings, query, tenantID, remoteUserIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询用户ID映射失败: %w", err)
	}

	// 转换为映射字典
	mapping := make(map[int]string, len(mappings))
	for _, m := range mappings {
		mapping[m.RemoteUserID] = m.LocalUserID
	}

	return mapping, nil
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
