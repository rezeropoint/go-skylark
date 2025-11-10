package event

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Get 查询事件配置（不含字段）
func (m *eventManager) get(ctx context.Context, id, tenantID string) (*core.EventConfig, error) {
	query := `
        SELECT id, name, flow_id, flow_title, org_field_name, description, enabled,
               tenant_id, created_by, updated_by, created_at, updated_at
        FROM event_configs
        WHERE id = $1 AND tenant_id = $2
    `

	var config core.EventConfig
	err := m.dbConn.QueryRowCtx(ctx, &config, query, id, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrEventConfigNotFound
		}
		return nil, fmt.Errorf("查询事件配置失败: %w", err)
	}

	return &config, nil
}

// List 查询事件配置列表（不含字段）
func (m *eventManager) list(ctx context.Context, tenantID string, enabled *bool) ([]*core.EventConfig, error) {
	query := `
        SELECT id, name, flow_id, flow_title, org_field_name, description, enabled,
               tenant_id, created_by, updated_by, created_at, updated_at
        FROM event_configs
        WHERE tenant_id = $1
    `

	args := []interface{}{tenantID}

	// 根据 enabled 参数过滤
	if enabled != nil {
		query += " AND enabled = $2"
		args = append(args, *enabled)
	}

	query += " ORDER BY created_at DESC"

	var configs []*core.EventConfig
	err := m.dbConn.QueryRowsCtx(ctx, &configs, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询事件配置列表失败: %w", err)
	}

	return configs, nil
}

// 事件配置内部方法

