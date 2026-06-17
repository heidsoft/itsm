-- 为 roles 表添加 is_active 字段
-- 迁移版本: 20260617_role_is_active
-- 描述: 支持前端 inactiveRoles 统计，允许角色启用/禁用状态管理

ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN roles.is_active IS '角色是否启用，默认为 true（已启用）';
