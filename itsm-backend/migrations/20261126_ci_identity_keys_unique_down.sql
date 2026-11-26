-- CMDB CI 身份键部分唯一索引回滚
-- Date: 2026-11-26
-- Author: Codex（阻塞性预检，不是自动去重）
-- Description: 移除本迁移新增的两个索引；serial_number 既有索引保留。
-- 数据策略：不修改 CI 数据；回滚后不再防止这些身份键的并发重复写入。
BEGIN;
DROP INDEX IF EXISTS ci_asset_tag_tenant_unique;
DROP INDEX IF EXISTS ci_cloud_identity_unique;
COMMIT;
