-- 回滚 20261119_ci_owner_fk_and_certification.sql
-- 注意：回滚只移除新增列与索引，owned_by 原文本在 up 迁移中从未删除，故数据不丢失。
-- ci_certification_task 表由 ent schema 管理，本 down 不删表（避免与 schema 生命周期冲突）。

DROP INDEX IF EXISTS ci_last_confirmed_idx;
DROP INDEX IF EXISTS ci_tenant_department_idx;
DROP INDEX IF EXISTS ci_tenant_owner_idx;

ALTER TABLE configuration_items
  DROP COLUMN IF EXISTS confirmed_by,
  DROP COLUMN IF EXISTS last_confirmed_at,
  DROP COLUMN IF EXISTS department_id,
  DROP COLUMN IF EXISTS owner_id;
