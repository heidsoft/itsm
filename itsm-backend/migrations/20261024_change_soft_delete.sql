-- 20261024_change_soft_delete.sql
--
-- 变更单转软删：
--   - changes 增补 deleted_at。此前 change 是全域唯一没有软删字段的业务实体，
--     删除走物理删，后果有两条：①PIR / 评审成员 / 回滚步骤 / 审批链全部成为
--     孤儿（无级联、无清理任务，外键之外的从表行留在库里永远查不到父记录）；
--     ②已用过的变更编号可被复用，审计上「CHG-20261024-0031 被删过一次又重新
--     出现」无法解释。
--   - 写路径：DELETE /api/v1/changes/:id 改为写 deleted_at（handlers/change/
--     repository_impl.go）。
--   - 读路径：database.RegisterSoftDeleteInterceptors 对 ChangeQuery 附加
--     DeletedAtIsNil()，列表/详情/统计与 WithXxx 关联查询统一屏蔽已删记录，
--     与 ticket / incident / problem / service_request 四域口径一致。
--
-- 存量数据无需回填：deleted_at IS NULL 即未删除。已物理删掉的行无法恢复，
-- 本迁移对它们无能为力——这正是要改成软删的原因。
--
-- 本文件面向「跳过 auto-migrate、只跑版本化迁移」的库，幂等可重复执行。

ALTER TABLE changes
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN changes.deleted_at IS '删除时间（软删标记，NULL 表示未删除）';

-- 读路径每条查询都带 deleted_at IS NULL，部分索引只收未删行，体积远小于全表
-- 索引，且不会被墓碑数据拖大。
CREATE INDEX IF NOT EXISTS idx_changes_deleted_at
    ON changes (deleted_at)
    WHERE deleted_at IS NULL;
