-- 20261011_add_menus_tenant_path_unique.sql
--
-- menus 表按 (tenant_id, path) 加业务唯一索引：
-- seedMenus 按 (tenant, path) upsert（先查后 Only 取行），重复 path 会让 Only 报
-- not found、菜单 reconcile 失效，前端被迫做 deduplicateMenus 兜底；
-- CreateMenu API 原先不查重，并发/误用即可造出重复行。
--
-- 执行顺序：bootstrap 先跑 ent client.Schema.Create（按新 schema 建好唯一索引），
-- 再跑本文件。CREATE 只对「跳过 auto-migrate、只跑版本化迁移」的库生效，写成幂等。

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT tenant_id, path
        FROM menus
        GROUP BY tenant_id, path
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'menus has % duplicated (tenant_id, path) group(s); deduplicate before adding the unique index',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS menu_tenant_id_path
    ON menus (tenant_id, path);
