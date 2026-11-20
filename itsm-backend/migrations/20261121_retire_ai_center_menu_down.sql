-- 回滚 20261121_retire_ai_center_menu：恢复「AI 中心」一级菜单行。
-- 注意：基线已删除该条目，回滚菜单后新装不会重新播种；完整回滚需同时还原
-- pkg/menubaseline/baseline.go 的基线条目与三个子菜单的 ParentPath。

UPDATE menus
SET is_enabled = true
WHERE path = '/ai/chat'
  AND is_enabled = false;
