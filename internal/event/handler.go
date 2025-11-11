package event

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// eventManager 事件+字段配置管理器
type eventManager struct {
	dbConn      sqlx.SqlConn         // 本地数据库连接
	getRemoteDB core.GetRemoteDBFunc // 获取远程数据库连接的函数（用于验证flow_id、字段名等）
}

// newEventManager 创建事件+字段配置管理器
func newEventManager(db sqlx.SqlConn, getRemoteDB core.GetRemoteDBFunc) (*eventManager, error) {
	if getRemoteDB == nil {
		return nil, fmt.Errorf("getRemoteDB 函数不能为空")
	}

	manager := &eventManager{
		dbConn:      db,
		getRemoteDB: getRemoteDB,
	}

	// 初始化数据库表（在包初始化时执行）
	ctx := context.Background()
	if err := manager.initTable(ctx); err != nil {
		return nil, fmt.Errorf("初始化事件配置表失败: %w", err)
	}

	return manager, nil
}

// CreateWithFields 创建事件配置（包含字段）
func (m *eventManager) CreateWithFields(ctx context.Context, req *core.CreateEventRequest) (string, error) {
	// 1. 验证基本信息
	if req.EventConfig.ID != "" {
		return "", fmt.Errorf("%w: 创建时不允许指定ID，系统会自动生成", core.ErrInvalidEventConfig)
	}

	if err := m.validateEventConfig(&req.EventConfig); err != nil {
		return "", err
	}
	if len(req.Fields) == 0 {
		return "", fmt.Errorf("%w: 至少需要配置一个字段", core.ErrInvalidEventConfig)
	}

	// 2. 验证字段配置
	for i := range req.Fields {
		if err := m.validateFieldConfig(&req.Fields[i]); err != nil {
			return "", fmt.Errorf("字段配置[%d]无效: %w", i, err)
		}
	}

	// 3. 生成事件配置ID
	req.EventConfig.ID = uuid.New().String()

	var createdID string

	// 4. 使用事务创建
	err := m.dbConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 4.1 检查事件名称是否已存在（租户内唯一）
		var existingID string
		checkQuery := "SELECT id FROM event_configs WHERE name = $1 AND tenant_id = $2"
		err := session.QueryRowCtx(ctx, &existingID, checkQuery, req.EventConfig.Name, req.EventConfig.TenantID)
		if err == nil {
			return fmt.Errorf("%w: 事件名称 '%s' 已存在", core.ErrInvalidEventConfig, req.EventConfig.Name)
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("检查事件名称失败: %w", err)
		}

		// 4.2 检查flow_id 是否已被使用（租户内唯一）
		checkFlowQuery := "SELECT id FROM event_configs WHERE flow_id = $1 AND tenant_id = $2"
		err = session.QueryRowCtx(ctx, &existingID, checkFlowQuery, req.EventConfig.FlowID, req.EventConfig.TenantID)
		if err == nil {
			return fmt.Errorf("%w: flow_id %d 已被事件配置 %s 使用", core.ErrInvalidEventConfig, req.EventConfig.FlowID, existingID)
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("检查flow_id 失败: %w", err)
		}

		// 4.3 插入事件配置
		id, err := m.insertEvent(ctx, session, &req.EventConfig)
		if err != nil {
			return err
		}
		createdID = id

		// 4.4 为字段配置填event_config_id
		for i := range req.Fields {
			req.Fields[i].EventConfigID = createdID
		}

		// 4.5 批量插入字段配置
		if err := m.insertFields(ctx, session, req.Fields); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "event_manager"),
		logx.Field("operation", "create_with_fields"),
		logx.Field("event_config_id", createdID),
		logx.Field("tenant_id", req.EventConfig.TenantID),
		logx.Field("field_count", len(req.Fields)),
	).Info("创建事件配置成功（包含字段）")

	// 5. 查询返回完整的事件配置
	return createdID, nil
}

