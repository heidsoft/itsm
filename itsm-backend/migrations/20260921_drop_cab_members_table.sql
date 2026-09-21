-- 清理遗留 cab_members 表
-- CAB 已重命名为 change_review（评审组），数据迁移到 change_review_members 表
-- cab_members 表从未被新代码写入，可安全删除

BEGIN;

-- 删除遗留表
DROP TABLE IF EXISTS cab_members;

COMMIT;
