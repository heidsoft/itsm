-- 回滚：移除 dry-run 的查询索引与列注释（不删列）。
--
-- 为什么不 DROP COLUMN dry_run：
--   该列由 ent schema 声明（ent/schema/tool_invocation.go），Schema.Create 会
--   在任何开启 ITSM_AUTO_MIGRATE 的实例上重新建它。手工 DROP 只会造成
--   「迁移回滚了、下次 bootstrap 又建回来」的空转，且中间窗口内的写入会失败。
--   回滚本批的**能力**应当通过下线端点完成（代码层），不是删数据列。
--
-- 因此本 down 只撤销「为读面性能新增的索引」：
--   去掉索引后功能完全不变，只是审批待办查询退化为租户内扫描。
--   数据与语义零损失，属安全降级。

DROP INDEX IF EXISTS tool_invocations_tenant_queue_idx;

COMMENT ON COLUMN tool_invocations.dry_run IS NULL;
