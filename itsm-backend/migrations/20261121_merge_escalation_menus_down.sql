-- 回滚 20261121_merge_escalation_menus：恢复「升级矩阵」菜单行。
-- 注意：页面本身已删除，回滚菜单后该入口指向 404；完整回滚需同时还原前端页面与基线条目。

UPDATE menus
SET is_enabled = true
WHERE path = '/admin/escalation-matrices'
  AND is_enabled = false;
