-- 20261101_retire_ticket_dashboard_menu_down
--
-- 回滚：重新启用 /tickets/dashboard 菜单行。
--
-- 已知边界：无法区分「被本次迁移停用」与「运维此前自行停用」的同名行，
-- 回滚会把该 path 下 is_enabled=false 的行全部重新激活。
-- 另外页面文件已删除，仅恢复菜单会让入口指向 404：
-- next.config.ts 的 /tickets/dashboard → /tickets/analytics redirect 仍在，
-- 点击会跳到工单统计页；真正要回到退役前形态需同时恢复页面文件与该基线条目。

UPDATE menus
SET is_enabled = true
WHERE path = '/tickets/dashboard'
  AND is_enabled = false;
