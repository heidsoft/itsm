-- 20261113_tenant_scope_vendor_contract_unique_down.sql
--
-- 回滚 20261113_tenant_scope_vendor_contract_unique.sql：恢复 vendors.code /
-- contracts.contract_number 全局唯一。实测生产原形态是独立唯一索引
-- vendors_code_key / contracts_contract_number_key，回滚按同一形态恢复。
--
-- 这条回滚是「有前提的」：放宽之后，多租户库里合法存在多个租户持有同一个
-- 供应商编码或合同编号，直接重建全局唯一索引必然失败。回滚前必须先人工消解
-- 跨租户重复，本文件只做检测并拒绝，不自动改数据。前提不成立时正确处置是
-- 前滚修复，不是回滚。

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT code
        FROM vendors
        GROUP BY code
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'vendors has % globally duplicated code(s) across tenants; resolve them before restoring the global unique constraint',
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
        SELECT contract_number
        FROM contracts
        GROUP BY contract_number
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'contracts has % globally duplicated contract_number(s) across tenants; resolve them before restoring the global unique constraint',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

DROP INDEX IF EXISTS vendor_tenant_id_code;
DROP INDEX IF EXISTS contract_tenant_id_contract_number;

CREATE UNIQUE INDEX IF NOT EXISTS vendors_code_key ON vendors (code);
CREATE UNIQUE INDEX IF NOT EXISTS contracts_contract_number_key ON contracts (contract_number);