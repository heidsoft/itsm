-- 20261121_retire_ai_center_menu
--
-- 方案 A：取消一级导航「AI 中心」(/ai/chat)，问答能力只保留顶栏全局「AI 助手」入口。
-- 三个真实 AI 能力页改挂 /tickets、/approvals、/audit-logs（基线已改），权限码 ai:read
-- 与页面路由均不变。
--
-- 依据（沿用 20261101_retire_ticket_dashboard_menu / 20261121_merge_escalation_menus 口径）：
--   1. 基线是 add-only，不会删除存量行，必须靠本迁移收口；
--   2. 停用不删：菜单行是运营可编辑对象，保留历史记录，回滚只需翻回开关；
--   3. 只按 path 封闭匹配，一次覆盖全部租户，不碰自增主键；
--   4. 三条子菜单的更名与重挂父级不写在本迁移：启动期 InitMenusFromBaseline 的
--      upsertBaselineMenu 按 (tenantID, path) 更新 Name 与 SetParentID，三个业务父级
--      路径均已在基线存在，无需手写父级字段。
--
-- 幂等：只翻 is_enabled=true 的行，重复执行无新变化。
-- 已知边界：迁移先于 seed 执行时 menus 为空表，影响 0 行——新装形态由基线保证。

UPDATE menus
SET is_enabled = false
WHERE path = '/ai/chat'
  AND is_enabled = true;
