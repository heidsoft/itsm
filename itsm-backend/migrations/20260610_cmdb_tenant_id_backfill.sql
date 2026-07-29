-- 租户隔离硬化：回填 ci_relationships / discovery_sources 的空 tenant_id 并加 NOT NULL 约束
-- 必须在部署包含 tenant_id 必填 schema（ent/schema/ci_relationship.go、ent/schema/discovery_source.go）
-- 的后端版本之前执行。

-- 1. ci_relationships：优先从源 CI 继承租户，其次目标 CI，最后兜底到最小租户ID
UPDATE ci_relationships cr
SET tenant_id = ci.tenant_id
FROM configuration_items ci
WHERE cr.tenant_id IS NULL
  AND ci.id = cr.source_ci_id;

UPDATE ci_relationships cr
SET tenant_id = ci.tenant_id
FROM configuration_items ci
WHERE cr.tenant_id IS NULL
  AND ci.id = cr.target_ci_id;

-- 极端情况：两端 CI 均已不存在的孤儿关系，兜底归入最小租户（通常为默认租户）
UPDATE ci_relationships
SET tenant_id = (SELECT MIN(id) FROM tenants)
WHERE tenant_id IS NULL;

ALTER TABLE ci_relationships
ALTER COLUMN tenant_id SET NOT NULL;

-- 2. discovery_sources：历史上作为"全局"发现源写入的空租户记录，归入最小租户
UPDATE discovery_sources
SET tenant_id = (SELECT MIN(id) FROM tenants)
WHERE tenant_id IS NULL;

ALTER TABLE discovery_sources
ALTER COLUMN tenant_id SET NOT NULL;

COMMENT ON COLUMN ci_relationships.tenant_id IS '租户ID（必填，跨租户隔离）';
COMMENT ON COLUMN discovery_sources.tenant_id IS '租户ID（必填，跨租户隔离）';
