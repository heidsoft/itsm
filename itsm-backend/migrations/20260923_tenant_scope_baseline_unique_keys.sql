-- 20260923_tenant_scope_baseline_unique_keys.sql
--
-- 事件分类 code、标签 code 与流程部署 deployment_id 原本是全局唯一，但基线数据要按租户安装：
-- 第二个租户装 hardware/urgent 或部署 <key>-v1 必然撞全局唯一键，导致租户开通失败。
-- 这里放宽为「按租户组合唯一」，与 AGENTS.md「业务唯一键默认按 tenant 组合唯一」一致。
-- 方向是放宽约束，不会让原本合法的存量数据变得非法；仍在建索引前显式检测组内重复。

ALTER TABLE ticket_categories DROP CONSTRAINT IF EXISTS ticket_categories_code_key;
DROP INDEX IF EXISTS ticket_categories_code_key;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS tags_code_key;
DROP INDEX IF EXISTS tags_code_key;

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT tenant_id, code
        FROM ticket_categories
        GROUP BY tenant_id, code
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'ticket_categories has % duplicated (tenant_id, code) group(s); deduplicate before adding the unique index',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;

    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT tenant_id, deployment_id
        FROM process_deployments
        GROUP BY tenant_id, deployment_id
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'process_deployments has % duplicated (tenant_id, deployment_id) group(s); deduplicate before adding the unique index',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;

    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT tenant_id, code
        FROM tags
        GROUP BY tenant_id, code
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'tags has % duplicated (tenant_id, code) group(s); deduplicate before adding the unique index',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

-- process_deployments 同理：内置流程模板在每个租户使用同名 deployment_id，
-- 全局唯一会让第二个租户部署失败。
ALTER TABLE process_deployments DROP CONSTRAINT IF EXISTS process_deployments_deployment_id_key;
DROP INDEX IF EXISTS process_deployments_deployment_id_key;
DROP INDEX IF EXISTS processdeployment_deployment_id;

CREATE UNIQUE INDEX IF NOT EXISTS ticketcategory_tenant_id_code
    ON ticket_categories (tenant_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS tag_tenant_id_code
    ON tags (tenant_id, code);

CREATE UNIQUE INDEX IF NOT EXISTS processdeployment_tenant_id_deployment_id
    ON process_deployments (tenant_id, deployment_id);
