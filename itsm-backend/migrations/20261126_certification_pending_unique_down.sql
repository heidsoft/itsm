-- 回滚 20261126_certification_pending_unique.sql
-- Date: 2026-11-26
-- Description: 仅移除 pending 唯一约束，保留所有任务与人工修复结果。
DROP INDEX IF EXISTS ci_cert_pending_unique;
