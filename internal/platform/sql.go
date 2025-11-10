package platform

// TableName 平台配置表名
const TableName = "skylark_platform_configs"

// CheckTableExistsSQL 检查表是否存在的SQL
const CheckTableExistsSQL = `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name = 'skylark_platform_configs'
`

// CreateTableSQL 创建平台配置表的PostgreSQL SQL语句
// 注意：基于 nexlyn-deploy/postgres-init-scripts/008-event-handler-schema.sql
// 扩展：新增 api_base_url 和 api_token 字段用于组织/用户管理
const CreateTableSQL = `
CREATE TABLE IF NOT EXISTS skylark_platform_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE,

    -- 远程数据库连接信息
    host VARCHAR(200) NOT NULL,
    port INTEGER NOT NULL DEFAULT 5432,
    database VARCHAR(100) NOT NULL DEFAULT 'sync',
    username VARCHAR(100) NOT NULL,
    password TEXT NOT NULL,
    namespace_id INTEGER NOT NULL,

    -- API认证（用于写操作）- 新增字段
    api_base_url VARCHAR(255),
    api_token TEXT,

    -- 审计字段
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_platform_configs_tenant_id ON skylark_platform_configs(tenant_id);

-- 触发器 - 自动更新updated_at字段
CREATE OR REPLACE FUNCTION update_platform_configs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_platform_configs_updated_at
    BEFORE UPDATE ON skylark_platform_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_platform_configs_updated_at();

-- 表注释
COMMENT ON TABLE skylark_platform_configs IS 'Skylark平台对接配置表，每个租户只能配置一个平台连接';
COMMENT ON COLUMN skylark_platform_configs.api_base_url IS 'Skylark API基础地址，纯域名（如：skylark.example.com，不含https://前缀）';
COMMENT ON COLUMN skylark_platform_configs.api_token IS 'API认证Token（加密存储）';
`
