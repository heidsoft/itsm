-- 20261101_cloud_signal_tables_down.sql
--
-- 回滚 20261101_cloud_signal_tables.sql。
--
-- 只删本迁移新建的对象；alerts 表本身属于既有迁移，这里仅移除本迁移
-- 额外加上的桥接扫描索引。security_findings 里的漏洞快照与同步游标都是
-- 可重建的派生数据（下次轮询会重新拉取），删除不会丢失权威业务状态。
--
-- 幂等：IF EXISTS 全覆盖，重复执行安全。

DROP INDEX IF EXISTS incidents_cloud_alert_idx;
DROP INDEX IF EXISTS alerts_cloudmonitor_bridge_idx;

DROP TABLE IF EXISTS cloud_signal_sync_states;
DROP TABLE IF EXISTS security_findings;
