-- 回填存量客户租户的 MSP 服务商归属（tenants.msp_provider_id）。
--
-- 背景（组织与权限域审计 P0-3 修复的前置数据收口）：
--   分配写路径新增「服务商↔客户委托关系必须真实存在」的校验，判据是
--   客户的 msp_provider_id 必须指向目标 MSP 员工所属的服务商租户。
--   但 msp_provider_id 此前只有租户管理 API（tenants 的 mspProviderId 字段）
--   一条写入路径，且前端无对应设置入口；存量部署的客户租户该字段普遍为 NULL。
--   修复落地后，MSP 分配写入与客户目录读取都会以「委托关系缺失」拒绝，
--   存量 MSP 租户将全线不可用——这是安全默认（fail-closed），但需要数据兜底。
--
-- 回填判据（只用既有事实，不新建授权）：
--   从已存在的 msp_allocations 活跃分配（deassigned_at IS NULL）反推：
--   分配行两端分别是「MSP 员工」与「客户租户」，MSP 员工归属的服务商租户
--   即该客户的 provider。只有当某客户租户的全部活跃分配都指向同一个
--   服务商租户时，该归属才是无歧义的，才回填。
--
-- 歧义与无据的处理：
--   - 同一客户被多个服务商分配（历史越权或客户改换服务商）→ 不回填，
--     保留 NULL，交由人工核对后经租户管理 API 显式设定；
--   - 没有任何活跃分配 → 无据可依，不回填；
--   - 分配两端租户缺失（员工已删/客户已删）→ 该行不参与判定。
--
-- 幂等性：
--   - 只更新 msp_provider_id IS NULL 的行，重复执行不改已回填的行；
--   - 目标值唯一确定（单一服务商，GROUP BY customer_tenant_id HAVING
--     COUNT(DISTINCT provider) = 1）；
--   - 可重复执行。
--
-- 不回填的租户类型：provider 必须是 msp_provider / msp 类型（历史兼容值），
-- 否则说明分配两端类型判断有误，宁可不回填也不写错；客户侧同理限定
-- customer / msp_customer / saas_customer。
--
-- 语法说明：写成相关子查询而非 UPDATE...FROM——后者是 PostgreSQL 扩展语法，
-- SQLite 不支持（迁移在 sqlite 上做行为验证，语法必须两库通用）。

BEGIN;

UPDATE tenants
SET msp_provider_id = (
    SELECT MIN(u.tenant_id)
    FROM msp_allocations a
    JOIN users u
      ON u.id = a.msp_user_id
    JOIN tenants p
      ON p.id = u.tenant_id
     AND p.type IN ('msp_provider', 'msp')
    WHERE a.customer_tenant_id = tenants.id
      AND a.deassigned_at IS NULL
    HAVING COUNT(DISTINCT u.tenant_id) = 1
)
WHERE msp_provider_id IS NULL
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
  );

COMMIT;
