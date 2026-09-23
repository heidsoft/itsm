-- 20260923_tenant_scope_baseline_unique_keys.sql
--
-- 事件分类 code 与标签 code 原本是全局唯一，但基线数据要按租户安装：
-- 第二个租户装 hardware/urgent 必然撞全局唯一键，导致租户开通失败。
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

CREATE UNIQUE INDEX IF NOT EXISTS ticketcategory_tenant_id_code
    ON ticket_categories (tenant_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS tag_tenant_id_code
    ON tags (tenant_id, code);