// UpdateWithFields 更新事件配置（包含字段）
func (m *eventManager) UpdateWithFields(ctx context.Context, req *core.UpdateEventRequest) error {
	// 1. 验证基本信息
	if req.EventConfig.ID == "" {
		return fmt.Errorf("%w: 事件配置ID不能为空", core.ErrInvalidEventConfig)
	}
	if err := m.validateEventConfig(&req.EventConfig); err != nil {
		return err
	}
	if len(req.Fields) == 0 {
		return fmt.Errorf("%w: 至少需要配置一个字段", core.ErrInvalidEventConfig)
	}

	// 2. 验证字段配置
	for i := range req.Fields {
		if err := m.validateFieldConfig(&req.Fields[i]); err != nil {
			return fmt.Errorf("字段配置[%d]无效: %w", i, err)
		}
	}

	// 3. 使用事务更新（完整替换策略）
	err := m.dbConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 3.1 验证事件配置存在且有权限
		var existingTenantID string
		checkQuery := "SELECT tenant_id FROM event_configs WHERE id = $1"
		err := session.QueryRowCtx(ctx, &existingTenantID, checkQuery, req.EventConfig.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return core.ErrEventConfigNotFound
			}
			return fmt.Errorf("查询事件配置失败: %w", err)
		}

		// 3.2 验证租户ID一致
		if existingTenantID != req.EventConfig.TenantID {
			return fmt.Errorf("%w: 不允许修改租户ID", core.ErrInvalidEventConfig)
		}

		// 3.3 检查事件名称是否与其他事件冲突
		var conflictID string
		checkNameQuery := "SELECT id FROM event_configs WHERE name = $1 AND tenant_id = $2 AND id != $3"
		err = session.QueryRowCtx(ctx, &conflictID, checkNameQuery, req.EventConfig.Name, req.EventConfig.TenantID, req.EventConfig.ID)
		if err == nil {
			return fmt.Errorf("%w: 事件名称 '%s' 已被其他事件使用", core.ErrInvalidEventConfig, req.EventConfig.Name)
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("检查事件名称失败: %w", err)
		}

		// 3.4 检查flow_id 是否与其他事件冲突
		checkFlowQuery := "SELECT id FROM event_configs WHERE flow_id = $1 AND tenant_id = $2 AND id != $3"
		err = session.QueryRowCtx(ctx, &conflictID, checkFlowQuery, req.EventConfig.FlowID, req.EventConfig.TenantID, req.EventConfig.ID)
		if err == nil {
			return fmt.Errorf("%w: flow_id %d 已被事件配置 %s 使用", core.ErrInvalidEventConfig, req.EventConfig.FlowID, conflictID)
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("检查flow_id 失败: %w", err)
		}

		// 3.5 更新事件配置基本信息
		if err := m.updateEvent(ctx, session, &req.EventConfig); err != nil {
			return err
		}

		// 3.6 删除所有旧字段配置（完整替换策略）
		if err := m.deleteFieldsByEventID(ctx, session, req.EventConfig.ID); err != nil {
			return err
		}

		// 3.7 为字段配置填event_config_id
		for i := range req.Fields {
			req.Fields[i].EventConfigID = req.EventConfig.ID
		}

		// 3.8 批量插入新字段配置
		if err := m.insertFields(ctx, session, req.Fields); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "event_manager"),
		logx.Field("operation", "update_with_fields"),
		logx.Field("event_config_id", req.EventConfig.ID),
		logx.Field("tenant_id", req.EventConfig.TenantID),
		logx.Field("field_count", len(req.Fields)),
	).Info("更新事件配置成功（包含字段）")

	return nil
}

// GetWithFields 查询事件配置（包含字段）
func (m *eventManager) GetWithFields(ctx context.Context, id, tenantID string) (*core.EventConfigWithFields, error) {
	// 1. 查询事件配置
	event, err := m.get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. 查询字段配置列表
	fields, err := m.listFieldsByEventID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. 组装返回
	return &core.EventConfigWithFields{
		EventConfig: *event,
		Fields:      fields,
	}, nil
}

// ListWithFields 查询事件配置列表（包含字段）
func (m *eventManager) ListWithFields(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventConfigWithFields, error) {
	// 1. 查询事件配置列表
	events, err := m.list(ctx, tenantID, enabled)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return []*core.EventConfigWithFields{}, nil
	}

	// 2. 提取所有event_config_id
	eventIDs := make([]string, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}

	// 3. 批量查询所有字段配置（一次SQL查询，性能优化）
	fieldsMap, err := m.listFieldsByEventIDs(ctx, eventIDs)
	if err != nil {
		return nil, err
	}

	// 4. 组装返回结果
	result := make([]*core.EventConfigWithFields, len(events))
	for i, event := range events {
		fields, exists := fieldsMap[event.ID]
		if !exists {
			fields = []*core.FieldConfig{}
		}
		result[i] = &core.EventConfigWithFields{
			EventConfig: *event,
			Fields:      fields,
		}
	}

	return result, nil
}

// Delete 删除事件配置（硬删除）
func (m *eventManager) Delete(ctx context.Context, id, tenantID string) error {
	// 硬删除事件配置
	deleteQuery := `
		DELETE FROM event_configs
		WHERE id = $1 AND tenant_id = $2
	`
	result, err := m.dbConn.ExecCtx(ctx, deleteQuery, id, tenantID)
	if err != nil {
		return fmt.Errorf("删除事件配置失败: %w", err)
	}

	// 检查是否有记录被删除
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}
	if affected == 0 {
		return core.ErrEventConfigNotFound
	}

	// 注意：字段配置会通过数据库外键级联删除（ON DELETE CASCADE）

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "event_manager"),
		logx.Field("operation", "delete"),
		logx.Field("event_config_id", id),
		logx.Field("tenant_id", tenantID),
	).Info("删除事件配置成功")

	return nil
}

