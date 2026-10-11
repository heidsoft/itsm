-- 20261011_add_servicerequestapproval_version.sql
--
-- service_request_approvals 加乐观锁版本列：会签投票是对 node JSON 的整块读改写，
-- 两个并发审批读同一份 approved_ids 后先后覆盖，先提交者的票被静默丢弃，
-- quorum 永不达标。写入侧以 version 作条件更新谓词（handlers/service_request/repository_impl.go），
-- 冲突方重读合并重试。
--
-- 带默认值回填存量行为 1，幂等可重复执行。
-- 执行顺序说明：bootstrap 先跑 ent 的 client.Schema.Create（它会按新 schema 建好 version 列），
-- 再跑本文件；本文件只对「跳过 auto-migrate、只跑版本化迁移」的库生效。

ALTER TABLE service_request_approvals
    ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
