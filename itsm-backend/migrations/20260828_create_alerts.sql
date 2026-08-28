CREATE TABLE IF NOT EXISTS alerts (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    source VARCHAR(100) NOT NULL,
    external_alert_id VARCHAR(255) NOT NULL,
    source_raw VARCHAR(100) NOT NULL DEFAULT '',
    name VARCHAR(500) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    severity VARCHAR(20) NOT NULL,
    status VARCHAR(40) NOT NULL,
    labels JSONB NULL,
    annotations JSONB NULL,
    source_ip VARCHAR(255) NOT NULL DEFAULT '',
    service VARCHAR(255) NOT NULL DEFAULT '',
    tags JSONB NULL,
    fired_at TIMESTAMPTZ NOT NULL,
    acknowledged_at TIMESTAMPTZ NULL,
    resolved_at TIMESTAMPTZ NULL,
    raw_payload JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT alerts_tenant_source_external_id_key
        UNIQUE (tenant_id, source, external_alert_id)
);

CREATE INDEX IF NOT EXISTS alerts_tenant_status_fired_at_idx
    ON alerts (tenant_id, status, fired_at);

ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_alerts ON alerts;
CREATE POLICY tenant_isolation_alerts ON alerts
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT);
