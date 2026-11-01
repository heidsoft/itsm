-- 20261101_cloud_signal_tables.sql
--
-- 云信号（CloudMonitor 告警 / 云安全中心漏洞）的持久化与同步状态。
--
-- 背景：此前云信号只有两条断头路——
--   1. POST /api/v1/alerts/sources/:source/ingest 只把 Webhook 告警写进 alerts，
--      不建单、不通知、也没有任何读取接口；
--   2. 云安全中心漏洞完全未接入，ECS 详情只到发现层，资产风险不可见。
-- 本迁移只解决存储：告警继续复用 alerts（已有租户隔离与唯一去重约束），
-- 新增 vulnerabilities 与 sync state 两张表，供 worker 定时轮询写入。
--
-- 为什么漏洞单独建表：漏洞的生命周期是「发现 → 修复中 → 已修复/已忽略」，
-- 与告警的「firing → acknowledged → resolved」不是同一套状态机；塞进 alerts
-- 会让两类数据共享严重度与确认语义，报表也无法区分「正在告警」与「待修漏洞」。
--
-- 租户隔离口径：两张表都带 tenant_id，查询一律 tenant_id 前置；RLS 按既有
-- tenant_isolation_<table> 姿态补齐（ENABLE 非 FORCE，helper 缺失即失败）。
--
-- 幂等：IF NOT EXISTS / DROP POLICY IF EXISTS，可重复执行。

-- ============================================================================
-- 0. 前置守卫：helper 由权威迁移流（009/019）声明，这里不重复定义。
-- ============================================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'get_current_tenant_id') THEN
        RAISE EXCEPTION 'missing helper get_current_tenant_id(): replay the canonical migration stream before this file';
    END IF;
END $$;

-- ============================================================================
-- 1. security_findings —— 云安全中心漏洞快照
-- ============================================================================
CREATE TABLE IF NOT EXISTS security_findings (
    id               BIGSERIAL PRIMARY KEY,
    tenant_id        BIGINT NOT NULL,
    cloud_account_id BIGINT NOT NULL REFERENCES cloud_accounts (id) ON DELETE CASCADE,
    provider         VARCHAR(32) NOT NULL DEFAULT '',
    vuln_type        VARCHAR(16) NOT NULL,                        -- cve / sys / cms / app / emg
    finding_key      VARCHAR(128) NOT NULL,                       -- 云厂商漏洞主键（SAS PrimaryId）
    name             VARCHAR(512) NOT NULL DEFAULT '',
    alias_name       VARCHAR(512) NOT NULL DEFAULT '',
    severity         VARCHAR(16) NOT NULL DEFAULT 'unknown',      -- critical / high / medium / low / unknown
    status           VARCHAR(16) NOT NULL DEFAULT 'open',         -- open / fixing / fixed / ignored
    necessity        VARCHAR(16) NOT NULL DEFAULT '',
    instance_id      VARCHAR(128) NOT NULL DEFAULT '',
    instance_name    VARCHAR(255) NOT NULL DEFAULT '',
    uuid             VARCHAR(128) NOT NULL DEFAULT '',
    internet_ip      VARCHAR(64) NOT NULL DEFAULT '',
    intranet_ip      VARCHAR(64) NOT NULL DEFAULT '',
    region           VARCHAR(64) NOT NULL DEFAULT '',
    ci_id            BIGINT,                                       -- 关联 configuration_items.id；CI 删除后保留历史
    cvss_score       NUMERIC(4, 1),
    raw_payload      JSONB,
    first_seen_at    TIMESTAMPTZ,
    last_seen_at     TIMESTAMPTZ,
    last_synced_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, cloud_account_id, vuln_type, finding_key)
);

-- 列表页默认按「未修复 + 严重度」筛选
CREATE INDEX IF NOT EXISTS security_findings_status_severity_idx
    ON security_findings (tenant_id, status, severity);

CREATE INDEX IF NOT EXISTS security_findings_account_idx
    ON security_findings (tenant_id, cloud_account_id, vuln_type);

CREATE INDEX IF NOT EXISTS security_findings_ci_idx
    ON security_findings (tenant_id, ci_id)
    WHERE ci_id IS NOT NULL;

-- ============================================================================
-- 2. cloud_signal_sync_states —— 每个账号每种信号的同步游标与健康状态
--
-- state 语义与「能力状态」口径一致：
--   unconfigured = 账号未配置凭据，能力未启用；
--   unavailable  = 凭据/接口不可用，当前无法同步；
--   ready        = 最近一次同步成功。
-- 不允许用「0 条结果」冒充 ready——空结果同样要能区分「真的没有」与「没在跑」。
-- ============================================================================
CREATE TABLE IF NOT EXISTS cloud_signal_sync_states (
    id                   BIGSERIAL PRIMARY KEY,
    tenant_id            BIGINT NOT NULL,
    cloud_account_id     BIGINT NOT NULL REFERENCES cloud_accounts (id) ON DELETE CASCADE,
    signal_kind          VARCHAR(16) NOT NULL,                    -- alarm / vulnerability
    state                VARCHAR(16) NOT NULL DEFAULT 'unconfigured',
    last_attempt_at      TIMESTAMPTZ,
    last_success_at      TIMESTAMPTZ,
    watermark            TIMESTAMPTZ,                             -- 下次增量同步的起点
    last_error_code      VARCHAR(128) NOT NULL DEFAULT '',
    last_error_message   TEXT NOT NULL DEFAULT '',
    items_synced         BIGINT NOT NULL DEFAULT 0,
    consecutive_failures INT NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, cloud_account_id, signal_kind)
);

CREATE INDEX IF NOT EXISTS cloud_signal_sync_states_kind_idx
    ON cloud_signal_sync_states (tenant_id, signal_kind, last_attempt_at);

-- ============================================================================
-- 3. RLS：与 21 张既有业务表同姿态（ENABLE 非 FORCE，owner/superuser 仍可运维）
-- ============================================================================
ALTER TABLE security_findings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_security_findings ON security_findings;
CREATE POLICY tenant_isolation_security_findings ON security_findings
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE cloud_signal_sync_states ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_cloud_signal_sync_states ON cloud_signal_sync_states;
CREATE POLICY tenant_isolation_cloud_signal_sync_states ON cloud_signal_sync_states
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- ============================================================================
-- 4. 告警建单桥的扫描索引：只覆盖 CloudMonitor 来源，避免拖慢通用告警查询
-- ============================================================================
CREATE INDEX IF NOT EXISTS alerts_cloudmonitor_bridge_idx
    ON alerts (tenant_id, fired_at DESC)
    WHERE source = 'cloudmonitor';

-- 建单桥的幂等反查（崩溃恢复）：按 metadata.cloudAlertId 找已创建的事件。
-- 部分索引只覆盖带该键的行，不影响 incidents 的常规查询。
CREATE INDEX IF NOT EXISTS incidents_cloud_alert_idx
    ON incidents (tenant_id, (metadata->>'cloudAlertId'))
    WHERE metadata ? 'cloudAlertId';
