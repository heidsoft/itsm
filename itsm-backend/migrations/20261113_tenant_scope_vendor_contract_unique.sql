-- 20261113_tenant_scope_vendor_contract_unique.sql
--
-- vendors.code 与 contracts.contract_number 原本是 ent 字段级全局唯一。
-- 实测生产形态：ent 在 PostgreSQL 上把字段级 Unique 物化为独立唯一索引
-- vendors_code_key / contracts_contract_number_key（pg_constraint 无对应行，
-- 不是列约束）。两张表都带 tenant_id，业务读写全部按租户过滤
-- （handlers/vendor/service.go）。约束域全局、使用域按租户，后果与
-- users.username(20261006)、tickets.ticket_number(20261013) 同一缺陷类：
-- MSP 场景下新租户无法使用已被其他租户占用的供应商编码/合同编号。
-- 这里放宽为按租户组合唯一，与 AGENTS.md「业务唯一键默认按 tenant 组合唯一」一致。
--
-- 执行顺序说明：bootstrap 先跑 ent 的 client.Schema.Create（additive，按新
-- schema 建好 vendor_tenant_id_code / contract_tenant_id_contract_number，
-- 但不会删旧索引），再跑本文件。本文件职责是删掉遗留全局唯一索引；
-- CREATE 语句只对「跳过 auto-migrate、只跑版本化迁移」的库生效，全部幂等。
--
-- 存量检查（2026-11-13 生产实测）：vendors / contracts 均 0 行、0 重复组，
-- 无需数据修复。守卫仍保留：任何环境出现同租户重复都会中止迁移，
-- 而不是静默跳过或建出半个索引。

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT tenant_id, code
        FROM vendors
        GROUP BY tenant_id, code
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'vendors has % duplicated (tenant_id, code) group(s); deduplicate before adding the unique index',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT tenant_id, contract_number
        FROM contracts
        GROUP BY tenant_id, contract_number
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'contracts has % duplicated (tenant_id, contract_number) group(s); deduplicate before adding the unique index',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

-- 旧全局唯一对象实测为独立索引；先按约束删（独立索引时 no-op），再按索引删，
-- 两种历史形态都能清除，不会静默残留。
ALTER TABLE vendors DROP CONSTRAINT IF EXISTS vendors_code_key;
DROP INDEX IF EXISTS vendors_code_key;
ALTER TABLE contracts DROP CONSTRAINT IF EXISTS contracts_contract_number_key;
DROP INDEX IF EXISTS contracts_contract_number_key;

CREATE UNIQUE INDEX IF NOT EXISTS vendor_tenant_id_code
    ON vendors (tenant_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS contract_tenant_id_contract_number
    ON contracts (tenant_id, contract_number);