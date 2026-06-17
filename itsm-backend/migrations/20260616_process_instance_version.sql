-- 为 process_instances 表添加乐观锁版本号字段
ALTER TABLE process_instances ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
COMMENT ON COLUMN process_instances.version IS '乐观锁版本号';
