-- 20261121_merge_escalation_menus
--
-- 合并 SLA 域两个「升级」菜单为一页双 Tab：
--   /admin/escalation-rules  改名「超时升级」（基线已改，InitMenusFromBaseline 按
--                            (tenantID, path) upsert Name/Icon/SortOrder，存量行自动更名）
--   /admin/escalation-matrices 从基线移除，页面已删除；本迁移停用存量菜单行。
--
-- 依据（沿用 20261101_retire_ticket_dashboard_menu 口径）：
--   1. 基线是 add-only，不会删除存量行，必须靠本迁移收口；
--   2. 停用不删：菜单行是运营可编辑对象，保留历史记录，回滚只需翻回开关；
--   3. 只按 path 封闭匹配，一次覆盖全部租户，不碰自增主键。
--
-- 幂等：只翻 is_enabled=true 的行，重复执行无新变化。
-- 已知边界：迁移先于 seed 执行时 menus 为空表，影响 0 行——新装形态由基线保证。

UPDATE menus
SET is_enabled = false
WHERE path = '/admin/escalation-matrices'
  AND is_enabled = true;
