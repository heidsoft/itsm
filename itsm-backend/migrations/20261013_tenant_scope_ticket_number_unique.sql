-- 20261013_tenant_scope_ticket_number_unique.sql
--
-- tickets.ticket_number 原本是「字段级全局唯一 + 全局唯一索引 ticket_ticket_number」，
-- 但编号生成走的是「该租户当月 max+1」（repository/ticket/ticket_number_repository.go
-- 的存在性检查也已经是 WHERE ticket_number=$1 AND tenant_id=$2）。生成域按租户、
-- 约束域按全局，两者不一致的后果是：
--   租户 A 当月首单拿到 TKT-YYYYMM-000001 之后，新开通的租户 B 当月首单同样算出
--   000001，撞全局唯一索引；重试路径每次仍算出同一个号，3 次后建单永久失败。
--   即 MSP 场景下新客户租户在当月无法建单。
-- 这里放宽为「按租户组合唯一」，与 AGENTS.md「业务唯一键默认按 tenant 组合唯一」一致，
-- 也与 20261006_tenant_scope_user_username_unique.sql（users.username）、
-- 20261011_add_menus_tenant_path_unique.sql（menus）同一处置口径。
--
-- 执行顺序说明：bootstrap 先跑 ent 的 client.Schema.Create（它会按新 schema 建好
-- ticket_tenant_id_ticket_number），再跑本文件。所以本文件的职责是删掉遗留的两个
-- 全局唯一对象；CREATE 语句只对「跳过 auto-migrate、只跑版本化迁移」的库生效，
-- 两者都写成幂等。

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT tenant_id, ticket_number
        FROM tickets
        WHERE ticket_number IS NOT NULL AND ticket_number <> ''
        GROUP BY tenant_id, ticket_number
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'tickets has % duplicated (tenant_id, ticket_number) group(s); deduplicate before adding the unique index',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

-- 旧的全局唯一键有两个来源：ent 字段级 Unique() 生成的列约束，
-- 以及 index.Fields("ticket_number").Unique() 生成的显式索引。二者都只在存在时才删。
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_ticket_number_key;
DROP INDEX IF EXISTS ticket_ticket_number;

CREATE UNIQUE INDEX IF NOT EXISTS ticket_tenant_id_ticket_number
    ON tickets (tenant_id, ticket_number);
