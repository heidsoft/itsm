-- 20261013_tenant_scope_ticket_number_unique_down.sql
--
-- 回滚 20261013_tenant_scope_ticket_number_unique.sql：恢复 ticket_number 全局唯一。
--
-- 这条回滚是「有前提的」：本次放宽之后，多租户库里合法存在多个租户持有同一个
-- TKT-YYYYMM-NNNNNN（每个租户各自从 000001 起算），直接重建全局唯一键必然失败。
-- 因此回滚前必须先人工消解跨租户重号，本文件只做检测并拒绝，不自动改数据。
-- 前提不成立时正确处置是前滚修复，不是回滚。

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT ticket_number
        FROM tickets
        WHERE ticket_number IS NOT NULL AND ticket_number <> ''
        GROUP BY ticket_number
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'tickets has % globally duplicated ticket_number(s) across tenants; resolve them before restoring the global unique constraint',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

DROP INDEX IF EXISTS ticket_tenant_id_ticket_number;

-- 恢复放宽前的两个全局唯一对象：ent 字段级 Unique() 的列约束 + 显式唯一索引。
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_ticket_number_key;
ALTER TABLE tickets ADD CONSTRAINT tickets_ticket_number_key UNIQUE (ticket_number);
CREATE UNIQUE INDEX IF NOT EXISTS ticket_ticket_number ON tickets (ticket_number);
