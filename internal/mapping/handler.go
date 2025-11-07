package mapping

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// mappingManager 组织映射管理器实现
type mappingManager struct {
	dbConn sqlx.SqlConn // 本地数据库连接
	rdb    *redis.Redis // Redis客户端（go-zero版本，用于缓存）
}

// newMappingManager 创建组织映射管理器
func newMappingManager(db sqlx.SqlConn, rdb *redis.Redis) (*mappingManager, error) {
	return &mappingManager{
		dbConn: db,
		rdb:    rdb,
	}, nil
}

// CreateOrgMapping 创建组织映射
func (m *mappingManager) CreateOrgMapping(ctx context.Context, mapping *core.OrgMapping) (string, error) {
	// 1. 生成 UUID（禁止调用方指定）
	if mapping.ID != "" {
		return "", fmt.Errorf("创建时不允许指定ID，系统会自动生成")
	}
	mapping.ID = uuid.New().String()

	// 2. 验证必填字段
	if mapping.TenantID == "" {
		return "", fmt.Errorf("租户ID不能为空")
	}
	if mapping.RemoteOrgValue == "" {
		return "", fmt.Errorf("远程组织值不能为空")
	}
	if mapping.LocalOrgID == "" {
		return "", fmt.Errorf("%w: 本地组织ID不能为空", core.ErrInvalidLocalOrgID)
	}

	// 3. 验证 local_org_id 是否存在
	if err := m.validateOrgExists(ctx, mapping.LocalOrgID, mapping.TenantID); err != nil {
		return "", err
	}

	// 4. 验证唯一约束 (tenant_id + remote_org_value)
	if err := m.checkMappingExists(ctx, mapping.TenantID, mapping.RemoteOrgValue, ""); err != nil {
		return "", err
	}

	// 5. 验证唯一约束 (tenant_id + local_org_id)
	if err := m.checkLocalOrgIDExists(ctx, mapping.TenantID, mapping.LocalOrgID, ""); err != nil {
		return "", err
	}

	// 6. 使用事务插入数据库并更新缓存
	err := m.dbConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 插入数据库
		insertQuery := `
			INSERT INTO event_org_mappings (id, remote_org_value, local_org_id, tenant_id)
			VALUES ($1, $2, $3, $4)
		`
		_, err := session.ExecCtx(ctx, insertQuery,
			mapping.ID, mapping.RemoteOrgValue, mapping.LocalOrgID, mapping.TenantID,
		)
		if err != nil {
			return fmt.Errorf("插入组织映射失败: %w", err)
		}

		// 获取完整的映射信息（包含组织名称）
		queryMapping := `
			SELECT
				m.id,
				m.tenant_id,
				m.remote_org_value,
				m.local_org_id,
				COALESCE(o.name, '') AS local_org_name,
				m.created_at,
				m.updated_at
			FROM event_org_mappings m
			LEFT JOIN organizations o ON m.local_org_id = o.id
			WHERE m.id = $1
		`
		var createdMapping core.OrgMapping
		if err := session.QueryRowCtx(ctx, &createdMapping, queryMapping, mapping.ID); err != nil {
			return fmt.Errorf("查询新创建的映射失败: %w", err)
		}

		// 更新缓存（单个映射）
		if m.rdb != nil {
			if err := m.setCachedMapping(ctx, &createdMapping); err != nil {
				logx.WithContext(ctx).Error("缓存组织映射失败（非致命错误）:", err)
			}
			// 删除列表缓存，触发下次查询时重新加载
			if err := m.deleteCachedMappingList(ctx, mapping.TenantID); err != nil {
				logx.WithContext(ctx).Error("删除映射列表缓存失败（非致命错误）:", err)
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "mapping_manager"),
		logx.Field("operation", "create"),
		logx.Field("mapping_id", mapping.ID),
		logx.Field("tenant_id", mapping.TenantID),
		logx.Field("remote_org_value", mapping.RemoteOrgValue),
		logx.Field("local_org_id", mapping.LocalOrgID),
	).Info("创建组织映射成功")

	return mapping.ID, nil
}

