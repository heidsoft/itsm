-- 回滚 20261126 的 MSP 服务商归属回填。
--
-- 精确清空本次回填写入的 tenants.msp_provider_id，不删除任何租户行。
--
-- 局限（与 SQL 能力的边界一致，如实写明）：
--   本 down 无法区分「本次回填写入的行」与「升级前就由租户管理 API
--   显式设定过的行」——回填前 msp_provider_id IS NULL 的行本次被写入，
--   回填前已有值的行本次未被触碰。SQL 层没有「本次执行写入集合」的痕迹，
--   只能按「值等于从活跃分配反推出的唯一服务商」这一条件来定位，
--   因此管理员在升级后经租户管理 API 手动改过 provider 的租户，
--   若改后的值恰好等于反推值，回滚会一并清空。
--
--   与 20261125 sysadmin 补偿迁移的 down 同类取舍：优先保证回滚可执行，
--   遗留由本文件顶部的说明承担。
--
-- 语法说明：写成相关子查询而非 UPDATE...FROM——后者是 PostgreSQL 扩展语法，
-- SQLite 不支持（迁移在 sqlite 上做行为验证，语法必须两库通用）。

BEGIN;

UPDATE tenants
SET msp_provider_id = NULL
WHERE msp_provider_id IS NOT NULL
  AND type IN ('customer', 'msp_customer', 'saas_customer')
  AND EXISTS (
      SELECT 1
      FROM msp_allocations a
      JOIN users u
        ON u.id = a.msp_user_id
      JOIN tenants p
        ON p.id = u.tenant_id
       AND p.type IN ('msp_provider', 'msp')
      WHERE a.customer_tenant_id = tenants.id
        AND a.deassigned_at IS NULL
      GROUP BY a.customer_tenant_id
      HAVING COUNT(DISTINCT u.tenant_id) = 1
         AND MIN(u.tenant_id) = tenants.msp_provider_id
  );

COMMIT;