// insertEvent 插入事件配置（事务内使用）
func (m *eventManager) insertEvent(ctx context.Context, session sqlx.Session, config *core.EventConfig) (string, error) {

	if config.ID == "" {
		return "", fmt.Errorf("id不能为空")
	}

	insertQuery := `
        INSERT INTO event_configs
        (id, name, flow_id, flow_title, org_field_name, description, enabled, tenant_id, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
	_, err := session.ExecCtx(ctx, insertQuery,
		config.ID, config.Name, config.FlowID, config.FlowTitle,
		config.OrgFieldName, config.Description, config.Enabled,
		config.TenantID, config.CreatedBy,
	)
	if err != nil {
		return "", fmt.Errorf("插入事件配置失败: %w", err)
	}

	return config.ID, nil
}

// updateEvent 更新事件配置（事务内使用）
func (m *eventManager) updateEvent(ctx context.Context, session sqlx.Session, config *core.EventConfig) error {
	updateQuery := `
        UPDATE event_configs
        SET name = $1, flow_id = $2, flow_title = $3, org_field_name = $4,
            description = $5, enabled = $6, updated_by = $7, updated_at = CURRENT_TIMESTAMP
        WHERE id = $8 AND tenant_id = $9
    `
	_, err := session.ExecCtx(ctx, updateQuery,
		config.Name, config.FlowID, config.FlowTitle, config.OrgFieldName,
		config.Description, config.Enabled, config.UpdatedBy,
		config.ID, config.TenantID,
	)
	if err != nil {
		return fmt.Errorf("更新事件配置失败: %w", err)
	}

	return nil
}

// validateEventConfig 验证事件配置基本信息
func (m *eventManager) validateEventConfig(config *core.EventConfig) error {
	if config.TenantID == "" {
		return fmt.Errorf("%w: 租户ID不能为空", core.ErrInvalidEventConfig)
	}
	if config.Name == "" {
		return fmt.Errorf("%w: 事件名称不能为空", core.ErrInvalidEventConfig)
	}
	if config.FlowID <= 0 {
		return fmt.Errorf("%w: FlowID 必须大于0", core.ErrInvalidEventConfig)
	}

	return nil
}

// insertFields 批量插入字段配置（事务内使用）
func (m *eventManager) insertFields(ctx context.Context, session sqlx.Session, fields []core.FieldConfig) error {
	if len(fields) == 0 {
		return nil
	}

	// 构建批量插入SQL
	// 使用 VALUES ($1, $2, ...), ($9, $10, ...) 格式
	valueStrings := make([]string, 0, len(fields))
	valueArgs := make([]interface{}, 0, len(fields)*8)

	for i, field := range fields {
		// 生成UUID（如果没有）
		if field.ID == "" {
			fields[i].ID = uuid.New().String()
		}

		// 每个字段8个参数
		offset := i * 8
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7, offset+8))

		valueArgs = append(valueArgs,
			fields[i].ID,
			fields[i].EventConfigID,
			fields[i].FieldName,
			fields[i].DisplayName,
			fields[i].FieldType,
			fields[i].IsVisible,
			fields[i].DisplayOrder,
			fields[i].IsSearchable,
		)
	}

	insertQuery := fmt.Sprintf(`
        INSERT INTO event_field_configs
        (id, event_config_id, field_name, display_name, field_type, is_visible, display_order, is_searchable)
        VALUES %s
    `, strings.Join(valueStrings, ", "))

	_, err := session.ExecCtx(ctx, insertQuery, valueArgs...)
	if err != nil {
		return fmt.Errorf("批量插入字段配置失败: %w", err)
	}

	return nil
}

// deleteFieldsByEventID 删除指定事件的所有字段配置（事务内使用）
func (m *eventManager) deleteFieldsByEventID(ctx context.Context, session sqlx.Session, eventConfigID string) error {
	deleteQuery := "DELETE FROM event_field_configs WHERE event_config_id = $1"
	_, err := session.ExecCtx(ctx, deleteQuery, eventConfigID)
	if err != nil {
		return fmt.Errorf("删除字段配置失败: %w", err)
	}
	return nil
}

// listFieldsByEventID 查询指定事件的所有字段配置（按display_order排序）
func (m *eventManager) listFieldsByEventID(ctx context.Context, eventConfigID string) ([]*core.FieldConfig, error) {
	query := `
        SELECT id, event_config_id, field_name, display_name, field_type,
               is_visible, display_order, is_searchable, created_at, updated_at
        FROM event_field_configs
        WHERE event_config_id = $1
        ORDER BY display_order ASC, created_at ASC
    `

	var fields []*core.FieldConfig
	err := m.dbConn.QueryRowsCtx(ctx, &fields, query, eventConfigID)
	if err != nil {
		return nil, fmt.Errorf("查询字段配置失败: %w", err)
	}

	return fields, nil
}

// listFieldsByEventIDs 批量查询多个事件的字段配置（性能优化）
func (m *eventManager) listFieldsByEventIDs(ctx context.Context, eventConfigIDs []string) (map[string][]*core.FieldConfig, error) {
	if len(eventConfigIDs) == 0 {
		return make(map[string][]*core.FieldConfig), nil
	}

	// 使用 ANY($1) 批量查询
	query := `
        SELECT id, event_config_id, field_name, display_name, field_type,
               is_visible, display_order, is_searchable, created_at, updated_at
        FROM event_field_configs
        WHERE event_config_id = ANY($1)
        ORDER BY event_config_id, display_order ASC, created_at ASC
    `

	var fields []*core.FieldConfig
	err := m.dbConn.QueryRowsCtx(ctx, &fields, query, pq.Array(eventConfigIDs))
	if err != nil {
		return nil, fmt.Errorf("批量查询字段配置失败: %w", err)
	}

	// 按 event_config_id 分组
	fieldsMap := make(map[string][]*core.FieldConfig)
	for _, field := range fields {
		fieldsMap[field.EventConfigID] = append(fieldsMap[field.EventConfigID], field)
	}

	return fieldsMap, nil
}

// validateFieldConfig 验证字段配置
func (m *eventManager) validateFieldConfig(field *core.FieldConfig) error {
	// 验证字段名
	if field.FieldName == "" {
		return fmt.Errorf("%w: 字段名不能为空", core.ErrInvalidFieldName)
	}
	if !isValidFieldName(field.FieldName) {
		return fmt.Errorf("%w: 字段名 '%s' 格式无效（只允许字母、数字、下划线）", core.ErrInvalidFieldName, field.FieldName)
	}

	// 验证显示名称
	if field.DisplayName == "" {
		return fmt.Errorf("%w: 显示名称不能为空", core.ErrInvalidFieldType)
	}

	// 验证字段类型
	if !core.IsValidFieldType(field.FieldType) {
		return fmt.Errorf("%w: 字段类型 '%s' 无效", core.ErrInvalidFieldType, field.FieldType)
	}

	return nil
}
