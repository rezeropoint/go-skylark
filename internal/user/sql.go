package user

// TableName 用户ID映射表名
const TableName = "skylark_user_mappings"

// CheckTableExistsSQL 检查表是否存在的SQL
const CheckTableExistsSQL = `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name = 'skylark_user_mappings'
`

// CreateTableSQL 创建用户ID映射表的PostgreSQL SQL语句
// 注意：新增表，用于组织/用户管理功能
const CreateTableSQL = `
CREATE TABLE IF NOT EXISTS skylark_user_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    local_user_id UUID NOT NULL,
    remote_user_id INTEGER NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, local_user_id),
    UNIQUE(tenant_id, remote_user_id)
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_skylark_user_tenant ON skylark_user_mappings(tenant_id);

-- 触发器 - 自动更新updated_at字段
CREATE OR REPLACE FUNCTION update_skylark_user_mappings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_skylark_user_mappings_updated_at
    BEFORE UPDATE ON skylark_user_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_skylark_user_mappings_updated_at();

-- 表注释
COMMENT ON TABLE skylark_user_mappings IS '用户ID映射表（管理本地用户ID与Skylark用户ID的双向映射）';
`
