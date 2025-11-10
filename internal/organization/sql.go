package organization

// TableName 组织ID映射表名
const TableName = "skylark_org_mappings"

// CheckTableExistsSQL 检查表是否存在的SQL
const CheckTableExistsSQL = `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name = 'skylark_org_mappings'
`

// CreateTableSQL 创建组织ID映射表的PostgreSQL SQL语句
// 注意：新增表，用于组织/用户管理功能
const CreateTableSQL = `
CREATE TABLE IF NOT EXISTS skylark_org_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    local_org_id UUID NOT NULL,
    remote_org_id INTEGER NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, local_org_id),
    UNIQUE(tenant_id, remote_org_id)
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_skylark_org_tenant ON skylark_org_mappings(tenant_id);

-- 触发器 - 自动更新updated_at字段
CREATE OR REPLACE FUNCTION update_skylark_org_mappings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_skylark_org_mappings_updated_at
    BEFORE UPDATE ON skylark_org_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_skylark_org_mappings_updated_at();

-- 表注释
COMMENT ON TABLE skylark_org_mappings IS '组织ID映射表（管理本地组织ID与Skylark组织ID的双向映射）';
`