// Update 更新事件配置（不含字段）
func (m *eventManager) Update(ctx context.Context, config *core.EventConfig) error {
	// 验证配置
	if config.ID == "" || config.TenantID == "" {
		return fmt.Errorf("%w: ID和租户ID不能为空", core.ErrInvalidEventConfig)
	}
	if err := m.validateEventConfig(config); err != nil {
		return err
	}

	// 查询旧配置验证权限
	var oldTenantID string
	checkQuery := "SELECT tenant_id FROM event_configs WHERE id = $1"
	err := m.dbConn.QueryRowCtx(ctx, &oldTenantID, checkQuery, config.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return core.ErrEventConfigNotFound
		}
		return fmt.Errorf("查询事件配置失败: %w", err)
	}

	// 验证租户ID一致
	if oldTenantID != config.TenantID {
		return fmt.Errorf("%w: 不允许修改租户ID", core.ErrInvalidEventConfig)
	}

	// 检查事件名称是否与其他事件冲突
	var conflictID string
	checkNameQuery := "SELECT id FROM event_configs WHERE name = $1 AND tenant_id = $2 AND id != $3"
	err = m.dbConn.QueryRowCtx(ctx, &conflictID, checkNameQuery, config.Name, config.TenantID, config.ID)
	if err == nil {
		return fmt.Errorf("%w: 事件名称 '%s' 已被其他事件使用", core.ErrInvalidEventConfig, config.Name)
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("检查事件名称失败: %w", err)
	}

	// 检查flow_id 是否与其他事件冲突
	checkFlowQuery := "SELECT id FROM event_configs WHERE flow_id = $1 AND tenant_id = $2 AND id != $3"
	err = m.dbConn.QueryRowCtx(ctx, &conflictID, checkFlowQuery, config.FlowID, config.TenantID, config.ID)
	if err == nil {
		return fmt.Errorf("%w: flow_id %d 已被事件配置 %s 使用", core.ErrInvalidEventConfig, config.FlowID, conflictID)
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("检查flow_id 失败: %w", err)
	}

	// 更新数据
	updateQuery := `
		UPDATE event_configs
		SET name = $1, flow_id = $2, flow_title = $3, org_field_name = $4,
		    description = $5, enabled = $6, updated_by = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND tenant_id = $9
	`
	result, err := m.dbConn.ExecCtx(ctx, updateQuery,
		config.Name, config.FlowID, config.FlowTitle, config.OrgFieldName,
		config.Description, config.Enabled, config.UpdatedBy,
		config.ID, config.TenantID,
	)
	if err != nil {
		return fmt.Errorf("更新事件配置失败: %w", err)
	}

	// 检查是否有记录被更
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取更新结果失败: %w", err)
	}
	if affected == 0 {
		return core.ErrEventConfigNotFound
	}

	logx.WithContext(ctx).WithFields(
		logx.Field("module", "event_manager"),
		logx.Field("operation", "update"),
		logx.Field("event_config_id", config.ID),
		logx.Field("tenant_id", config.TenantID),
	).Info("更新事件配置成功")

	return nil
}

// initTable 初始化事件配置相关表（event_configs + event_field_configs）（私有方法，在包初始化时调用）
func (m *eventManager) initTable(ctx context.Context) error {
	// 1. 检查并创建事件配置表
	var eventTableCount int
	err := m.dbConn.QueryRowCtx(ctx, &eventTableCount, CheckEventConfigTableExistsSQL)
	if err != nil {
		return fmt.Errorf("检查事件配置表存在性失败: %w", err)
	}

	if eventTableCount == 0 {
		_, err = m.dbConn.ExecCtx(ctx, CreateEventConfigTableSQL)
		if err != nil {
			return fmt.Errorf("创建事件配置表失败: %w", err)
		}
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "event_manager"),
			logx.Field("operation", "init_table"),
			logx.Field("table", EventConfigTableName),
		).Info("事件配置表初始化成功")
	}

	// 2. 检查并创建字段配置表
	var fieldTableCount int
	err = m.dbConn.QueryRowCtx(ctx, &fieldTableCount, CheckFieldConfigTableExistsSQL)
	if err != nil {
		return fmt.Errorf("检查字段配置表存在性失败: %w", err)
	}

	if fieldTableCount == 0 {
		_, err = m.dbConn.ExecCtx(ctx, CreateFieldConfigTableSQL)
		if err != nil {
			return fmt.Errorf("创建字段配置表失败: %w", err)
		}
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "event_manager"),
			logx.Field("operation", "init_table"),
			logx.Field("table", FieldConfigTableName),
		).Info("字段配置表初始化成功")
	}

	return nil
}
