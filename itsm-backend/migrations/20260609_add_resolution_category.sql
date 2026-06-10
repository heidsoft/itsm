-- 添加 resolution_category 字段到 tickets 表
-- 用于工单解决时记录解决方案分类（代码缺陷/环境问题/用户操作/其他）

ALTER TABLE tickets
ADD COLUMN IF NOT EXISTS resolution_category VARCHAR(50);

-- 字段说明：
-- - 解决工单时由处理人填写
-- - 取值参考：code_defect | environment | user_operation | config_change | other
COMMENT ON COLUMN tickets.resolution_category IS '解决分类：code_defect/environment/user_operation/config_change/other';

-- 添加 closed_at 字段（关闭工单时记录关闭时间）
ALTER TABLE tickets
ADD COLUMN IF NOT EXISTS closed_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN tickets.closed_at IS '工单关闭时间';
