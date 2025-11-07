package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// getCachedFlowList 从缓存获取 flows 列表
func (m *queryManager) getCachedFlowList(ctx context.Context, tenantID string, namespaceID int) ([]*core.FlowInfo, error) {

	key := fmt.Sprintf("%s%s:%d", core.CacheFlowListKeyPrefix, tenantID, namespaceID)
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err // 缓存未命中或错误
	}

	var flows []*core.FlowInfo
	if err := json.Unmarshal([]byte(val), &flows); err != nil {
		return nil, fmt.Errorf("反序列化 flows 缓存失败: %w", err)
	}

	return flows, nil
}

// setCachedFlowList 缓存 flows 列表
func (m *queryManager) setCachedFlowList(ctx context.Context, tenantID string, namespaceID int, flows []*core.FlowInfo) error {

	key := fmt.Sprintf("%s%s:%d", core.CacheFlowListKeyPrefix, tenantID, namespaceID)
	data, err := json.Marshal(flows)
	if err != nil {
		return fmt.Errorf("序列化 flows 列表失败: %w", err)
	}

	return m.rdb.SetexCtx(ctx, key, string(data), int(m.config.FlowListCacheTTL.Seconds()))
}

// getCachedFlowFields 从缓存获取 flow 字段列表
func (m *queryManager) getCachedFlowFields(ctx context.Context, tenantID string, flowID int) ([]*core.FieldMetadata, error) {

	key := fmt.Sprintf("%s%s:%d", core.CacheFlowFieldsKeyPrefix, tenantID, flowID)
	val, err := m.rdb.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}

	var fields []*core.FieldMetadata
	if err := json.Unmarshal([]byte(val), &fields); err != nil {
		return nil, fmt.Errorf("反序列化 flow fields 缓存失败: %w", err)
	}

	return fields, nil
}

// setCachedFlowFields 缓存 flow 字段列表
func (m *queryManager) setCachedFlowFields(ctx context.Context, tenantID string, flowID int, fields []*core.FieldMetadata) error {

	key := fmt.Sprintf("%s%s:%d", core.CacheFlowFieldsKeyPrefix, tenantID, flowID)
	data, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("序列化 flow fields 列表失败: %w", err)
	}

	return m.rdb.SetexCtx(ctx, key, string(data), int(m.config.FlowFieldsCacheTTL.Seconds()))
}

// batchGetUserNames 批量查询用户名（优先从 Redis 缓存获取）
func (m *queryManager) batchGetUserNames(ctx context.Context, remoteDB sqlx.SqlConn, tenantID string, userIDs []string) (map[string]string, error) {
	if len(userIDs) == 0 {
		return make(map[string]string), nil
	}

	userNames := make(map[string]string, len(userIDs))
	uncachedIDs := []string{}

	// 1. 尝试从 Redis 获取缓存（go-zero Redis 逐个获取）
	if m.rdb != nil {
		for _, userID := range userIDs {
			key := fmt.Sprintf("%s%s:%s", core.CacheUserNameKeyPrefix, tenantID, userID)
			val, err := m.rdb.Get(key)
			if err == nil && val != "" {
				userNames[userID] = val
			} else {
				uncachedIDs = append(uncachedIDs, userID)
			}
		}
	} else {
		uncachedIDs = userIDs
	}

	// 2. 查询未命中的用户名（从远程 users 表）
	if len(uncachedIDs) > 0 {
		type userRow struct {
			ID   string `db:"id"`
			Name string `db:"name"`
		}

		query := "SELECT id, name FROM users WHERE id = ANY($1)"
		var users []*userRow
		err := remoteDB.QueryRowsCtx(ctx, &users, query, pq.Array(uncachedIDs))
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("批量查询用户名失败: %w", err)
		}

		for _, user := range users {
			userNames[user.ID] = user.Name

			// 3. 写入 Redis 缓存
			if m.rdb != nil {
				cacheKey := fmt.Sprintf("%s%s:%s", core.CacheUserNameKeyPrefix, tenantID, user.ID)
				_ = m.rdb.Setex(cacheKey, user.Name, int(m.config.UserNameCacheTTL.Seconds()))
			}
		}
	}

	return userNames, nil
}
