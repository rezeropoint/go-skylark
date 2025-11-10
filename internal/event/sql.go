package event

// EventConfigTableName 事件配置表名
const EventConfigTableName = "event_configs"

// FieldConfigTableName 事件字段配置表名
const FieldConfigTableName = "event_field_configs"

// CheckEventConfigTableExistsSQL 检查事件配置表是否存在的SQL
const CheckEventConfigTableExistsSQL = `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name = 'event_configs'
`

// CheckFieldConfigTableExistsSQL 检查字段配置表是否存在的SQL
const CheckFieldConfigTableExistsSQL = `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name = 'event_field_configs'
`

// CreateEventConfigTableSQL 创建事件配置表的PostgreSQL SQL语句
// 注意：完全基于 nexlyn-deploy/postgres-init-scripts/008-event-handler-schema.sql
const CreateEventConfigTableSQL = `
CREATE TABLE IF NOT EXISTS event_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    flow_id INTEGER NOT NULL,
    flow_title VARCHAR(300),
    org_field_name VARCHAR(100),
    description TEXT,

    -- 状态控制
    enabled BOOLEAN DEFAULT TRUE,

    -- 多租户支持
    tenant_id UUID NOT NULL,

    -- 审计字段
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 唯一约束
    CONSTRAINT uk_event_configs_name_tenant UNIQUE (name, tenant_id),
    CONSTRAINT uk_event_configs_flow_tenant UNIQUE (flow_id, tenant_id)
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_event_configs_name ON event_configs(name);
CREATE INDEX IF NOT EXISTS idx_event_configs_flow_id ON event_configs(flow_id);
CREATE INDEX IF NOT EXISTS idx_event_configs_tenant_id ON event_configs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_event_configs_enabled ON event_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_event_configs_created_at ON event_configs(created_at);
CREATE INDEX IF NOT EXISTS idx_event_configs_tenant_enabled ON event_configs(tenant_id, enabled);

-- 触发器
CREATE OR REPLACE FUNCTION update_event_configs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_event_configs_updated_at
    BEFORE UPDATE ON event_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_event_configs_updated_at();

-- 表注释
COMMENT ON TABLE event_configs IS '事件配置表，存储Skylark平台事件的配置信息';
`

// CreateFieldConfigTableSQL 创建事件字段配置表的PostgreSQL SQL语句
// 注意：完全基于 nexlyn-deploy/postgres-init-scripts/008-event-handler-schema.sql
const CreateFieldConfigTableSQL = `
CREATE TABLE IF NOT EXISTS event_field_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_config_id UUID NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200) NOT NULL,
    field_type VARCHAR(50) NOT NULL,
    is_visible BOOLEAN DEFAULT TRUE,
    display_order INTEGER DEFAULT 0,
    is_searchable BOOLEAN DEFAULT FALSE,

    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 外键约束
    CONSTRAINT fk_field_config_event FOREIGN KEY (event_config_id)
        REFERENCES event_configs(id)
        ON DELETE CASCADE,

    -- 唯一约束
    CONSTRAINT uk_field_config_event_field UNIQUE (event_config_id, field_name)
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_field_configs_event_id ON event_field_configs(event_config_id);
CREATE INDEX IF NOT EXISTS idx_field_configs_field_name ON event_field_configs(field_name);
CREATE INDEX IF NOT EXISTS idx_field_configs_is_visible ON event_field_configs(is_visible);
CREATE INDEX IF NOT EXISTS idx_field_configs_display_order ON event_field_configs(display_order);
CREATE INDEX IF NOT EXISTS idx_field_configs_event_visible_order ON event_field_configs(event_config_id, is_visible, display_order);

-- 触发器
CREATE OR REPLACE FUNCTION update_event_field_configs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_event_field_configs_updated_at
    BEFORE UPDATE ON event_field_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_event_field_configs_updated_at();

-- 表注释
COMMENT ON TABLE event_field_configs IS '事件字段配置表，定义事件数据的展示字段';
`
