-- CMDB 行级权限载体 + 数据认证字段迁移
-- Date: 2026-11-19
-- Description: 给 configuration_items 加 owner_id/department_id（user FK 载体）与
--   last_confirmed_at/confirmed_by（数据认证冗余），并把旧 owned_by 自由文本按用户名回填。
--
-- 背景：
--   1. 行级权限（DataScope）需要资产上有 owner_id/department_id 作为判定载体。
--      旧 owned_by 是自由文本（可空 string），无法与 user 表关联，做不了行级过滤。
--   2. ServiceNow CMDB Health 的 Compliance 维度要求记录「谁在何时确认该 CI 仍准确」，
--      CI 上冗余 last_confirmed_at/confirmed_by 供列表/治理看板排序，明细在
--      ci_certification_task 表（由 ent schema create 建表，本迁移只管数据列）。
--
-- 数据策略（幂等、保留无法映射数据）：
--   - owner_id 回填只匹配「同租户 + 用户名/姓名精确匹配」的用户，避免跨租户串号；
--   - department_id 取回填命中的用户的 department_id；
--   - 无法匹配的 owned_by 原文本**保留不删**，待人工核对后补录，不静默丢弃；
--   - 所有 ALTER 用 IF NOT EXISTS，UPDATE 用 WHERE owner_id IS NULL，可重复执行。

ALTER TABLE configuration_items
  ADD COLUMN IF NOT EXISTS owner_id INTEGER,
  ADD COLUMN IF NOT EXISTS department_id INTEGER,
  ADD COLUMN IF NOT EXISTS last_confirmed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS confirmed_by INTEGER;

COMMENT ON COLUMN configuration_items.owner_id IS '拥有者用户ID（行级权限判定用）';
COMMENT ON COLUMN configuration_items.department_id IS '所属部门ID（DataScopeDepartment 过滤用）';
COMMENT ON COLUMN configuration_items.last_confirmed_at IS '最近一次人工认证时间';
COMMENT ON COLUMN configuration_items.confirmed_by IS '最近一次认证人用户ID';
-- owned_by 降级为纯展示字段：行级权限一律读 owner_id。保留文本是为了
-- 迁移窗口内旧客户端仍能显示负责人姓名，以及无法回填的历史数据不丢失。
COMMENT ON COLUMN configuration_items.owned_by IS '拥有者显示名（历史字段，仅展示；权限判定一律用 owner_id）';

-- 回填 owner_id，分两段（都是为了幂等与避坑，别合并成一条）：
--   第一段按 username 精确匹配——username 在租户内唯一，不会误配；
--   第二段按 name 兜底——只处理第一段没回填上的行。
--
-- 为什么不能写成 `owned_by = COALESCE(NULLIF(username,''), NULLIF(name,''))`：
-- COALESCE 只会取**第一个非空值**。若某用户 username='zhangsan' 而 name='张三'，
-- 存量 owned_by 里写的偏偏是「张三」（本批新写路径 resolveOwner 正是优先写 name），
-- 那么该行会静默匹配不上、owner_id 永远为 NULL——而外面看不出任何异常，
-- 只是这条资产悄悄没有负责人，行级权限下无法被认领。
UPDATE configuration_items ci
SET owner_id = u.id
FROM users u
WHERE ci.owner_id IS NULL
  AND ci.owned_by IS NOT NULL
  AND ci.tenant_id = u.tenant_id
  AND ci.owned_by = NULLIF(u.username, '');

-- 第二段按 name 兜底，但**只在租户内该姓名唯一时**才回填。
-- name 不是唯一键：同名两人的租户里，UPDATE ... FROM 多行命中时 PostgreSQL 会
-- 任意挑一行，等于把资产随机挂到某人名下——这种「看起来成功」的错误比留 NULL 更坏，
-- 因为 NULL 至少能在「未归属资产」清单里被人发现。歧义的留 NULL 待人工确认。
UPDATE configuration_items ci
SET owner_id = u.id
FROM users u
WHERE ci.owner_id IS NULL
  AND ci.owned_by IS NOT NULL
  AND ci.tenant_id = u.tenant_id
  AND ci.owned_by = NULLIF(u.name, '')
  AND u.id = (
    SELECT MIN(u2.id) FROM users u2
    WHERE u2.tenant_id = u.tenant_id AND u2.name = u.name
    HAVING COUNT(*) = 1
  );

-- 回填 department_id：取刚回填的 owner 所在部门。
UPDATE configuration_items ci
SET department_id = u.department_id
FROM users u
WHERE ci.department_id IS NULL
  AND ci.owner_id = u.id
  AND ci.tenant_id = u.tenant_id;

-- 索引：行级权限过滤与认证新鲜度排序都走这两列。
CREATE INDEX IF NOT EXISTS ci_tenant_owner_idx ON configuration_items (tenant_id, owner_id);
CREATE INDEX IF NOT EXISTS ci_tenant_department_idx ON configuration_items (tenant_id, department_id);
CREATE INDEX IF NOT EXISTS ci_last_confirmed_idx ON configuration_items (tenant_id, last_confirmed_at);