// GetOrgMapping 获取组织映射（根据ID查询）
func (m *mappingManager) GetOrgMapping(ctx context.Context, id string) (*core.OrgMapping, error) {
	// 1. 尝试从缓存获取
	if m.rdb != nil {
		cachedMapping, err := m.getCachedMapping(ctx, id)
		if err == nil {
			return cachedMapping, nil
		}
		// 缓存未命中，继续查询数据库
	}

	// 2. 从数据库查询（LEFT JOIN organizations 表获取组织名称）
	query := `
		SELECT
			m.id,
			m.tenant_id,
			m.remote_org_value,
			m.local_org_id,
			COALESCE(o.name, '') AS local_org_name,
			m.created_at,
			m.updated_at
		FROM event_org_mappings m
		LEFT JOIN organizations o ON m.local_org_id = o.id
		WHERE m.id = $1
	`

	var mapping core.OrgMapping
	err := m.dbConn.QueryRowCtx(ctx, &mapping, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrOrgMappingNotFound
		}
		return nil, fmt.Errorf("查询组织映射失败: %w", err)
	}

	// 3. 更新缓存
	if m.rdb != nil {
		if err := m.setCachedMapping(ctx, &mapping); err != nil {
			logx.WithContext(ctx).Error("缓存组织映射失败（非致命错误）:", err)
		}
	}

	return &mapping, nil
}

// ListOrgMappings 查询组织映射列表（根据租户ID）
func (m *mappingManager) ListOrgMappings(ctx context.Context, tenantID string) ([]*core.OrgMapping, error) {
	// 1. 尝试从缓存获取
	if m.rdb != nil {
		cachedList, err := m.getCachedMappingList(ctx, tenantID)
		if err == nil {
			return cachedList, nil
		}
		// 缓存未命中，继续查询数据库
	}

	// 2. 从数据库查询（LEFT JOIN organizations 表获取组织名称）
	query := `
		SELECT
			m.id,
			m.tenant_id,
			m.remote_org_value,
			m.local_org_id,
			COALESCE(o.name, '') AS local_org_name,
			m.created_at,
			m.updated_at
		FROM event_org_mappings m
		LEFT JOIN organizations o ON m.local_org_id = o.id
		WHERE m.tenant_id = $1
		ORDER BY m.remote_org_value ASC
	`

	var mappings []*core.OrgMapping
	err := m.dbConn.QueryRowsCtx(ctx, &mappings, query, tenantID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询组织映射列表失败: %w", err)
	}

	// 如果没有记录，返回空数组而不是 nil
	if mappings == nil {
		mappings = []*core.OrgMapping{}
	}

	// 3. 更新缓存
	if m.rdb != nil {
		if err := m.setCachedMappingList(ctx, tenantID, mappings); err != nil {
			logx.WithContext(ctx).Error("缓存组织映射列表失败（非致命错误）:", err)
		}
	}

	return mappings, nil
}

