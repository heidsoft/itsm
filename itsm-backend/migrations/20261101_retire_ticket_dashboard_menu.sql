-- 20261101_retire_ticket_dashboard_menu
--
-- 停用「工单仪表盘」(/tickets/dashboard) 菜单：它与「工单统计」(/tickets/analytics) 功能重复，
-- 页面已删除，菜单继续挂着等于给使用者留一个 404 入口。
--
-- 依据：
--   1. pkg/menubaseline/baseline.go 已移除该条目，新装与逐租户 provision 不再播种这行；
--      基线补齐 (service/menu_service.go InitMenusFromBaseline) 是 add-only，
--      不会删除存量行，因此存量库必须靠本迁移收口。
--   2. 停用不删（项目存量数据修复口径）：菜单行是运营可编辑对象（is_visible/sort_order
--      由管理员改过），删行会让「菜单管理」页面失去这条历史记录，也会让回滚只能重建半行。
--      service/menu_service.go GetUserMenus 按 is_enabled + is_visible 过滤，
--      置 is_enabled=false 即可让侧边栏不再出现该入口。
--   3. 不按主键 id、不固定 tenant_id（20260830_ticket_types_menu_reparenting.sql 的教训：
--      自增主键跨安装不稳定，硬编码租户违反租户隔离约束）。这里只按封闭的 path 清单匹配，
--      一次覆盖全部租户。
--
-- 幂等：只翻 is_enabled=true 的行，重复执行不产生新变化。
-- 已知边界：迁移在 seed 之前执行，全新库此时 menus 为空表，本语句影响 0 行——
-- 新装的正确形态由基线保证，不依赖本迁移。

UPDATE menus
SET is_enabled = false
WHERE path = '/tickets/dashboard'
  AND is_enabled = true;
