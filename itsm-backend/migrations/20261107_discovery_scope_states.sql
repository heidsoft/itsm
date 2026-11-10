-- 20261107_discovery_scope_states.sql
--
-- 云发现覆盖水位线（蓝图 §3.2-17）：每个 (source, service_code, region) scope
-- 独立保存 coverage start/end、分页完成证明与 provider watermark。
--
-- 语义：
--   provider_watermark     未完成翻页的 nextToken；窗口封口即清空。
--   coverage_start         当前覆盖窗口起点。断点续扫沿用同一窗口（失败 scope 的
--                          补采不得伪装成新时间快照），缺失判定以它为水位线。
--   pagination_completed   完整覆盖证明；只有翻页自然耗尽才置位，缺失扫描只在
--                          有证明的窗口上执行。
--   last_full_completed_at 水位线：上次完整覆盖完成时刻。
--   consecutive_failures   同一水位线连续失败次数；达上限由执行器重置水位线，
--                          防 provider 游标失效后的无限续扫。
--
-- 时间列存 RFC3339 文本（同 ai_audit_ledger_events 的取舍）：跨 PG/SQLite
-- 逐字节一致，水位线比较在 Go 侧完成。

CREATE TABLE IF NOT EXISTS discovery_scope_states (
    tenant_id              BIGINT NOT NULL,
    source_id              VARCHAR(128) NOT NULL,
    service_code           VARCHAR(64) NOT NULL,
    region                 VARCHAR(64) NOT NULL DEFAULT '',
    provider_watermark     TEXT NOT NULL DEFAULT '',
    coverage_start         TEXT NOT NULL DEFAULT '',
    pagination_completed   BOOLEAN NOT NULL DEFAULT FALSE,
    last_full_completed_at TEXT NOT NULL DEFAULT '',
    consecutive_failures   INT NOT NULL DEFAULT 0,
    last_error_code        VARCHAR(128) NOT NULL DEFAULT '',
    created_at             TEXT NOT NULL DEFAULT '',
    updated_at             TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_id, source_id, service_code, region)
);

-- 与既有业务表同姿态的租户隔离（ENABLE 非 FORCE，owner/superuser 仍可运维）。
ALTER TABLE discovery_scope_states ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_discovery_scope_states ON discovery_scope_states;
CREATE POLICY tenant_isolation_discovery_scope_states ON discovery_scope_states
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
