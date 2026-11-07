-- 20261107_discovery_scope_states_down.sql
--
-- 回滚 20261107_discovery_scope_states.sql。
--
-- discovery_scope_states 里的水位线都是可重建的派生状态（下次发现会重新开窗
-- 全量列举），删除不会丢失权威业务状态；存量 cloud_resources 不受影响。
--
-- 幂等：IF EXISTS 全覆盖，重复执行安全。

DROP TABLE IF EXISTS discovery_scope_states;
