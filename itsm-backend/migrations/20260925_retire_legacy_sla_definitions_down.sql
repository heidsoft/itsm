-- 20260925_retire_legacy_sla_definitions_down.sql
--
-- 回滚：把封闭清单内的旧代 SLA 定义重新置 is_active=true。
--
-- 已知边界：无法区分「被本迁移停用」与「运维此前自行停用」的行，
-- 回滚会把同名清单行全部重新激活；如需精确回滚，请以执行前的备份为准。

UPDATE sla_definitions
SET is_active = true,
    updated_at = now()
WHERE is_active = false
  AND name IN (
    'SLA-P0-紧急',
    'SLA-P1-高',
    'SLA-P2-中',
    'SLA-P3-低',
    'SLA-服务请求',
    'SLA-变更',
    '[Template] 事件 P1 紧急',
    '[Template] 事件 P2 高',
    '[Template] 事件 P3 中',
    '[Template] 变更 - 普通',
    '[Template] 变更 - 紧急'
  );
