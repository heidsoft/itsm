-- 20260611_enable_preset_ci_types.sql
-- 修复历史种子缺陷：CITypeSeed.IsActive 曾为 bool 零值，配置省略 is_active 时
-- 8 个预置 CI 类型被隐式种为禁用。预置类型应开箱可用，统一回填为启用。
-- 仅针对默认租户下的预置类型名单，不影响管理员显式禁用的自建类型。

UPDATE ci_types
SET is_active = TRUE
WHERE is_active = FALSE
  AND name IN ('server', 'database', 'network', 'storage',
               'application', 'middleware', 'cloud_vm', 'kubernetes')
  AND tenant_id = (SELECT id FROM tenants WHERE code = 'default');