// UpdateOrgMapping 更新组织映射
func (m *mappingManager) UpdateOrgMapping(ctx context.Context, mapping *core.OrgMapping) error {
	// 1. 验证必填字段
	if mapping.ID == "" {
		return fmt.Errorf("%w: 映射ID不能为空", core.ErrOrgMappingNotFound)
	}
	if mapping.TenantID == "" {
		return fmt.Errorf("租户ID不能为空")
	}
	if mapping.RemoteOrgValue == "" {
		return fmt.Errorf("远程组织值不能为空")
	}
	if mapping.LocalOrgID == "" {
		return fmt.Errorf("%w: 本地组织ID不能为空", core.ErrInvalidLocalOrgID)
	}

	// 2. 查询旧映射验证权限
	oldMapping, err := m.GetOrgMapping(ctx, mapping.ID)
	if err != nil {
		return err
	}

	// 3. 验证租户ID一致性（不允许修改）
	if oldMapping.TenantID != mapping.TenantID {
		return fmt.Errorf("不允许修改租户ID")
	}

	// 4. 如果修改了 local_org_id，重新验证组织是否存在
	if oldMapping.LocalOrgID != mapping.LocalOrgID {
		if err := m.validateOrgExists(ctx, mapping.LocalOrgID, mapping.TenantID); err != nil {
			return err
		}
		// 验证 local_org_id 唯一性
		if err := m.checkLocalOrgIDExists(ctx, mapping.TenantID, mapping.LocalOrgID, mapping.ID); err != nil {
			return err
		}
	}

	// 5. 如果修改了 remote_org_value，验证新的唯一约束
	if oldMapping.RemoteOrgValue != mapping.RemoteOrgValue {
		if err := m.checkMappingExists(ctx, mapping.TenantID, mapping.RemoteOrgValue, mapping.ID); err != nil {
			return err
		}
	}

	// 6. 使用事务更新数据库并更新缓存
	err = m.dbConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 更新数据库（不更新 local_org_name）
		updateQuery := `
			UPDATE event_org_mappings
			SET remote_org_value = $1, local_org_id = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3 AND tenant_id = $4
		`
		result, err := session.ExecCtx(ctx, updateQuery,
			mapping.RemoteOrgValue, mapping.LocalOrgID, mapping.ID, mapping.TenantID,
		)
		if err != nil {
			return fmt.Errorf("更新组织映射失败: %w", err)
		}

		// 检查是否有记录被更新
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("获取更新结果失败: %w", err)
		}
		if affected == 0 {
			return core.ErrOrgMappingNotFound
		}

		// 获取更新后的完整映射信息（包含组织名称）
		queryMapping := `
			SELECT
				m.id,
				m.tenant_id,
				m.remote_org_value,
				m.local_org_id,
				COALESCE(o.name, '') AS local_org_name,
				m.created_at,
				m.updated_at
			FROM event_org_mappings m
			LEFT JOIN organizations o ON m.local_org_id = o.id
			WHERE m.id = $1
		`
		var updatedMapping core.OrgMapping
		if err := session.QueryRowCtx(ctx, &updatedMapping, queryMapping, mapping.ID); err != nil {
			return fmt.Errorf("查询更新后的映射失败: %w", err)
		}

		// 更新缓存
		if m.rdb != nil {
			// 更新单个映射缓存
			if err := m.setCachedMapping(ctx, &updatedMapping); err != nil {
				logx.WithContext(ctx).Error("更新组织映射缓存失败（非致命错误）:", err)
			}
			// 删除列表缓存，触发下次查询时重新加载
			if err := m.deleteCachedMappingList(ctx, mapping.TenantID); err != nil {
				logx.WithContext(ctx).Error("删除映射列表缓存失败（非致命错误）:", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "mapping_manager"),
		logx.Field("operation", "update"),
		logx.Field("mapping_id", mapping.ID),
		logx.Field("tenant_id", mapping.TenantID),
		logx.Field("remote_org_value", mapping.RemoteOrgValue),
		logx.Field("local_org_id", mapping.LocalOrgID),
	).Info("更新组织映射成功")

	return nil
}

// DeleteOrgMapping 删除组织映射（硬删除）
func (m *mappingManager) DeleteOrgMapping(ctx context.Context, id string) error {
	// 1. 查询映射是否存在，以便记录日志
	mapping, err := m.GetOrgMapping(ctx, id)
	if err != nil {
		// 如果映射不存在，直接返回错误，避免后续操作
		if errors.Is(err, core.ErrOrgMappingNotFound) {
			return core.ErrOrgMappingNotFound
		}
		// 对于其他查询错误，也直接返回
		return fmt.Errorf("删除前获取映射信息失败: %w", err)
	}

	// 根据用户要求，移除"检查是否被事件配置使用"的逻辑

	// 2. 使用事务硬删除数据库记录并清理缓存
	err = m.dbConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 硬删除数据库记录
		deleteQuery := "DELETE FROM event_org_mappings WHERE id = $1"
		result, err := session.ExecCtx(ctx, deleteQuery, id)
		if err != nil {
			return fmt.Errorf("删除组织映射失败: %w", err)
		}

		// 检查是否有记录被删除
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("获取删除结果失败: %w", err)
		}
		if affected == 0 {
			return core.ErrOrgMappingNotFound
		}

		// 清理缓存
		if m.rdb != nil {
			// 删除单个映射缓存
			if err := m.deleteCachedMapping(ctx, id); err != nil {
				logx.WithContext(ctx).Error("删除组织映射缓存失败（非致命错误）:", err)
			}
			// 删除列表缓存，触发下次查询时重新加载
			if err := m.deleteCachedMappingList(ctx, mapping.TenantID); err != nil {
				logx.WithContext(ctx).Error("删除映射列表缓存失败（非致命错误）:", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "mapping_manager"),
		logx.Field("operation", "delete"),
		logx.Field("mapping_id", id),
		logx.Field("tenant_id", mapping.TenantID),
		logx.Field("remote_org_value", mapping.RemoteOrgValue),
	).Info("删除组织映射成功")

	return nil
}
