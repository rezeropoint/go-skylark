package mapping

// TableName 组织映射表名（业务字段值映射）
const TableName = "event_org_mappings"

// CheckTableExistsSQL 检查表是否存在的SQL
const CheckTableExistsSQL = `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name = 'event_org_mappings'
`

// CreateTableSQL 创建组织映射表的PostgreSQL SQL语句
// 注意：完全基于 nexlyn-deploy/postgres-init-scripts/008-event-handler-schema.sql
const CreateTableSQL = `
CREATE TABLE IF NOT EXISTS event_org_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    remote_org_value VARCHAR(200) NOT NULL,
    local_org_id UUID NOT NULL,
    local_org_name VARCHAR(200),
    tenant_id UUID NOT NULL,

    -- 审计字段
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 唯一约束
    CONSTRAINT uk_org_mapping_tenant_remote UNIQUE (tenant_id, remote_org_value)
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_org_mappings_local_org_id ON event_org_mappings(local_org_id);
CREATE INDEX IF NOT EXISTS idx_org_mappings_remote_org_value ON event_org_mappings(remote_org_value);
CREATE INDEX IF NOT EXISTS idx_org_mappings_tenant_id ON event_org_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_org_mappings_tenant_local_org ON event_org_mappings(tenant_id, local_org_id);

-- 触发器
CREATE OR REPLACE FUNCTION update_event_org_mappings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_event_org_mappings_updated_at
    BEFORE UPDATE ON event_org_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_event_org_mappings_updated_at();

-- 表注释
COMMENT ON TABLE event_org_mappings IS '组织映射表（平台级别共享），将远程表组织字段值映射到本地组织ID，多个事件类型可以共享同一映射';
`
