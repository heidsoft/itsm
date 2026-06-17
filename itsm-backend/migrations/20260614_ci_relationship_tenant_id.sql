-- SEC-001 修复：为 ci_relationships 表添加 tenant_id 字段
-- 背景：原表缺少租户隔离字段，导致跨租户数据泄露风险

-- 1. 添加 tenant_id 列（允许 NULL 以兼容存量数据）
ALTER TABLE ci_relationships ADD COLUMN IF NOT EXISTS tenant_id INTEGER;

-- 2. 为存量数据填充 tenant_id（从 source_ci 的 tenant_id 继承）
UPDATE ci_relationships
SET tenant_id = (
    SELECT ci.tenant_id
    FROM configuration_items ci
    WHERE ci.id = ci_relationships.source_ci_id
)
WHERE ci_relationships.tenant_id IS NULL;

-- 3. 添加索引
CREATE INDEX IF NOT EXISTS idx_ci_relationships_tenant_id ON ci_relationships(tenant_id);

-- 4. 添加外键约束（可选，确保数据完整性）
-- ALTER TABLE ci_relationships
--     ADD CONSTRAINT fk_ci_relationships_tenant
--     FOREIGN KEY (tenant_id) REFERENCES tenants(id);
